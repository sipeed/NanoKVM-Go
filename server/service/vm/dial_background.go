package vm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"NanoKVM-Server/proto"
	"NanoKVM-Server/utils"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
	log "github.com/sirupsen/logrus"
)

const (
	dialBackgroundDir        = "/kvmcomm/ui/bg"
	dialBackgroundConfigPath = "/kvmcomm/ui/bg/bg.toml"
	kvmUIConfigPath          = "/etc/kvm/kvm_ui.toml"
	dialBackgroundAPIHost    = "127.0.0.1"
	dialBackgroundAPIPort    = 65501
	maxDialBackgroundNameLen = 128
	maxDialBackgroundSize    = 32 * 1024 * 1024
)

type kvmUIConfig struct {
	HTTPAPI struct {
		Port  int    `toml:"port"`
		Token string `toml:"token"`
	} `toml:"http_api"`
}

type dialBackgroundConfig struct {
	Main struct {
		Image string `toml:"image"`
	} `toml:"main"`
	Settings struct {
		Image string `toml:"image"`
	} `toml:"settings"`
}

func (s *Service) GetDialBackgroundImages(c *gin.Context) {
	var rsp proto.Response

	files := make([]proto.DialBackgroundImage, 0)
	entries, err := os.ReadDir(dialBackgroundDir)
	if err != nil {
		if os.IsNotExist(err) {
			rsp.OkRspWithData(c, &proto.GetDialBackgroundImagesRsp{
				Files: files,
			})
			return
		}

		log.Errorf("failed to read dial background directory: %s", err)
		rsp.ErrRsp(c, -1, "get dial background images failed")
		return
	}

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Warnf("failed to stat dial background image %s: %s", entry.Name(), err)
			continue
		}

		if !info.Mode().IsRegular() || !isDialBackgroundImageName(entry.Name()) {
			continue
		}

		if err := validateBMPFile(filepath.Join(dialBackgroundDir, entry.Name())); err != nil {
			log.Warnf("skip invalid dial background BMP %s: %s", entry.Name(), err)
			continue
		}

		files = append(files, proto.DialBackgroundImage{
			Name:    entry.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime().Unix(),
			URL:     fmt.Sprintf("/api/vm/dial-background/image/%s", url.PathEscape(entry.Name())),
		})
	}

	rsp.OkRspWithData(c, &proto.GetDialBackgroundImagesRsp{
		Files:   files,
		Current: getCurrentDialBackground(),
	})
}

func (s *Service) GetDialBackgroundImage(c *gin.Context) {
	var rsp proto.Response
	name := c.Param("image")

	path, err := getDialBackgroundImagePath(name)
	if err != nil {
		rsp.ErrRsp(c, -1, "invalid image")
		return
	}

	c.Header("Cache-Control", "no-store")
	c.Header("Content-Type", "image/bmp")
	c.File(path)
}

func (s *Service) SwitchDialBackground(c *gin.Context) {
	var req proto.SwitchDialBackgroundReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if _, err := getDialBackgroundImagePath(req.ImageName); err != nil {
		log.Errorf("invalid dial background image %s: %s", req.ImageName, err)
		rsp.ErrRsp(c, -2, "invalid image")
		return
	}

	if err := switchDialBackground(req.ImageName); err != nil {
		log.Errorf("failed to switch dial background %s: %s", req.ImageName, err)
		rsp.ErrRsp(c, -3, "switch dial background failed")
		return
	}

	rsp.OkRsp(c)
}

