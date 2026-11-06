package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func currentJWTUsername(c *gin.Context) string {
	if c == nil {
		return ""
	}
	username, ok := c.Get("username")
	if !ok {
		return ""
	}
	value, ok := username.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}
