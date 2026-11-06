package apps

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
)

type installStreamEvent struct {
	Type string      `json:"type"`
	Text string      `json:"text,omitempty"`
	Code int         `json:"code,omitempty"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

type installStream struct {
	mu      sync.Mutex
	writer  io.Writer
	flusher http.Flusher
}

func newInstallStream(c *gin.Context) *installStream {
	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	stream := &installStream{writer: c.Writer}
	stream.flusher, _ = c.Writer.(http.Flusher)
	return stream
}

func (s *installStream) event(event installStreamEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = s.writer.Write(append(data, '\n'))
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *installStream) Write(data []byte) (int, error) {
	if len(data) > 0 {
		s.event(installStreamEvent{Type: "log", Text: string(data)})
	}
	return len(data), nil
}

func streamInstall(c *gin.Context, install func(context.Context, io.Writer) (interface{}, error)) {
	stream := newInstallStream(c)
	stream.event(installStreamEvent{Type: "log", Text: "Starting installation...\n"})
	data, err := install(c.Request.Context(), stream)
	if err != nil {
		stream.event(installStreamEvent{Type: "result", Code: -1, Msg: err.Error()})
		return
	}
	stream.event(installStreamEvent{Type: "result", Data: data})
}
