package application

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"NanoKVM-Server/config"
	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const (
	// serverFeatureConfigPath and serverFeatureSyncURLPath compose the only
	// published manifest location, appended to LegacyBaseURL. Both legacy regions
	// use the existing /cn/ URL so devices share one server-controlled switch.
	serverFeatureConfigPath  = "/config"
	serverFeatureSyncURLPath = "features.json"
	// serverFeatureSyncTimeout bounds one fetch so a stalled CDN cannot hold
	// the region mutation lock indefinitely.
	serverFeatureSyncTimeout = 10 * time.Second
	// serverFeatureMaxRedirects is the largest number of hops a manifest
	// source may make before the redirect policy refuses it.
	serverFeatureMaxRedirects = 3
)

var (
	// serverFeatureBaseURL is a test seam over LegacyBaseURL so httptest
	// servers can stand in for the CDN. The manifest stays on the legacy root
	// so pre-v2 and v2 devices read one copy.
	serverFeatureBaseURL = LegacyBaseURL
	// serverFeatureHTTPClient is a test seam over the redirect-restricted
	// client.
	serverFeatureHTTPClient = newServerFeatureHTTPClient()
	// serverFeatureSyncMu serializes a fetch with its compare-and-publish so
	// two region mutations cannot interleave a read and a write.
	serverFeatureSyncMu sync.Mutex
	// serverFeatureSyncEnabled is the seam over the user-owned
	// serverFeatureSync.enabled setting in /etc/kvm/server.yaml. Tests replace
	// it so host runs never touch /etc/kvm through config.GetInstance().
	serverFeatureSyncEnabled = config.GetServerFeatureSyncEnabled
)

// serverFeatureManifestURL uses the existing published source for both regions.
func serverFeatureManifestURL(region proto.ApplicationRegion) (string, error) {
	if !validApplicationRegion(region) {
		return "", fmt.Errorf("no feature manifest source for region %q", region)
	}
	return serverFeatureBaseURL + serverFeatureConfigPath + "/cn/" + serverFeatureSyncURLPath, nil
}

