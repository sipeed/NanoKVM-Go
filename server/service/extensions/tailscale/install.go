package tailscale

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"NanoKVM-Server/utils"

	"github.com/shirou/gopsutil/v4/host"
	log "github.com/sirupsen/logrus"
)

const (
	OriginalURL      = "https://pkgs.tailscale.com/stable/"
	TailscaleVersion = "1.102.2"

	installStagingRoot = "/root"

	TailscaleFile  = "/usr/bin/tailscale"
	TailscaledFile = "/usr/sbin/tailscaled"
	EnvFile        = "/etc/default/tailscaled"

	Tailscaled     = "tailscaled.service"
	TailscaledConf = "/etc/systemd/system/tailscaled.service"
)

type installStaging struct {
	dir     string
	archive string
}

func newInstallStaging(parentDir, packageName string) (*installStaging, error) {
	if packageName == "" || packageName == "." || filepath.Base(packageName) != packageName {
		return nil, fmt.Errorf("invalid tailscale package name %q", packageName)
	}

	dir, err := os.MkdirTemp(parentDir, ".tailscale-install-*")
	if err != nil {
		return nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return nil, err
	}

	archive := filepath.Join(dir, packageName)
	file, err := os.OpenFile(archive, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		cleanup()
		return nil, err
	}
	if err := file.Close(); err != nil {
		cleanup()
		return nil, err
	}
	if err := os.Chmod(archive, 0o600); err != nil {
		cleanup()
		return nil, err
	}

	return &installStaging{dir: dir, archive: archive}, nil
}

func (s *installStaging) cleanup() {
	if s != nil {
		_ = os.RemoveAll(s.dir)
	}
}

type installPayload struct {
	tailscale   string
	tailscaled  string
	service     string
	serviceMode os.FileMode
}

