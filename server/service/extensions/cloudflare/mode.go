package cloudflare

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"NanoKVM-Server/proto"
	authservice "NanoKVM-Server/service/auth"

	"github.com/gin-gonic/gin"
)

func (s *Service) SetMode(c *gin.Context) {
	var req proto.SetCloudflareModeReq
	var rsp proto.Response
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 16<<10)
	if c.ShouldBindJSON(&req) != nil || (req.Mode != modeNamed && req.Mode != modeQuick) {
		rsp.ErrRsp(c, -1, "invalidConfig")
		return
	}
	s.mutate(c, authservice.SensitiveActionCloudflareConfigure, req.Challenge, func(context.Context) error {
		cfg, err := readConfig(s.paths)
		if err != nil {
			return errors.New("configInvalid")
		}
		if cfg.Mode == req.Mode {
			return nil
		}
		// Switching never starts another public tunnel implicitly. Retain the
		// named tunnel's credential and URL so users can switch back later.
		_, unitErr := os.Lstat(s.paths.unit)
		if unitErr != nil && !errors.Is(unitErr, os.ErrNotExist) {
			return errors.New("serviceFailed")
		}
		hasUnit := unitErr == nil
		if hasUnit && s.system.stop() != nil {
			return errors.New("stopFailed")
		}
		if os.MkdirAll(filepath.Dir(s.paths.config), 0700) != nil {
			return errors.New("saveFailed")
		}
		paths := []string{s.paths.config}
		modes := []os.FileMode{0600}
		if hasUnit {
			paths = append(paths, s.paths.unit)
			modes = append(modes, 0644)
		}
		old := make([][]byte, len(paths))
		for i, path := range paths {
			old[i], err = readPrivateFile(path)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return errors.New("saveFailed")
			}
		}
		cfg.Mode = req.Mode
		metadata, _ := json.Marshal(cfg)
		data := [][]byte{metadata}
		if hasUnit {
			data = append(data, []byte(s.unitContents(cfg.Mode)))
		}
		rollback := func() error {
			var failures []error
			for i, path := range paths {
				if old[i] == nil {
					err = os.Remove(path)
					if errors.Is(err, os.ErrNotExist) {
						err = nil
					}
				} else {
					err = replaceFile(path, old[i], modes[i])
				}
				failures = append(failures, err)
			}
			if hasUnit {
				failures = append(failures, s.system.reload())
			}
			if errors.Join(failures...) != nil {
				return errors.New("rollbackFailed")
			}
			return errors.New("saveFailed")
		}
		for i, path := range paths {
			if replaceFile(path, data[i], modes[i]) != nil {
				return rollback()
			}
		}
		if hasUnit && s.system.reload() != nil {
			return rollback()
		}
		return nil
	})
}

var quickHostnamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.trycloudflare\.com$`)

// Read the live process's hostname, never a persisted log from a previous run.
func (s *Service) quickTunnelURL(ctx context.Context) string {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+metricsAddress+"/quicktunnel", nil)
	response, err := s.client.Do(request)
	if err != nil {
		return ""
	}
	defer response.Body.Close()
	var data struct {
		Hostname string `json:"hostname"`
	}
	if response.StatusCode != http.StatusOK || json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&data) != nil {
		return ""
	}
	host := strings.TrimPrefix(strings.ToLower(data.Hostname), "https://")
	if !quickHostnamePattern.MatchString(host) {
		return ""
	}
	return "https://" + host
}
