package network

import (
	"encoding/json"
	"os/exec"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

// Byte-level bounds enforced by wifi.sh on a pre-stored WiFi. The validator's
// max tag counts runes, so the 32-byte SSID limit needs an explicit check.
const (
	wifiPresetMaxSSIDBytes     = 32
	wifiPresetMaxPasswordBytes = 128
)

// wifiPresetRequestValid reports whether a preset request stays within the
// byte bounds that wifi.sh applies to SSIDs and passphrases.
func wifiPresetRequestValid(req *proto.PresetWifiReq) bool {
	if req.Ssid == "" || len(req.Ssid) > wifiPresetMaxSSIDBytes {
		return false
	}
	return len(req.Password) <= wifiPresetMaxPasswordBytes
}

// parseWifiPresetStatus parses the JSON emitted by `wifi.sh preset status`.
// It tolerates the `\xNN` escapes the script emits for non-ASCII SSIDs.
func parseWifiPresetStatus(out []byte) (string, error) {
	var data proto.GetWifiPresetRsp
	if err := json.Unmarshal(fixInvalidJSONEscapes(out), &data); err != nil {
		return "", err
	}
	return data.Ssid, nil
}

func (s *Service) PresetWifi(c *gin.Context) {
	var req proto.PresetWifiReq
	var rsp proto.Response

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}
	if !wifiPresetRequestValid(&req) {
		rsp.ErrRsp(c, -1, "invalid parameters")
		return
	}

	args := []string{"preset", "save", req.Ssid}
	if req.Password != "" {
		args = append(args, req.Password)
	}

	if out, err := exec.Command(WiFiScript, args...).CombinedOutput(); err != nil {
		rsp.ErrRsp(c, -2, "preset wifi failed")
		log.Errorf("preset wifi failed, err: %s, out: %s", err, string(out))
		return
	}

	rsp.OkRsp(c)
	log.Debugf("preset wifi successfully")
}

func (s *Service) GetWifiPreset(c *gin.Context) {
	var rsp proto.Response

	out, err := exec.Command(WiFiScript, "preset", "status").CombinedOutput()
	if err != nil {
		rsp.ErrRsp(c, -1, "get wifi preset failed")
		log.Errorf("get wifi preset failed, err: %s, out: %s", err, string(out))
		return
	}

	ssid, err := parseWifiPresetStatus(out)
	if err != nil {
		rsp.ErrRsp(c, -2, "get wifi preset failed")
		log.Errorf("parse wifi preset failed, err: %s", err)
		return
	}

	rsp.OkRspWithData(c, &proto.GetWifiPresetRsp{
		Ssid: ssid,
	})
	log.Debugf("get wifi preset successfully")
}

func (s *Service) RemoveWifiPreset(c *gin.Context) {
	var rsp proto.Response

	if out, err := exec.Command(WiFiScript, "preset", "remove").CombinedOutput(); err != nil {
		rsp.ErrRsp(c, -1, "remove wifi preset failed")
		log.Errorf("remove wifi preset failed, err: %s, out: %s", err, string(out))
		return
	}

	rsp.OkRsp(c)
	log.Debugf("remove wifi preset successfully")
}
