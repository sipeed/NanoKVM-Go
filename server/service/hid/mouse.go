package hid

import (
	log "github.com/sirupsen/logrus"
)

func (h *Hid) Mouse(queue <-chan []byte) {
	defer func() {
		if err := h.ReleaseInputSession(InternalInputSessionMouseQueue, "mouse queue closed"); err != nil {
			log.Debugf("release mouse queue input session: %s", err)
		}
	}()

	for event := range queue {
		switch len(event) {
		case 4:
			if err := h.RoutePointerInput(InternalInputSessionMouseQueue, event); err != nil {
				log.Debugf("route relative mouse event: %s", err)
			}
		case 7:
			if err := h.RoutePointerInput(InternalInputSessionMouseQueue, event); err != nil {
				log.Debugf("route absolute mouse event: %s", err)
			}
		case TouchScreenReportSize:
			if err := h.RouteTouchInput(InternalInputSessionMouseQueue, event); err != nil {
				log.Debugf("route touchscreen event: %s", err)
			}
		default:
			log.Debugf("invalid mouse event: %v", event)
		}
	}
}
