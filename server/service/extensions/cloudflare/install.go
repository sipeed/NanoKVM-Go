package cloudflare

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	CloudflaredVersion       = "2026.10.0"
	serviceName              = "nanokvm-cloudflared.service"
	releaseURL               = "https://github.com/cloudflare/cloudflared/releases/download/" + CloudflaredVersion + "/cloudflared-linux-"
	maxBinaryBytes     int64 = 64 << 20
)

// Digests published with the official 2026.10.0 GitHub release assets.
var releaseChecksums = map[string]string{
	"armhf": "236afbc341625f5af0a8c461577d9da8aa66e3fd123a5b34e801d18a8a5a4d95",
	"arm64": "e6422b9d4f72d3194bc5a38676f13667c06666523217b842a877d72a80b5ac08",
	"amd64": "d33ff2d14475178d2012c2c56beba87389ac5ded27649519f198a7d3134a99db",
}

type filePaths struct {
	binary  string
	token   string
	config  string
	unit    string
	version string
}

var defaultPaths = filePaths{
	binary:  "/usr/local/lib/nanokvm/cloudflare/cloudflared",
	token:   "/etc/kvm/cloudflare/token",
	config:  "/etc/kvm/cloudflare/config.json",
	unit:    "/etc/systemd/system/" + serviceName,
	version: "/usr/local/lib/nanokvm/cloudflare/version",
}

func packageArch() string {
	if runtime.GOARCH == "arm" {
		return "armhf"
	}
	return runtime.GOARCH
}

func (s *Service) installed() bool {
	for _, path := range []string{s.paths.binary, s.paths.unit, s.paths.version} {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || (path == s.paths.binary && info.Mode().Perm()&0111 == 0) {
			return false
		}
	}
	return true
}

func (s *Service) install(ctx context.Context) error {
	if s.installed() {
		return s.system.reload()
	}
	cfg, err := readConfig(s.paths)
	if err != nil {
		return err
	}
	arch := packageArch()
	checksum, ok := releaseChecksums[arch]
	if !ok {
		return errors.New("unsupportedArchitecture")
	}
	dir := filepath.Dir(s.paths.binary)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(dir, ".install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	binary := filepath.Join(staging, "cloudflared")
	if err := downloadBinary(ctx, releaseURL+arch, binary, checksum); err != nil {
		return err
	}
	if err := os.Chmod(binary, 0755); err != nil {
		return err
	}
	if err := os.Rename(binary, s.paths.binary); err != nil {
		return err
	}
	for _, entry := range []struct {
		path string
		data []byte
	}{
		{s.paths.version, []byte(CloudflaredVersion + "\n")},
		{s.paths.unit, []byte(s.unitContents(cfg.Mode))},
	} {
		if err := replaceFile(entry.path, entry.data, 0644); err != nil {
			return err
		}
	}
	// Installing never enables a tunnel. Users configure and explicitly start it.
	return s.system.reload()
}

func downloadBinary(ctx context.Context, url, path, checksum string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = downloadBinaryOnce(ctx, url, path, checksum)
		if err == nil {
			return nil
		}
		var networkErr net.Error
		if attempt == 2 || ctx.Err() != nil || (!errors.As(err, &networkErr) && !errors.Is(err, io.ErrUnexpectedEOF)) {
			return err
		}
		timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

func downloadBinaryOnce(ctx context.Context, url, path, checksum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxBinaryBytes {
		return errors.New("invalidDownload")
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer file.Close()
	complete := false
	defer func() {
		if !complete {
			_ = os.Remove(path)
		}
	}()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, maxBinaryBytes+1))
	if err != nil {
		return err
	}
	if n == 0 || n > maxBinaryBytes || hex.EncodeToString(hash.Sum(nil)) != checksum {
		return errors.New("checksumMismatch")
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

func (s *Service) unitContents(mode string) string {
	condition := "ConditionPathExists=" + s.paths.token + "\n"
	command := fmt.Sprintf("%s tunnel --no-autoupdate --metrics %s run --token-file %s", s.paths.binary, metricsAddress, s.paths.token)
	if mode == modeQuick {
		_, port := s.localEndpoint()
		condition = ""
		// Only the fixed loopback origin uses relaxed TLS verification. Explicit
		// empty configuration keeps unrelated cloudflared settings out of this mode.
		command = fmt.Sprintf("%s tunnel --config /dev/null --no-autoupdate --metrics %s --protocol auto --url https://127.0.0.1:%d --no-tls-verify", s.paths.binary, metricsAddress, port)
	}
	return fmt.Sprintf(`[Unit]
Description=NanoKVM Cloudflare Tunnel
Wants=network-online.target
After=network-online.target nanokvm-go.service
%sStartLimitIntervalSec=0

[Service]
Type=simple
ExecStart=%s
Restart=on-failure
RestartSec=10s
TimeoutStopSec=10s
UMask=0077
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
`, condition, command)
}

func (s *Service) uninstall() error {
	if _, err := os.Lstat(s.paths.unit); err == nil {
		if err := s.system.stop(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, path := range []string{s.paths.unit, s.paths.binary, s.paths.version, filepath.Join(filepath.Dir(s.paths.binary), "LICENSE")} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// Preserve local settings for reinstallation. Factory reset clears /etc/kvm.
	return s.system.reload()
}