// newServerFeatureHTTPClient returns the bounded client used for manifest
// fetches. Redirects must stay on the original scheme and host (so an
// HTTPS-to-HTTP downgrade or a cross-host hop is refused and surfaces as an
// unrejected 3xx status for the caller to reject), and the hop count is
// capped at serverFeatureMaxRedirects.
func newServerFeatureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: serverFeatureSyncTimeout,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) > serverFeatureMaxRedirects {
				return http.ErrUseLastResponse
			}
			if len(via) == 0 {
				return nil
			}
			origin := via[0].URL
			if request.URL.Scheme != origin.Scheme || request.URL.Host != origin.Host {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// fetchServerFeatureManifest performs one bounded GET. Any status other than
// 200, an oversized or unreadable body, a cross-host final URL, or a body
// close failure reports an error, and a partial payload is never returned
// with a nil error. The body is closed on every path.
func fetchServerFeatureManifest(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build feature manifest request: %w", err)
	}
	request.Header.Set("Accept", "application/json")

	response, err := serverFeatureHTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("request feature manifest: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	if response.Request == nil || response.Request.URL == nil {
		return nil, errors.New("feature manifest response has no final URL")
	}
	if response.Request.URL.Host != request.URL.Host {
		return nil, fmt.Errorf(
			"feature manifest redirected to unexpected host %q",
			response.Request.URL.Host,
		)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("feature manifest returned status %d", response.StatusCode)
	}

	payload, err := io.ReadAll(io.LimitReader(response.Body, serverFeatureMaxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read feature manifest: %w", err)
	}
	if len(payload) > serverFeatureMaxBytes {
		return nil, errors.New("feature manifest exceeds the size limit")
	}
	if err := response.Body.Close(); err != nil {
		return nil, fmt.Errorf("close feature manifest response: %w", err)
	}
	return payload, nil
}

// publish atomically replaces the manifest file with payload. It rejects an
// unset store and an empty or oversized payload before touching the disk.
func (store *serverFeatureStore) publish(payload []byte) error {
	if store == nil || store.path == "" {
		return errors.New("feature manifest path is empty")
	}
	if len(payload) == 0 || len(payload) > serverFeatureMaxBytes {
		return errors.New("feature manifest payload size is invalid")
	}
	return publishAtomicFile(store.path, payload, 0o644)
}

// SyncRegionFeatureManifest fetches, validates, and atomically publishes the
// shared manifest for either legacy region. Invalid regions return nil without
// issuing a request or touching an existing file. The whole
// fetch-compare-publish sequence is serialized and bounded by
// serverFeatureSyncTimeout on top of ctx.
//
// A fetch, validation, or write failure returns an error and leaves the
// existing manifest untouched; enabled:false is a valid manifest and is
// published normally.
//
// The user-owned serverFeatureSync.enabled setting gates the whole sequence:
// when it is false, an existing manifest (valid, invalid, or explicitly
// disabled) is never fetched over or overwritten, and only a missing manifest
// is repaired.
func SyncRegionFeatureManifest(ctx context.Context, region proto.ApplicationRegion) error {
	if !validApplicationRegion(region) {
		return nil
	}

	// The lock is taken before URL construction (which is pure) so the toggle
	// decision below and the fetch it authorizes cannot interleave with another
	// sync.
	serverFeatureSyncMu.Lock()
	defer serverFeatureSyncMu.Unlock()

	// Local-modification mode: with the pull toggle off, an existing manifest is
	// authoritative and must not even trigger a fetch, let alone be overwritten.
	// That covers valid, invalid, and explicitly disabled payloads alike; only a
	// manifest the tri-state reader calls FeatureStateMissing may be repaired.
	// The state read is short-circuited away when the toggle is on so the
	// default path performs no extra I/O and stays byte-for-byte unchanged.
	if !serverFeatureSyncEnabled() && defaultServerFeatureStore.state() != FeatureStateMissing {
		log.Debugf("feature manifest sync disabled and a local manifest exists; skipping")
		return nil
	}

	rawURL, err := serverFeatureManifestURL(region)
	if err != nil {
		return err
	}

	syncContext, cancel := context.WithTimeout(ctx, serverFeatureSyncTimeout)
	defer cancel()

	payload, err := fetchServerFeatureManifest(syncContext, rawURL)
	if err != nil {
		return fmt.Errorf("fetch feature manifest: %w", err)
	}
	if _, ok := classifyServerFeatureManifest(payload); !ok {
		return errors.New("validate feature manifest: invalid manifest")
	}

	// Re-check before publishing: a developer may have written the manifest
	// while the fetch was in flight. The fetch itself cannot be undone, but this
	// stops the sync from clobbering a manifest that appeared mid-sync.
	if !serverFeatureSyncEnabled() && defaultServerFeatureStore.state() != FeatureStateMissing {
		log.Debugf("feature manifest appeared during sync; skipping publish")
		return nil
	}

	store := defaultServerFeatureStore
	if current, ok := store.read(); ok && bytes.Equal(current, payload) {
		log.Debugf("feature manifest already up to date: %s", rawURL)
		return nil
	}
	if err := store.publish(payload); err != nil {
		return fmt.Errorf("publish feature manifest: %w", err)
	}
	log.Debugf("synced feature manifest: %s", rawURL)
	return nil
}

// SyncServerFeatures refreshes the shared manifest when the remote-network
// settings page opens. Wait for publication before allowing subsequent status
// requests to consume the flags. Failed syncs leave the local manifest in use.
func (s *Service) SyncServerFeatures(c *gin.Context) {
	var rsp proto.Response
	if err := s.syncRegionFeatureManifest(c.Request.Context(), s.applicationRegionStore().load()); err != nil {
		log.Warnf("sync feature manifest for remote-network settings failed: %s", err)
		rsp.ErrRsp(c, -1, "sync feature manifest failed")
		return
	}
	rsp.OkRsp(c)
}
