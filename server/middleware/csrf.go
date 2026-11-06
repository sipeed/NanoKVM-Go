package middleware

import (
	"net/http"
	"net/url"
	"strings"

	"NanoKVM-Server/config"

	"github.com/gin-gonic/gin"
)

type csrfPolicy struct {
	TrustedOrigins     []string
	AllowMissingOrigin bool
}

func CSRFProtection() gin.HandlerFunc {
	conf := config.GetInstance()
	return csrfProtection(csrfPolicy{
		TrustedOrigins:     append([]string(nil), conf.Security.CSRF.TrustedOrigins...),
		AllowMissingOrigin: conf.Security.CSRF.AllowMissingOrigin,
	})
}

func csrfProtection(policy csrfPolicy) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requiresCSRFCheck(c.Request.Method) || hasBearerToken(c) || !hasCookieToken(c) {
			c.Next()
			return
		}

		if isSameOrigin(c.Request, policy) {
			c.Next()
			return
		}

		if policy.AllowMissingOrigin && missingOrigin(c.Request) {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"code": -1, "msg": "csrf validation failed", "data": nil})
		c.Abort()
	}
}

func requiresCSRFCheck(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func hasBearerToken(c *gin.Context) bool {
	authHeader := c.GetHeader("Authorization")
	return strings.HasPrefix(authHeader, "Bearer ") && strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")) != ""
}

func hasCookieToken(c *gin.Context) bool {
	token, err := c.Cookie("nano-kvm-token")
	return err == nil && strings.TrimSpace(token) != ""
}

func isSameOrigin(r *http.Request, policy csrfPolicy) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return sameOrigin(origin, requestOrigin(r)) || isTrustedOrigin(origin, policy)
	}

	if referer := r.Header.Get("Referer"); referer != "" {
		return sameOrigin(referer, requestOrigin(r)) || isTrustedOrigin(referer, policy)
	}

	return false
}

func missingOrigin(r *http.Request) bool {
	return r.Header.Get("Origin") == "" && r.Header.Get("Referer") == ""
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = strings.TrimSpace(strings.Split(forwardedProto, ",")[0])
	}

	return scheme + "://" + r.Host
}

func sameOrigin(candidate string, expected string) bool {
	candidateURL, err := url.Parse(candidate)
	if err != nil || candidateURL.Scheme == "" || candidateURL.Host == "" {
		return false
	}

	expectedURL, err := url.Parse(expected)
	if err != nil || expectedURL.Scheme == "" || expectedURL.Host == "" {
		return false
	}

	return strings.EqualFold(candidateURL.Scheme, expectedURL.Scheme) && strings.EqualFold(candidateURL.Host, expectedURL.Host)
}

func isTrustedOrigin(candidate string, policy csrfPolicy) bool {
	for _, trusted := range policy.TrustedOrigins {
		if sameOrigin(candidate, trusted) {
			return true
		}
	}

	return false
}
