package cloudflare

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"NanoKVM-Server/proto"

	"github.com/google/uuid"
)

const (
	modeNamed = "named"
	modeQuick = "quick"
)

type clientConfig struct {
	Mode      string `json:"mode"`
	AccessURL string `json:"accessUrl"`
	Token     string `json:"-"`
}

func validateForm(req *proto.SetCloudflareConfigReq) error {
	req.AccessURL = strings.TrimSpace(req.AccessURL)
	if req.AccessURL != "" {
		u, err := url.Parse(req.AccessURL)
		if err != nil || len(req.AccessURL) > 2048 || u.Scheme != "https" || u.Hostname() == "" ||
			u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery ||
			(u.Path != "" && u.Path != "/") || u.Opaque != "" || strings.ContainsAny(req.AccessURL, "\\\r\n\t ") {
			return errors.New("invalidURL")
		}
		if u.Port() != "" {
			port, err := strconv.Atoi(u.Port())
			if err != nil || port < 1 || port > 65535 {
				return errors.New("invalidURL")
			}
		}
	}
	if req.Token != nil {
		token := strings.TrimSpace(*req.Token)
		if !validToken(token) {
			return errors.New("invalidToken")
		}
		req.Token = &token
	}
	return nil
}

// Remotely managed tunnel tokens are base64 JSON, not account API tokens or
// shell commands copied from the dashboard. Never include token data in errors.
func validToken(value string) bool {
	if len(value) == 0 || len(value) > 4096 || strings.ContainsAny(value, "\r\n\t ") {
		return false
	}
	data, err := base64.StdEncoding.Strict().DecodeString(value)
	if err != nil {
		return false
	}
	var token struct {
		Account string `json:"a"`
		Tunnel  string `json:"t"`
		Secret  []byte `json:"s"`
	}
	if json.Unmarshal(data, &token) != nil || token.Account == "" || len(token.Secret) == 0 {
		return false
	}
	id, err := uuid.Parse(token.Tunnel)
	return err == nil && id != uuid.Nil
}

func readConfig(paths filePaths) (*clientConfig, error) {
	cfg := &clientConfig{Mode: modeNamed}
	data, err := readPrivateFile(paths.config)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(data) > 0 && json.Unmarshal(data, cfg) != nil {
		return nil, errors.New("configInvalid")
	}
	if cfg.Mode != modeNamed && cfg.Mode != modeQuick {
		return nil, errors.New("configInvalid")
	}
	data, err = readPrivateFile(paths.token)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg.Token = strings.TrimSpace(string(data))
	req := proto.SetCloudflareConfigReq{AccessURL: cfg.AccessURL}
	if cfg.Token != "" {
		req.Token = &cfg.Token
	}
	if validateForm(&req) != nil {
		return nil, errors.New("configInvalid")
	}
	cfg.AccessURL = req.AccessURL
	return cfg, nil
}

func readPrivateFile(path string) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > 16<<10 {
		return nil, errors.New("configInvalid")
	}
	return os.ReadFile(path)
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".cloudflare-*")
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
