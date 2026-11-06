package hid

import (
	log "github.com/sirupsen/logrus"
)

func (h *Hid) Keyboard(queue <-chan []byte) {
	defer func() {
		if err := h.ReleaseInputSession(InternalInputSessionKeyboardQueue, "keyboard queue closed"); err != nil {
			log.Debugf("release keyboard queue input session: %s", err)
		}
	}()

	for event := range queue {
		if len(event) != 8 {
			log.Debugf("invalid keyboard event: %v", event)
			continue
		}

		if err := h.RouteKeyboardInput(InternalInputSessionKeyboardQueue, event); err != nil {
			log.Debugf("route keyboard event: %s", err)
		}
	}
}