func (s *Service) UploadDialBackgroundImage(c *gin.Context) {
	var rsp proto.Response

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		rsp.ErrRsp(c, -1, "bad request")
		return
	}
	defer file.Close()

	if header.Filename != filepath.Base(header.Filename) || strings.ContainsAny(header.Filename, `/\\`) {
		rsp.ErrRsp(c, -2, "invalid image name")
		return
	}

	name := header.Filename
	if !isDialBackgroundImageName(name) {
		rsp.ErrRsp(c, -2, "invalid image name")
		return
	}
	if header.Size > maxDialBackgroundSize {
		rsp.ErrRsp(c, -2, "image too large")
		return
	}

	if err := os.MkdirAll(dialBackgroundDir, 0o755); err != nil {
		log.Errorf("failed to create dial background directory: %s", err)
		rsp.ErrRsp(c, -3, "create directory failed")
		return
	}

	tmpFile, err := os.CreateTemp(dialBackgroundDir, ".dial-bg-*")
	if err != nil {
		log.Errorf("failed to create temporary dial background file: %s", err)
		rsp.ErrRsp(c, -4, "create file failed")
		return
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	buf := make([]byte, 32*1024)
	written, err := io.CopyBuffer(tmpFile, io.LimitReader(file, maxDialBackgroundSize+1), buf)
	if err != nil {
		_ = tmpFile.Close()
		log.Errorf("failed to save dial background image %s: %s", name, err)
		rsp.ErrRsp(c, -5, "save image failed")
		return
	}
	if written > maxDialBackgroundSize {
		_ = tmpFile.Close()
		rsp.ErrRsp(c, -2, "image too large")
		return
	}

	if err := tmpFile.Close(); err != nil {
		log.Errorf("failed to close dial background image %s: %s", name, err)
		rsp.ErrRsp(c, -5, "save image failed")
		return
	}

	if err := validateBMPFile(tmpPath); err != nil {
		log.Errorf("invalid dial background BMP %s: %s", name, err)
		rsp.ErrRsp(c, -6, "invalid BMP image")
		return
	}

	target := filepath.Join(dialBackgroundDir, name)
	if err := os.Rename(tmpPath, target); err != nil {
		log.Errorf("failed to install dial background image %s: %s", name, err)
		rsp.ErrRsp(c, -7, "save image failed")
		return
	}

	_ = utils.EnsurePermission(target, 0o644)

	rsp.OkRspWithData(c, &proto.UploadDialBackgroundRsp{File: name})
	log.Debugf("upload dial background image %s success", name)
}

func getDialBackgroundImagePath(name string) (string, error) {
	if !isDialBackgroundImageName(name) {
		return "", fmt.Errorf("invalid image name")
	}

	path := filepath.Join(dialBackgroundDir, name)
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}

	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("not a regular file")
	}

	if err := validateBMPFile(path); err != nil {
		return "", err
	}

	return path, nil
}

func isDialBackgroundImageName(name string) bool {
	if name == "" || len(name) > maxDialBackgroundNameLen {
		return false
	}

	if strings.ContainsAny(name, `/\\`) || name == "." || name == ".." {
		return false
	}

	if !strings.HasSuffix(strings.ToLower(name), ".bmp") || len(name) <= len(".bmp") {
		return false
	}

	for _, r := range name {
		switch {
		case r >= 'A' && r <= 'Z':
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '.' || r == '_' || r == '-':
		default:
			return false
		}
	}

	return true
}

func validateBMPFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 2)
	if _, err := io.ReadFull(file, header); err != nil {
		return err
	}

	if header[0] != 'B' || header[1] != 'M' {
		return fmt.Errorf("invalid BMP header")
	}

	return nil
}

func getCurrentDialBackground() string {
	data, err := os.ReadFile(dialBackgroundConfigPath)
	if err != nil {
		return ""
	}

	var config dialBackgroundConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		log.Warnf("failed to parse dial background config: %s", err)
		return ""
	}

	if isDialBackgroundImageName(config.Main.Image) {
		return config.Main.Image
	}

	if isDialBackgroundImageName(config.Settings.Image) {
		return config.Settings.Image
	}

	return ""
}

func switchDialBackground(imageName string) error {
	apiURL, token := getDialBackgroundAPIConfig()

	payload, err := json.Marshal(gin.H{"image_name": imageName})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ui api status %d: %s", resp.StatusCode, string(body))
	}

	var apiRsp struct {
		Status    string `json:"status"`
		ImageName string `json:"image_name"`
		Error     string `json:"error"`
		Detail    string `json:"detail"`
	}
	if err := json.Unmarshal(body, &apiRsp); err != nil {
		return err
	}

	if apiRsp.Status != "ok" || apiRsp.ImageName != imageName {
		return fmt.Errorf("ui api rejected image: %s %s", apiRsp.Error, apiRsp.Detail)
	}

	return nil
}

func getDialBackgroundAPIConfig() (string, string) {
	port := dialBackgroundAPIPort
	token := ""

	data, err := os.ReadFile(kvmUIConfigPath)
	if err == nil {
		var config kvmUIConfig
		if err := toml.Unmarshal(data, &config); err == nil {
			if config.HTTPAPI.Port > 0 {
				port = config.HTTPAPI.Port
			}
			token = strings.TrimSpace(config.HTTPAPI.Token)
		} else {
			log.Warnf("failed to parse %s: %s", kvmUIConfigPath, err)
		}
	}

	return fmt.Sprintf("http://%s:%d/api/v1/background", dialBackgroundAPIHost, port), token
}
