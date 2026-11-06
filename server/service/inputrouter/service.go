package inputrouter

import (
	"encoding/json"
	"net/http"
	"time"

	"NanoKVM-Server/proto"

	"github.com/gin-gonic/gin"
)

type Service struct {
	router *Router
}

type Status struct {
	ProtocolVersion int           `json:"protocol_version"`
	MaxContacts     int           `json:"max_contacts"`
	LegacyEnabled   bool          `json:"legacy_enabled"`
	Runtime         RuntimeStatus `json:"runtime"`
}

func NewService() *Service {
	return &Service{router: GetRouter()}
}

func (service *Service) GetStatus(c *gin.Context) {
	var rsp proto.Response
	rsp.OkRspWithData(c, &Status{
		ProtocolVersion: ProtocolVersion,
		MaxContacts:     MaxContacts,
		LegacyEnabled:   true,
		Runtime:         service.router.Status(),
	})
}

func (service *Service) StreamTraces(c *gin.Context) {
	records, unsubscribe := service.router.Traces().Subscribe(256)
	defer unsubscribe()

	c.Header("Content-Type", "application/x-ndjson")
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)
	c.Writer.Flush()

	encoder := json.NewEncoder(c.Writer)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case record, ok := <-records:
			if !ok || encoder.Encode(record) != nil {
				return
			}
			c.Writer.Flush()
		case <-heartbeat.C:
			if _, err := c.Writer.Write([]byte("\n")); err != nil {
				return
			}
			c.Writer.Flush()
		case <-c.Request.Context().Done():
			return
		}
	}
}
