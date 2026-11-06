package frp

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
	"path/filepath"
	"runtime"
	"time"
)

const (
	FRPVersion            = "0.71.0"
	serviceName           = "nanokvm-frpc.service"
	releaseURL            = "https://github.com/fatedier/frp/releases/download/v" + FRPVersion + "/"
	maxArchiveBytes int64 = 64 << 20
)

// Published in v0.71.0/frp_sha256_checksums.txt. Keep hashes pinned with the
// version so an installation never runs an unverified or moving release.
var releaseChecksums = map[string]string{
	"amd64":  "84f27e39f11169f7adcef8e8b70c9329de17747b1f14dad9fb95eef5682ea716",
	"arm_hf": "eab1ecb45b00e2f9cf2ebc458fde570ceecb50689c4c5c728677f44825bf3d88",
	"arm64":  "f33c293c275d8fc68c654b6fba8f10b2551d6463d09a9fc9cffb7227eae82266",
}

type filePaths struct {
	binary string
	config string
	unit   string
}

var defaultPaths = filePaths{
	binary: "/usr/local/lib/nanokvm/frp/frpc",
	config: "/etc/kvm/frp/frpc.toml",
	unit:   "/etc/systemd/system/" + serviceName,
}

func packageArch() string {
	if runtime.GOARCH == "arm" {
		return "arm_hf"
	}
	return runtime.GOARCH
}

func (s *Service) installed() bool {
	for _, path := range []string{s.paths.binary, s.paths.unit} {
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
	arch := packageArch()
	checksum, ok := releaseChecksums[arch]
	if !ok {
		return errors.New("unsupportedArchitecture")
	}
	binDir := filepath.Dir(s.paths.binary)
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(binDir, ".install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	packageName := fmt.Sprintf("frp_%s_linux_%s", FRPVersion, arch)
	archive := filepath.Join(staging, "package.tar.gz")
	if err := downloadArchive(ctx, releaseURL+packageName+".tar.gz", archive, checksum); err != nil {
		return err
	}
	if err := extractClient(archive, staging, packageName); err != nil {
		return err
	}
	// A partial install is retryable. Only the executable and unit together
	// count as installed, and installing never enables a public tunnel.
	if err := os.Rename(filepath.Join(staging, "LICENSE"), filepath.Join(binDir, "LICENSE")); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(staging, "frpc"), s.paths.binary); err != nil {
		return err
	}
	if err := replaceFile(s.paths.unit, []byte(s.unitContents()), 0644); err != nil {
		return err
	}
	return s.system.reload()
}

func downloadArchive(ctx context.Context, url, path, checksum string) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		err = downloadArchiveOnce(ctx, url, path, checksum)
		if err == nil {
			return nil
		}
		var networkErr net.Error
		// Retry interrupted transfers, but never retry a checksum mismatch or
		// an existing destination file. Failed attempts remove only their own file.
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

func downloadArchiveOnce(ctx context.Context, url, path, checksum string) error {
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
	if resp.StatusCode != http.StatusOK || resp.ContentLength > maxArchiveBytes {
		return errors.New("invalid frp download response")
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
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(resp.Body, maxArchiveBytes+1))
	if err != nil {
		return err
	}
	if n > maxArchiveBytes || hex.EncodeToString(hash.Sum(nil)) != checksum {
		return errors.New("frp package checksum mismatch")
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

// Extract only exact, regular entries to fixed destinations. No archive path
// is used as an output path, and neither frps nor symlinks are installed.
func extractClient(archive, destination, packageName string) error {
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
	reader := tar.NewReader(io.LimitReader(gz, 256<<20))
	found := map[string]bool{}
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		name := ""
		for _, entry := range []string{"frpc", "LICENSE"} {
			if header.Name == packageName+"/"+entry {
				name = entry
			}
		}
		if name == "" {
			continue
		}
		if found[name] || header.Typeflag != tar.TypeReg || header.Size <= 0 || header.Size > maxArchiveBytes {
			return errors.New("invalid frp archive entry")
		}
		mode := os.FileMode(0644)
		if name == "frpc" {
			mode = 0755
		}
		output, err := os.OpenFile(filepath.Join(destination, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
		if err != nil {
			return err
		}
		_, copyErr := io.CopyN(output, reader, header.Size)
		closeErr := output.Close()
		if err := errors.Join(copyErr, closeErr); err != nil {
			return err
		}
		found[name] = true
	}
	if !found["frpc"] || !found["LICENSE"] {
		return errors.New("incomplete frp archive")
	}
	return nil
}

func (s *Service) unitContents() string {
	return fmt.Sprintf(`[Unit]
Description=NanoKVM FRP client
Wants=network-online.target
After=network-online.target nanokvm-go.service
ConditionPathExists=%s
StartLimitIntervalSec=0

[Service]
Type=simple
ExecStart=%s -c %s
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
`, s.paths.config, s.paths.binary, s.paths.config)
}

func (s *Service) uninstall() error {
	if _, err := os.Lstat(s.paths.unit); err == nil {
		if err := s.system.stop(); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	for _, path := range []string{s.paths.unit, s.paths.binary, filepath.Join(filepath.Dir(s.paths.binary), "LICENSE")} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	// /etc/kvm/frp survives uninstall; factory reset removes it with /etc/kvm.
	return s.system.reload()
}