func validateInstallPayload(extractedDir string) (*installPayload, error) {
	payload := &installPayload{
		tailscale:  filepath.Join(extractedDir, "tailscale"),
		tailscaled: filepath.Join(extractedDir, "tailscaled"),
		service:    filepath.Join(extractedDir, "systemd", Tailscaled),
	}

	for _, expected := range []struct {
		name string
		path string
	}{
		{name: "tailscale", path: payload.tailscale},
		{name: "tailscaled", path: payload.tailscaled},
		{name: "systemd/tailscaled.service", path: payload.service},
	} {
		info, err := os.Lstat(expected.path)
		if err != nil {
			return nil, fmt.Errorf("validate %s: %w", expected.name, err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("validate %s: expected a regular file", expected.name)
		}
		if expected.path == payload.service {
			payload.serviceMode = info.Mode()
		}
	}

	return payload, nil
}

// mapTailscaleArch maps GO/kernel arch names to Tailscale package arch suffixes.
// See https://pkgs.tailscale.com/stable/ (arm, arm64, amd64, 386, riscv64, ...).
func mapTailscaleArch(kernelArch string) string {
	arch := strings.ToLower(strings.TrimSpace(kernelArch))
	switch arch {
	case "aarch64", "arm64":
		return "arm64"
	case "x86_64", "amd64":
		return "amd64"
	case "i386", "i686", "386":
		return "386"
	case "arm", "armv5l", "armv6l", "armv7", "armv7l", "armhf":
		// Official packages use "arm" for 32-bit ARM (including armv7).
		return "arm"
	case "riscv64":
		return "riscv64"
	case "mips", "mipsle", "mips64", "mips64le":
		return arch
	default:
		if arch == "" {
			return "arm64"
		}
		return arch
	}
}

func detectTailscaleArch() string {
	info, err := host.Info()
	if err != nil || info.KernelArch == "" {
		log.Warnf("failed to detect kernel arch for tailscale package, defaulting to arm64: %v", err)
		return "arm64"
	}
	mapped := mapTailscaleArch(info.KernelArch)
	if mapped != info.KernelArch {
		log.Debugf("mapped kernel arch %q to tailscale package arch %q", info.KernelArch, mapped)
	}
	return mapped
}

func GetTailscaleName() string {
	// A missing client is installed at the pinned version. isInstalled accepts
	// any existing version, so managed setup never replaces a working install.
	return fmt.Sprintf("tailscale_%s_%s.tgz", TailscaleVersion, detectTailscaleArch())
}

func isInstalled() bool {
	for _, file := range []string{TailscaleFile, TailscaledFile} {
		if _, err := os.Stat(file); err != nil {
			return false
		}
	}

	return true
}

func install() error {
	pkgName := GetTailscaleName()
	log.Infof("installing tailscale package %s", pkgName)
	staging, err := newInstallStaging(installStagingRoot, pkgName)
	if err != nil {
		log.Errorf("failed to create tailscale install staging: %s", err)
		return err
	}
	defer staging.cleanup()

	// download
	if err := download(staging.archive); err != nil {
		log.Errorf("failed to download tailscale: %s", err)
		return err
	}

	// decompress
	dir, err := utils.UnTarGz(staging.archive, staging.dir)
	if err != nil {
		log.Errorf("failed to decompress tailscale: %s", err)
		return err
	}
	payload, err := validateInstallPayload(dir)
	if err != nil {
		log.Errorf("invalid tailscale install payload: %s", err)
		return err
	}

	// move binaries
	if err := os.Rename(payload.tailscale, TailscaleFile); err != nil {
		log.Errorf("failed to rename tailscale file: %s", err)
		return err
	}

	if err := os.Rename(payload.tailscaled, TailscaledFile); err != nil {
		log.Errorf("failed to rename tailscaled file: %s", err)
		return err
	}

	// service config
	if err := os.Chmod(payload.service, payload.serviceMode|0111); err != nil {
		log.Printf("add service permission failed: %s", err)
		return err
	}

	if err := os.Rename(payload.service, TailscaledConf); err != nil {
		log.Printf("failed to move %s to %s: %s", payload.service, TailscaledConf, err)
		return err
	}

	// create required directories
	for _, directory := range []string{"/var/lib/tailscale", "/run/tailscale"} {
		if err := os.MkdirAll(directory, 0755); err != nil {
			log.Errorf("failed to create directory %s: %s", directory, err)
		}
		if err := os.Chmod(directory, 0755); err != nil {
			log.Errorf("failed to chmod directory %s: %s", directory, err)
		}
	}

	// create environment file
	envContent := `PORT=41641
FLAGS=""
`
	err = os.WriteFile(EnvFile, []byte(envContent), 0644)
	if err != nil {
		log.Errorf("failed to create tailscaled environment file: %s", err)
		return err
	}

	// start service
	if err := utils.DaemonReload(); err != nil {
		log.Errorf("failed to reload systemd: %s", err)
		return err
	}

	if err := utils.StartService(Tailscaled, true); err != nil {
		log.Errorf("failed to start tailscale: %s", err)
		return err
	}

	log.Debugf("install tailscale successfully")
	return nil
}

func uninstall() error {
	if !isInstalled() {
		// The daemon preferences live under /var/lib/tailscale and are retained.
		// Without a working daemon we cannot prove routing is disabled, so keep the
		// forwarding persistence needed by a later reinstall.
		log.Debugf("tailscale not install")
		return nil
	}

	if err := prepareRoutingForUninstall(); err != nil {
		// Keep the existing forwarding persistence when daemon state cannot be
		// read or safely updated. Removing it while route preferences survive
		// would make a later reinstall advertise routes without forwarding.
		log.Warnf("failed to prepare tailscale routing for uninstall; preserving forwarding: %s", err)
	}

	// stop service
	if err := utils.StopService(Tailscaled, true); err != nil {
		log.Errorf("failed to stop tailscale: %s", err)
		return err
	}

	// delete files
	for _, file := range []string{TailscaleFile, TailscaledFile, EnvFile, TailscaledConf} {
		if err := os.Remove(file); err != nil {
			log.Errorf("failed to remove file %s: %s", file, err)
			return err
		}
	}

	// reload
	if err := utils.DaemonReload(); err != nil {
		log.Warnf("failed to reload systemd: %s", err)
	}

	log.Debugf("uninstall tailscale successfully")
	return nil
}

func download(target string) error {
	url, err := getDownloadURL()
	if err != nil {
		log.Errorf("failed to get Tailscale download url: %s", err)
		return err
	}

	client := &http.Client{Timeout: 3 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		log.Errorf("failed to download Tailscale: %s", err)
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d url=%s", resp.StatusCode, url)
	}

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		log.Errorf("failed to create file: %s", err)
		return err
	}
	if err := os.Chmod(target, 0o600); err != nil {
		_ = out.Close()
		return err
	}

	_, copyErr := io.Copy(out, resp.Body)
	closeErr := out.Close()
	if copyErr != nil {
		log.Errorf("failed to copy response body to file: %s", copyErr)
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}

	log.Debugf("download Tailscale successfully")
	return nil
}

func getDownloadURL() (string, error) {
	pkgURL := OriginalURL + GetTailscaleName()
	client := &http.Client{
		Timeout: 30 * time.Second,
		// Follow redirects; keep final URL for the real download.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}
	resp, err := client.Get(pkgURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusFound &&
		resp.StatusCode != http.StatusMovedPermanently && resp.StatusCode != http.StatusTemporaryRedirect {
		return "", fmt.Errorf("unexpected status code: %d url=%s", resp.StatusCode, pkgURL)
	}

	// After redirects, Request.URL is the final location.
	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String(), nil
	}
	return pkgURL, nil
}
