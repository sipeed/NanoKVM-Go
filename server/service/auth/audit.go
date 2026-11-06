package auth

import (
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

var allowedAuthAuditFields = map[string]struct{}{
	"user":    {},
	"ip":      {},
	"op":      {},
	"code_id": {},
	"reason":  {},
}

func auditAuthEvent(event string, fields log.Fields) {
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}

	safeFields := log.Fields{}
	for key, value := range fields {
		if _, ok := allowedAuthAuditFields[key]; ok && value != nil {
			safeFields[key] = value
		}
	}

	log.WithFields(safeFields).Info(event)
}

func authAuditFields(c *gin.Context, username, op, reason string) log.Fields {
	fields := log.Fields{
		"user": authRateLimitValue(username),
	}
	if c != nil {
		fields["ip"] = c.ClientIP()
	}
	if op != "" {
		fields["op"] = op
	}
	if reason != "" {
		fields["reason"] = reason
	}
	return fields
}

func auditRecoveryCodeUsed(c *gin.Context, username, op, codeID string) {
	fields := authAuditFields(c, username, op, "recovery_code_used")
	if codeID != "" {
		fields["code_id"] = codeID
	}
	auditAuthEvent("auth_recovery_code_used", fields)
}
