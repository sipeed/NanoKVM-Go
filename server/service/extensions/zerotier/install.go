package zerotier

import (
	"archive/tar"
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
	ZeroTierVersion       = "1.14.1"
	serviceName           = "nanokvm-zerotier.service"
	releaseURL            = "https://download.zerotier.com/RELEASES/" + ZeroTierVersion + "/dist/debian/bookworm/"
	maxPackageBytes int64 = 16 << 20
)

// SHA-256 of the versioned official Debian packages, verified independently for each ABI.
var releaseChecksums = map[string]string{
	"armhf": "c3e434babcb8859232db980f075a8a9c72c54b8b1d539aca3c5d27c50e93c821",
	"arm64": "fa3c22e779e5bda4a38c6c62df5e01e2287b7650dae7a577a1bbc315b8af9199",
	"amd64": "6f6f8c0ed785b5f05b8d831b4f208f23c6ea8e91220f9f7d4417123873bb0317",
}

type filePaths struct{ binary, state, unit, version string }

var defaultPaths = filePaths{
	binary:  "/usr/local/lib/nanokvm/zerotier/zerotier-one",
	state:   "/var/lib/zerotier-one",
	unit:    "/etc/systemd/system/" + serviceName,
	version: "/usr/local/lib/nanokvm/zerotier/version",
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
		if err != nil || !info.Mode().IsRegular() || path == s.paths.binary && info.Mode().Perm()&0111 == 0 {
			return false
		}
	}
	version, err := readSmallFile(s.paths.version)
	return err == nil && strings.TrimSpace(string(version)) == ZeroTierVersion
}

func (s *Service) install(ctx context.Context) error {
	if s.installed() {
		return s.system.reload()
	}
	// The managed daemon must not compete with a separately installed daemon or
	// mutate another installation's node identity in the native state directory.
	if _, err := os.Lstat("/usr/sbin/zerotier-one"); !errors.Is(err, os.ErrNotExist) {
		return errors.New("externalInstallation")
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
	archive := filepath.Join(staging, "package.deb")
	if err := downloadPackage(ctx, releaseURL+"zerotier-one_"+ZeroTierVersion+"_"+arch+".deb", archive, checksum); err != nil {
		return err
	}
	if err := extractPackage(ctx, archive, staging); err != nil {
		return err
	}
	binary := filepath.Join(staging, "zerotier-one")
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	version, err := exec.CommandContext(checkCtx, binary, "-v").Output()
	if err != nil || strings.TrimSpace(string(version)) != ZeroTierVersion {
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
	for _, name := range []string{"zerotier-one", "copyright"} {
		if err := os.Rename(filepath.Join(staging, name), filepath.Join(dir, name)); err != nil {
			return err
		}
	}
	if err := os.Symlink("zerotier-one", filepath.Join(staging, "zerotier-cli")); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(staging, "zerotier-cli"), filepath.Join(dir, "zerotier-cli")); err != nil {
		return err
	}
	if err := replaceFile(s.paths.unit, []byte(s.unitContents()), 0644); err != nil {
		return err
	}
	if err := s.system.reload(); err != nil {
		return err
	}
	// The version marker is committed last; a partial install remains retryable.
	return replaceFile(s.paths.version, []byte(ZeroTierVersion+"\n"), 0644)
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
	req.Header.Set("User-Agent", "NanoKVM/ZeroTier-Installer")
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

func extractPackage(ctx context.Context, archive, staging string) error {
	// dpkg-deb only decompresses the payload; no package scripts are executed.
	cmd := exec.CommandContext(ctx, "dpkg-deb", "--fsys-tarfile", archive)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	extractErr := extractPayload(stdout, staging)
	if extractErr != nil {
		_ = cmd.Process.Kill()
	}
	return errors.Join(extractErr, cmd.Wait())
}

func extractPayload(input io.Reader, staging string) error {
	reader := tar.NewReader(io.LimitReader(input, 64<<20))
	allowed := map[string]string{"./usr/sbin/zerotier-one": "zerotier-one", "./usr/share/doc/zerotier-one/copyright": "copyright"}
	found := map[string]bool{}
	for {
		h, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name, ok := allowed[h.Name]
		if !ok {
			continue
		}
		if found[name] || h.Typeflag != tar.TypeReg || h.Size <= 0 || h.Size > 32<<20 {
			return errors.New("invalidPackage")
		}
		mode := os.FileMode(0644)
		if name == "zerotier-one" {
			mode = 0755
		}
		file, err := os.OpenFile(filepath.Join(staging, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(file, reader, h.Size)
		if err := errors.Join(copyErr, file.Close()); err != nil {
			return err
		}
		found[name] = true
	}
	if len(found) != len(allowed) {
		return errors.New("invalidPackage")
	}
	// Drain dpkg-deb's trailing tar padding before waiting for the subprocess.
	_, err := io.Copy(io.Discard, io.LimitReader(input, 1<<20))
	return err
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".zerotier-*")
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
Description=NanoKVM ZeroTier
Wants=network-online.target
After=network-online.target
StartLimitIntervalSec=0

[Service]
Type=simple
StateDirectory=zerotier-one
StateDirectoryMode=0700
ExecStart=%s -U %s
Restart=on-failure
RestartSec=5s
TimeoutStopSec=10s
UMask=0077
NoNewPrivileges=yes
CapabilityBoundingSet=CAP_NET_ADMIN CAP_NET_RAW CAP_NET_BIND_SERVICE
ProtectSystem=strict
ReadWritePaths=%s
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
`, s.paths.binary, s.paths.state, s.paths.state)
}

func (s *Service) uninstall() error {
	if _, err := os.Lstat(s.paths.unit); err == nil {
		if err := s.system.stop(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(s.paths.binary)
	for _, path := range []string{s.paths.unit, s.paths.version, s.paths.binary, filepath.Join(dir, "zerotier-cli"), filepath.Join(dir, "copyright")} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// Keep the node identity and memberships for reinstallation.
	return s.system.reload()
}
