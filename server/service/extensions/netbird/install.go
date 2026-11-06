package netbird

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	NetBirdVersion        = "0.80.0"
	serviceName           = "nanokvm-netbird.service"
	releaseURL            = "https://github.com/netbirdio/netbird/releases/download/v" + NetBirdVersion + "/"
	maxPackageBytes int64 = 32 << 20
)

// Checksums from the pinned upstream release, checked against downloaded packages.
var releaseChecksums = map[string]string{
	"armv6": "0dd140fe4a82ab90cd809ab29f8d28f2c2f722fa711605eb577189a64256de71",
	"arm64": "8cbd99fa068a7b0f3968b2d31dc341acc17dd1e97c3053e61ed5905dbdae7341",
	"amd64": "47ffaba4fc3929f31795bd6c5232d6c29744d3169e2c93e6d9c84624f0ef6405",
}

type filePaths struct{ binary, state, unit, version, socket string }

var defaultPaths = filePaths{
	binary:  "/usr/local/lib/nanokvm/netbird/netbird",
	state:   "/var/lib/nanokvm-netbird",
	unit:    "/etc/systemd/system/" + serviceName,
	version: "/usr/local/lib/nanokvm/netbird/version",
	socket:  "/run/nanokvm-netbird/api.sock",
}

func packageArch() string {
	if runtime.GOARCH == "arm" {
		return "armv6"
	}
	return runtime.GOARCH
}

func (s *Service) installed() bool {
	for _, path := range []string{s.paths.binary, s.paths.unit, s.paths.version} {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || path == s.paths.binary && info.Mode().Perm()&0111 == 0 {
			return false
		}
	}
	version, err := readSmallFile(s.paths.version)
	return err == nil && strings.TrimSpace(string(version)) == NetBirdVersion
}

func (s *Service) install(ctx context.Context) error {
	if s.installed() {
		return s.system.reload()
	}
	for _, path := range []string{"/usr/bin/netbird", "/usr/local/bin/netbird", "/etc/systemd/system/netbird.service", "/usr/lib/systemd/system/netbird.service"} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			return errors.New("externalInstallation")
		}
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
	archive := filepath.Join(staging, "package.tar.gz")
	if err := downloadPackage(ctx, releaseURL+"netbird_"+NetBirdVersion+"_linux_"+arch+".tar.gz", archive, checksum); err != nil {
		return err
	}
	if err := extractPackage(ctx, archive, staging); err != nil {
		return err
	}
	binary := filepath.Join(staging, "netbird")
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	version, err := exec.CommandContext(checkCtx, binary, "version").Output()
	if err != nil || strings.TrimSpace(string(version)) != NetBirdVersion {
		return errors.New("versionMismatch")
	}
	if _, err := os.Lstat(s.paths.unit); err == nil {
		if err := s.system.stop(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(s.paths.state, 0700); err != nil {
		return err
	}
	if err := os.Chmod(s.paths.state, 0700); err != nil {
		return err
	}
	for _, name := range packageFiles {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755); err != nil {
			return err
		}
		if err := os.Rename(filepath.Join(staging, name), filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	if err := replaceFile(s.paths.unit, []byte(s.unitContents()), 0644); err != nil {
		return err
	}
	if err := s.system.reload(); err != nil {
		return err
	}
	// The version marker is committed last; a partial install remains retryable.
	return replaceFile(s.paths.version, []byte(NetBirdVersion+"\n"), 0644)
}

func downloadPackage(ctx context.Context, url, path, checksum string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = downloadOnce(ctx, url, path, checksum)
		if err == nil {
			return nil
		}
		var networkErr net.Error
		if attempt == 2 || ctx.Err() != nil || !errors.As(err, &networkErr) && !errors.Is(err, io.ErrUnexpectedEOF) {
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

func downloadOnce(ctx context.Context, url, path, checksum string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "NanoKVM/NetBird-Installer")
	resp, err := (&http.Client{Timeout: 3 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxPackageBytes {
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
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, maxPackageBytes+1))
	if err != nil {
		return err
	}
	if n == 0 || n > maxPackageBytes || hex.EncodeToString(hash.Sum(nil)) != checksum {
		return errors.New("checksumMismatch")
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

var packageFiles = []string{"netbird", "LICENSE", "LICENSES/BSD-3-Clause.txt", "LICENSES/AGPL-3.0.txt", "LICENSES/REUSE.toml"}

func extractPackage(_ context.Context, archive, staging string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	return extractPayload(gz, staging)
}

func extractPayload(input io.Reader, staging string) error {
	reader := tar.NewReader(io.LimitReader(input, 160<<20))
	allowed := map[string]bool{}
	for _, name := range packageFiles {
		allowed[name] = true
	}
	found := map[string]bool{}
	for {
		h, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if !allowed[h.Name] {
			continue
		}
		if found[h.Name] || h.Typeflag != tar.TypeReg || h.Size <= 0 || h.Size > 128<<20 {
			return errors.New("invalidPackage")
		}
		dest := filepath.Join(staging, h.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if h.Name == "netbird" {
			mode = 0755
		}
		file, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(file, reader, h.Size)
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return err
		}
		found[h.Name] = true
	}
	if len(found) != len(allowed) {
		return errors.New("invalidPackage")
	}
	return nil
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".netbird-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if err := file.Chmod(mode); err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func (s *Service) unitContents() string {
	return fmt.Sprintf(`[Unit]
Description=NanoKVM NetBird
Wants=network-online.target
After=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
StateDirectory=nanokvm-netbird
StateDirectoryMode=0700
RuntimeDirectory=nanokvm-netbird
RuntimeDirectoryMode=0700
Environment=NB_STATE_DIR=%s
Environment=NB_FORCE_USERSPACE_FIREWALL=true
ExecStart=%s service run --service nanokvm-netbird --daemon-addr unix:///run/nanokvm-netbird/daemon.sock --enable-json-socket --json-socket unix://%s --disable-profiles --disable-networks --log-file console
Restart=on-failure
RestartSec=5s
TimeoutStopSec=30s
UMask=0077
NoNewPrivileges=yes
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE
ProtectSystem=full
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
`, s.paths.state, s.paths.binary, s.paths.socket)
}

func (s *Service) uninstall() error {
	if _, err := os.Lstat(s.paths.unit); err == nil {
		if err := s.system.stop(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, path := range []string{s.paths.unit, s.paths.version} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	for _, name := range packageFiles {
		if err := os.Remove(filepath.Join(filepath.Dir(s.paths.binary), name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// Preserve the device identity for reinstallation; factory reset clears it.
	return s.system.reload()
}
