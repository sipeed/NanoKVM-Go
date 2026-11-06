package application

import (
	"bytes"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

func TestGetLatestFetchesOnlyDiscoveryMetadata(t *testing.T) {
	digest := sha512.Sum512([]byte("archive"))
	rawMetadata, err := json.Marshal(map[string]any{
		"version": " V1.2.4 ",
		"name":    "custom_release-1.2.4.tar.gz",
		"sha512":  base64.StdEncoding.EncodeToString(digest[:]),
		"size":    7,
		"url":     "https://attacker.invalid/override.tar.gz",
		"future":  true,
	})
	if err != nil {
		t.Fatal(err)
	}

	requests := 0
	useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		if request.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", request.Method)
		}
		if request.URL.Path != "/channel/latest.json" {
			t.Errorf("path = %q, want /channel/latest.json", request.URL.Path)
		}
		query := request.URL.Query()
		if query.Get("now") == "" || len(query) != 1 {
			t.Errorf("query = %q, want only cache-busting now", request.URL.RawQuery)
		}
		return latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(rawMetadata))), nil
	}))

	latest, err := getLatestFromBaseURL("https://updates.invalid/channel/")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if requests != 1 {
		t.Fatalf("requests = %d, want exactly one latest.json request", requests)
	}
	if latest.Version != "1.2.4" || latest.Name != "custom_release-1.2.4.tar.gz" ||
		latest.Sha512 != base64.StdEncoding.EncodeToString(digest[:]) || latest.Size != 7 {
		t.Fatalf("unexpected latest metadata: %#v", latest)
	}
	if latest.Url != "https://updates.invalid/channel/custom_release-1.2.4.tar.gz" {
		t.Fatalf("derived URL = %q", latest.Url)
	}
}

func TestGetLatestHTTPSFallback(t *testing.T) {
	body := latestMetadataBytes(
		t,
		"1.2.4",
		"nanokvm-go_1.2.4.tar.gz",
		latestTestDigest(sha512.Size),
		7,
	)

	t.Run("HTTPS success does not request HTTP", func(t *testing.T) {
		var requested []string
		useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.URL.String())
			if request.URL.Scheme != "https" {
				t.Fatalf("request scheme = %q, want https", request.URL.Scheme)
			}
			return latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(body))), nil
		}))

		if _, err := getLatestFromBaseURL("https://updates.invalid:8443/channel"); err != nil {
			t.Fatalf("get latest: %v", err)
		}
		if len(requested) != 1 {
			t.Fatalf("requests = %v, want one HTTPS request", requested)
		}
	})

	t.Run("HTTPS failure retries the same URL over HTTP", func(t *testing.T) {
		var requested []*http.Request
		useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			switch len(requested) {
			case 1:
				return nil, errors.New("TLS certificate is not yet valid")
			case 2:
				return latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(body))), nil
			default:
				t.Fatalf("unexpected extra request: %s", request.URL)
				return nil, errors.New("unexpected extra request")
			}
		}))

		latest, err := getLatestFromBaseURL("https://updates.invalid:8443/channel")
		if err != nil {
			t.Fatalf("get latest through HTTP fallback: %v", err)
		}
		assertHTTPSFallbackRequests(t, requested)
		if latest.Url != "https://updates.invalid:8443/channel/nanokvm-go_1.2.4.tar.gz" {
			t.Fatalf("derived archive URL = %q, want original HTTPS base URL", latest.Url)
		}
	})

	t.Run("HTTPS and HTTP failures return an error", func(t *testing.T) {
		var requested []*http.Request
		useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requested = append(requested, request.Clone(request.Context()))
			return nil, errors.New("network failed")
		}))

		if _, err := getLatestFromBaseURL("https://updates.invalid:8443/channel"); err == nil {
			t.Fatal("HTTPS and HTTP failures were accepted")
		}
		assertHTTPSFallbackRequests(t, requested)
	})
}

func assertHTTPSFallbackRequests(t *testing.T, requested []*http.Request) {
	t.Helper()
	if len(requested) != 2 {
		t.Fatalf("requests = %d, want HTTPS followed by HTTP", len(requested))
	}
	httpsRequest := requested[0].URL
	httpRequest := requested[1].URL
	if httpsRequest.Scheme != "https" || httpRequest.Scheme != "http" {
		t.Fatalf("request schemes = %q, %q; want https, http", httpsRequest.Scheme, httpRequest.Scheme)
	}
	if httpRequest.Host != httpsRequest.Host ||
		httpRequest.EscapedPath() != httpsRequest.EscapedPath() ||
		httpRequest.RawQuery != httpsRequest.RawQuery {
		t.Fatalf("HTTP fallback changed URL beyond scheme: HTTPS=%s HTTP=%s", httpsRequest, httpRequest)
	}
}

