package ai

import (
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

const serviceStateEventSnapshot = "snapshot"

type serviceStateSnapshotEvent struct {
	Type     string                       `json:"type"`
	Revision int64                        `json:"revision"`
	Services map[string]ServiceStateEntry `json:"services"`
}

func (s *Service) StreamServiceStateEvents(c *gin.Context) {
	manager := DefaultServiceStateManager()
	ch := manager.Subscribe()
	defer manager.Unsubscribe(ch)

	snapshot, err := manager.List()
	if err != nil {
		log.Errorf("failed to list service states for event stream: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list services failed"})
		return
	}

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	ctx := c.Request.Context()
	sentSnapshot := false
	c.Stream(func(_ io.Writer) bool {
		if !sentSnapshot {
			c.SSEvent(serviceStateEventSnapshot, serviceStateSnapshotEvent{
				Type:     serviceStateEventSnapshot,
				Revision: snapshot.Revision,
				Services: snapshot.Services,
			})
			sentSnapshot = true
			return true
		}

		select {
		case event := <-ch:
			c.SSEvent(event.Type, event)
			return true
		case <-ctx.Done():
			return false
		}
	})
}
