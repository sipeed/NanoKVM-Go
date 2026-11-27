package hid

import (
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"NanoKVM-Server/keyboardlayout"
	"NanoKVM-Server/proto"
)

const (
	pasteKeyDownHold      = 10 * time.Millisecond
	pastePostReleaseDelay = 25 * time.Millisecond
)

type PasteReq struct {
	Content string `form:"content" validate:"required"`
}

func (s *Service) Paste(c *gin.Context) {
	var req PasteReq
	var rsp proto.Response
	pasteSleep := s.pasteSleep
	if pasteSleep == nil {
		pasteSleep = time.Sleep
	}

	if err := proto.ParseFormRequest(c, &req); err != nil {
		rsp.ErrRsp(c, -1, "invalid arguments")
		return
	}

	if utf8.RuneCountInString(req.Content) > 1024 {
		rsp.ErrRsp(c, -2, "content too long")
		return
	}

	s.pasteMutex.Lock()
	defer s.pasteMutex.Unlock()
	layout, err := keyboardlayout.Load(s.layoutPath())
	if err != nil {
		rsp.ErrRsp(c, -4, "cannot read target keyboard layout")
		return
	}
	strokes, err := layout.Encode(req.Content)
	if err != nil {
		rsp.ErrRsp(c, -5, err.Error())
		return
	}
	routerInstalled := InputRouterSinkInstalled()
	if routerInstalled {
		defer func() {
			if err := s.hid.ReleaseInputSession(InternalInputSessionPaste, "paste complete"); err != nil {
				log.Debugf("release paste input session: %s", err)
			}
		}()
	}

	keyUp := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	for _, stroke := range strokes {
		keyDown := stroke.Report()
		if routerInstalled {
			if err := s.hid.RouteKeyboardInput(InternalInputSessionPaste, keyDown); err != nil {
				rsp.ErrRsp(c, -3, "paste failed")
				return
			}
			pasteSleep(pasteKeyDownHold)
			if err := s.hid.RouteKeyboardInput(InternalInputSessionPaste, keyUp); err != nil {
				rsp.ErrRsp(c, -3, "paste failed")
				return
			}
			pasteSleep(pastePostReleaseDelay)
			continue
		}

		s.hid.withKeyboardLock(func() {
			if err = s.hid.writeHid0NoLock(keyDown); err != nil {
				return
			}
			pasteSleep(pasteKeyDownHold)
			err = s.hid.writeHid0NoLock(keyUp)
			pasteSleep(pastePostReleaseDelay)
		})
		if err != nil {
			rsp.ErrRsp(c, -3, "paste failed")
			return
		}
	}

	rsp.OkRsp(c)
	log.Debugf("hid paste success, total %d characters processed", utf8.RuneCountInString(req.Content))
}