func TestGetLatestRejectsInvalidMetadata(t *testing.T) {
	validDigest := latestTestDigest(sha512.Size)
	tooLongName := strings.Repeat("a", 249) + ".tar.gz"
	tests := []struct {
		name string
		body []byte
	}{
		{name: "invalid JSON", body: []byte(`{"version":`)},
		{name: "empty version", body: latestMetadataBytes(t, "", "update.tar.gz", validDigest, 1)},
		{name: "two version components", body: latestMetadataBytes(t, "1.2", "update.tar.gz", validDigest, 1)},
		{name: "four version components", body: latestMetadataBytes(t, "1.2.3.4", "update.tar.gz", validDigest, 1)},
		{name: "leading zero", body: latestMetadataBytes(t, "1.02.3", "update.tar.gz", validDigest, 1)},
		{name: "non decimal version", body: latestMetadataBytes(t, "1.x.3", "update.tar.gz", validDigest, 1)},
		{name: "version overflow", body: latestMetadataBytes(t, "18446744073709551616.2.3", "update.tar.gz", validDigest, 1)},
		{name: "more than one version prefix", body: latestMetadataBytes(t, "vV1.2.3", "update.tar.gz", validDigest, 1)},
		{name: "repeated lowercase version prefix", body: latestMetadataBytes(t, "vv1.2.3", "update.tar.gz", validDigest, 1)},
		{name: "repeated uppercase version prefix", body: latestMetadataBytes(t, "VV1.2.3", "update.tar.gz", validDigest, 1)},
		{name: "empty artifact name", body: latestMetadataBytes(t, "1.2.3", "", validDigest, 1)},
		{name: "path traversal", body: latestMetadataBytes(t, "1.2.3", "../update.tar.gz", validDigest, 1)},
		{name: "path separator", body: latestMetadataBytes(t, "1.2.3", "dir/update.tar.gz", validDigest, 1)},
		{name: "backslash", body: latestMetadataBytes(t, "1.2.3", `dir\update.tar.gz`, validDigest, 1)},
		{name: "NUL", body: latestMetadataBytes(t, "1.2.3", "update\x00.tar.gz", validDigest, 1)},
		{name: "Unicode artifact name", body: latestMetadataBytes(t, "1.2.3", "更新.tar.gz", validDigest, 1)},
		{name: "space in artifact name", body: latestMetadataBytes(t, "1.2.3", "update file.tar.gz", validDigest, 1)},
		{name: "query delimiter in artifact name", body: latestMetadataBytes(t, "1.2.3", "update?.tar.gz", validDigest, 1)},
		{name: "fragment delimiter in artifact name", body: latestMetadataBytes(t, "1.2.3", "update#.tar.gz", validDigest, 1)},
		{name: "percent escape in artifact name", body: latestMetadataBytes(t, "1.2.3", "update%.tar.gz", validDigest, 1)},
		{name: "wrong artifact suffix", body: latestMetadataBytes(t, "1.2.3", "update.tar.xz", validDigest, 1)},
		{name: "artifact name exceeds 255 bytes", body: latestMetadataBytes(t, "1.2.3", tooLongName, validDigest, 1)},
		{name: "zero size", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", validDigest, 0)},
		{name: "negative size", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", validDigest, -1)},
		{name: "malformed digest", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", "not-base64", 1)},
		{name: "short digest", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", latestTestDigest(sha512.Size-1), 1)},
		{name: "long digest", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", latestTestDigest(sha512.Size+1), 1)},
		{name: "non canonical digest", body: latestMetadataBytes(t, "1.2.3", "update.tar.gz", validDigest[:4]+"\n"+validDigest[4:], 1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(test.body))), nil
			}))
			if _, err := getLatestFromBaseURL("https://updates.invalid"); err == nil {
				t.Fatal("invalid latest metadata was accepted")
			}
		})
	}
}

