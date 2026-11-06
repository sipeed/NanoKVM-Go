package config

import (
	"errors"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/spf13/viper"
)

// NormalizeUpdateServer accepts a directory URL, including LAN hosts and ports.
// Queries, fragments and credentials cannot be preserved when appending channel
// and artifact paths, so reject them instead of silently changing their meaning.
func NormalizeUpdateServer(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	invalid := errors.New("update server must be an HTTP(S) directory URL without credentials, query or fragment")
	if len(raw) > 2048 || strings.ContainsAny(raw, "\\?#") || strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return "", invalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" {
		return "", invalid
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", invalid
		}
	} else if strings.HasSuffix(parsed.Host, ":") {
		return "", invalid
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func GetUpdateServer() string {
	GetInstance()
	configMu.Lock()
	defer configMu.Unlock()
	return instance.Update.Server
}

func SetUpdateServer(server string) error {
	server, err := NormalizeUpdateServer(server)
	if err != nil {
		return err
	}
	GetInstance()
	configMu.Lock()
	defer configMu.Unlock()

	updated := instance
	updated.Update.Server = server
	if err := save(updated); err != nil {
		return err
	}
	instance.Update.Server = server
	viper.Set("update.server", server)
	return nil
}
