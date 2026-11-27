package hid

import (
	"github.com/gin-gonic/gin"

	"NanoKVM-Server/keyboardlayout"
	"NanoKVM-Server/proto"
)

const KeyboardLayoutFile = "/etc/kvm/keyboard-layout"

func (s *Service) layoutPath() string {
	if s.keyboardLayoutPath != "" {
		return s.keyboardLayoutPath
	}
	return KeyboardLayoutFile
}

type keyboardLayoutSetting struct {
	Layout string `json:"layout" form:"layout" validate:"required"`
}

func (s *Service) GetKeyboardLayout(c *gin.Context) {
	var rsp proto.Response
	layout, err := keyboardlayout.Load(s.layoutPath())
	if err != nil {
		rsp.ErrRsp(c, -2, "cannot read target keyboard layout")
		return
	}
	rsp.OkRspWithData(c, keyboardLayoutSetting{Layout: layout.ID})
}

func (s *Service) SetKeyboardLayout(c *gin.Context) {
	var req keyboardLayoutSetting
	var rsp proto.Response
	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}
	if _, ok := keyboardlayout.Lookup(req.Layout); !ok {
		rsp.ErrRsp(c, -1, "unknown target keyboard layout")
		return
	}
	if err := writeAtomicSetting(s.layoutPath(), []byte(req.Layout+"\n"), 0o644); err != nil {
		rsp.ErrRsp(c, -2, "cannot save target keyboard layout")
		return
	}
	rsp.OkRspWithData(c, req)
}
