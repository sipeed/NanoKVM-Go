package application

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/proto"
)

const maxLatestMetadataBytes int64 = 64 << 10
const latestMetadataTimeout = 15 * time.Second

type Latest struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Sha512  string `json:"sha512"`
	Size    int64  `json:"size"`
	Url     string `json:"-"`
}

var currentVersionForVersion = CurrentVersion
var getLatestForVersion = getLatest
var applicationHTTPClient = http.DefaultClient

func (s *Service) GetVersion(c *gin.Context) {
	var rsp proto.Response

	currentVersion := currentVersionForVersion()
	latest, err := getLatestForVersion()
	if err != nil {
		rsp.ErrRsp(c, -1, "failed to get latest version")
		return
	}
	latestVersion := normalizeVersion(latest.Version)

	rsp.OkRspWithData(c, &proto.GetVersionRsp{
		Current: currentVersion,
		Latest:  latestVersion,
	})
	log.Debugf("current version: %s, latest version: %s", currentVersion, latestVersion)
}

func CurrentVersion() string {
	return normalizeVersion(getCurrentVersion())
}

func getCurrentVersion() string {
	version, err := getPackageVersion(mainAppName)
	if err == nil {
		checkComponentVersion(kvmCommAppName, version)
		return normalizeVersion(version)
	}

	log.Warnf("failed to get %s package version, fallback to legacy version file: %v", mainAppName, err)
	return getLegacyCurrentVersion()
}

func getPackageVersion(packageName string) (string, error) {
	output, err := exec.Command("dpkg-query", "-W", "-f=${Version}", packageName).CombinedOutput()
	version := normalizeVersion(string(output))
	if err != nil {
		return "", fmt.Errorf("dpkg-query %s failed: %w: %s", packageName, err, version)
	}

	if version == "" {
		return "", fmt.Errorf("%s package version is empty", packageName)
	}

	return version, nil
}

func checkComponentVersion(packageName string, expectedVersion string) {
	version, err := getPackageVersion(packageName)
	if err != nil {
		log.Warnf("failed to get %s package version: %v", packageName, err)
		return
	}

	normalizedExpectedVersion := normalizeVersion(expectedVersion)
	if version != normalizedExpectedVersion {
		log.Warnf("%s package version %s does not match %s version %s", packageName, version, mainAppName, normalizedExpectedVersion)
	}
}

func getLegacyCurrentVersion() string {
	defaultVersion := "0.0.1"

	versionFile := filepath.Join(AppDir, "version")
	content, err := os.ReadFile(versionFile)
	if err != nil {
		return defaultVersion
	}

	version := normalizeVersion(string(content))
	if version == "" {
		return defaultVersion
	}

	return version
}

func normalizeVersion(version string) string {
	version = strings.TrimSpace(version)
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	return version
}

func normalizeLatestVersion(version string) string {
	version = strings.TrimSpace(version)
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		return version[1:]
	}
	return version
}

func getLatest() (*Latest, error) {
	baseURL, err := applicationUpdateBaseURL(getConfiguredUpdateServer(), isPreviewEnabled())
	if err != nil {
		return nil, err
	}

	return getLatestFromBaseURL(baseURL)
}

func getLatestFromBaseURL(baseURL string) (*Latest, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("application update base URL is empty")
	}
	cacheKey := time.Now().Unix()
	url := fmt.Sprintf("%s/%s?now=%d", baseURL, latestFileName, cacheKey)
	body, status, err := fetchApplicationUpdateFile(url)
	if err != nil {
		log.Errorf("failed to get latest version: %v", err)
		return nil, err
	}
	if status != http.StatusOK {
		log.Errorf("server responded with status code: %d", status)
		return nil, fmt.Errorf("status code %d", status)
	}

	var latest Latest
	if err := json.Unmarshal(body, &latest); err != nil {
		log.Errorf("failed to unmarshal latest version metadata: %v", err)
		return nil, err
	}
	latest.Version = normalizeLatestVersion(latest.Version)
	if err := validateLatestVersion(latest.Version); err != nil {
		log.Errorf("invalid latest version metadata: %v", err)
		return nil, err
	}
	if err := validateLatestName(latest.Name); err != nil {
		log.Errorf("invalid latest artifact name: %v", err)
		return nil, err
	}
	if latest.Size <= 0 {
		return nil, fmt.Errorf("invalid latest artifact size")
	}
	digest, err := base64.StdEncoding.Strict().DecodeString(latest.Sha512)
	if err != nil || len(digest) != sha512.Size || base64.StdEncoding.EncodeToString(digest) != latest.Sha512 {
		return nil, fmt.Errorf("invalid latest artifact SHA-512")
	}
	latest.Url = fmt.Sprintf("%s/%s", baseURL, latest.Name)

	log.Debugf("get application latest version: %s", latest.Version)
	return &latest, nil
}

func validateLatestVersion(version string) error {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must contain exactly three decimal components")
	}
	for _, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return fmt.Errorf("version contains a non-canonical component")
		}
		for index := range len(part) {
			if part[index] < '0' || part[index] > '9' {
				return fmt.Errorf("version contains a non-decimal component")
			}
		}
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return fmt.Errorf("invalid version component: %w", err)
		}
	}
	return nil
}

func validateLatestName(name string) error {
	if len(name) <= len(".tar.gz") || len(name) > 255 || !strings.HasSuffix(name, ".tar.gz") {
		return fmt.Errorf("artifact name must be a plain .tar.gz filename")
	}
	for index := range len(name) {
		character := name[index]
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '.' || character == '_' || character == '-' {
			continue
		}
		return fmt.Errorf("artifact name contains an unsafe character")
	}
	return nil
}

func fetchApplicationUpdateFile(url string) ([]byte, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), latestMetadataTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := doApplicationRequestWithHTTPFallback(applicationHTTPClient, req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxLatestMetadataBytes+1))
	if err != nil {
		return nil, resp.StatusCode, err
	}
	if int64(len(body)) > maxLatestMetadataBytes {
		return nil, resp.StatusCode, fmt.Errorf("latest version metadata exceeds %d bytes", maxLatestMetadataBytes)
	}
	return body, resp.StatusCode, nil
}