func TestGetLatestMetadataResponseLimit(t *testing.T) {
	valid := latestMetadataBytes(t, "1.2.3", "update.tar.gz", latestTestDigest(sha512.Size), 1)
	if int64(len(valid)) >= maxLatestMetadataBytes {
		t.Fatalf("fixture is unexpectedly large: %d", len(valid))
	}
	exactLimit := append(append([]byte(nil), valid...), bytes.Repeat([]byte(" "), int(maxLatestMetadataBytes)-len(valid))...)

	tests := []struct {
		name    string
		body    []byte
		wantErr bool
	}{
		{name: "exact limit", body: exactLimit},
		{name: "over limit", body: append(append([]byte(nil), exactLimit...), ' '), wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useApplicationHTTPClient(t, roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return latestHTTPResponse(request, http.StatusOK, io.NopCloser(bytes.NewReader(test.body))), nil
			}))
			_, err := getLatestFromBaseURL("https://updates.invalid")
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGetLatestRejectsHTTPAndReadErrors(t *testing.T) {
	tests := []struct {
		name      string
		transport roundTripFunc
	}{
		{
			name: "request error",
			transport: func(*http.Request) (*http.Response, error) {
				return nil, errors.New("network failed")
			},
		},
		{
			name: "non-OK response",
			transport: func(request *http.Request) (*http.Response, error) {
				return latestHTTPResponse(request, http.StatusBadGateway, io.NopCloser(bytes.NewReader(nil))), nil
			},
		},
		{
			name: "response read error",
			transport: func(request *http.Request) (*http.Response, error) {
				return latestHTTPResponse(request, http.StatusOK, failingReadCloser{}), nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			useApplicationHTTPClient(t, test.transport)
			if _, err := getLatestFromBaseURL("https://updates.invalid"); err == nil {
				t.Fatal("fetch failure was accepted")
			}
		})
	}
}

func TestGetLatestRejectsEmptyBaseURL(t *testing.T) {
	if _, err := getLatestFromBaseURL("///"); err == nil {
		t.Fatal("empty application update base URL was accepted")
	}
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReadCloser) Close() error             { return nil }

func useApplicationHTTPClient(t *testing.T, transport roundTripFunc) {
	t.Helper()
	oldClient := applicationHTTPClient
	applicationHTTPClient = &http.Client{Transport: transport}
	t.Cleanup(func() { applicationHTTPClient = oldClient })
}

func latestHTTPResponse(request *http.Request, status int, body io.ReadCloser) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       body,
		Header:     make(http.Header),
		Request:    request,
	}
}

func latestMetadataBytes(t *testing.T, version, name, digest string, size int64) []byte {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"version": version,
		"name":    name,
		"sha512":  digest,
		"size":    size,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func latestTestDigest(size int) string {
	return base64.StdEncoding.EncodeToString(make([]byte, size))
}

type getVersionResponse struct {
	Code int                  `json:"code"`
	Msg  string               `json:"msg"`
	Data *proto.GetVersionRsp `json:"data"`
}

func TestGetVersionReturnsLatestVersion(t *testing.T) {
	setupGetVersionTest(t, "1.2.3", func() (*Latest, error) {
		return &Latest{Version: "v1.2.4"}, nil
	})

	rsp := getVersion(t)
	if rsp.Code != 0 || rsp.Msg != "success" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
	if rsp.Data == nil {
		t.Fatal("response data is nil")
	}
	if rsp.Data.Current != "1.2.3" || rsp.Data.Latest != "1.2.4" {
		t.Fatalf("unexpected version data: %#v", rsp.Data)
	}
}

func TestGetVersionReturnsErrorWhenLatestVersionLookupFails(t *testing.T) {
	setupGetVersionTest(t, "1.2.3", func() (*Latest, error) {
		return nil, errors.New("TLS certificate is not yet valid")
	})

	rsp := getVersion(t)
	if rsp.Code != -1 || rsp.Msg != "failed to get latest version" {
		t.Fatalf("unexpected response: %#v", rsp)
	}
	if rsp.Data != nil {
		t.Fatalf("response data = %#v, want nil", rsp.Data)
	}
}

func setupGetVersionTest(t *testing.T, currentVersion string, getLatest func() (*Latest, error)) {
	t.Helper()

	oldCurrentVersion := currentVersionForVersion
	oldGetLatest := getLatestForVersion
	currentVersionForVersion = func() string {
		return currentVersion
	}
	getLatestForVersion = getLatest

	t.Cleanup(func() {
		currentVersionForVersion = oldCurrentVersion
		getLatestForVersion = oldGetLatest
	})
}

func getVersion(t *testing.T) getVersionResponse {
	t.Helper()

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/application/version", nil)
	NewService().GetVersion(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var rsp getVersionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &rsp); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return rsp
}
