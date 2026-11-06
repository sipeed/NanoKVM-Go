package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	corsgin "github.com/rs/cors/wrapper/gin"
)

var loopbackViteOrigins = originSet(
	[]string{
		"http://localhost:3001",
		"http://127.0.0.1:3001",
		"http://[::1]:3001",
	},
)

// DevelopmentCORS permits only the local Vite server, the request's own
// origin, and explicitly configured exact origins. It is installed only when
// authentication is disabled, so cross-origin credentials are unnecessary.
func DevelopmentCORS(configuredOrigins []string) gin.HandlerFunc {
	allowed := originSet(configuredOrigins)
	for origin := range loopbackViteOrigins {
		allowed[origin] = struct{}{}
	}
	isAllowed := func(request *http.Request, origin string) bool {
		key, ok := normalizedOrigin(origin)
		if !ok {
			return false
		}

		if _, ok := allowed[key]; ok {
			return true
		}

		requestOrigin, ok := normalizedRequestOrigin(request)
		return ok && key == requestOrigin
	}

	corsMiddleware := corsgin.New(corsgin.Options{
		AllowOriginVaryRequestFunc: func(request *http.Request, origin string) (bool, []string) {
			return isAllowed(request, origin), nil
		},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
		},
		AllowedHeaders: []string{
			"Accept",
			"Authorization",
			"Content-Type",
			"X-AP-Key",
		},
	})

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && !isAllowed(c.Request, origin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": -1,
				"msg":  "origin is not allowed",
				"data": nil,
			})
			return
		}
		corsMiddleware(c)
	}
}

func originSet(origins []string) map[string]struct{} {
	result := make(map[string]struct{}, len(origins))
	for _, origin := range origins {
		if key, ok := normalizedOrigin(origin); ok {
			result[key] = struct{}{}
		}
	}
	return result
}

func normalizedRequestOrigin(request *http.Request) (string, bool) {
	scheme := "http"
	if request.TLS != nil {
		scheme = "https"
	}
	return normalizedOrigin(scheme + "://" + request.Host)
}

func normalizedOrigin(raw string) (string, bool) {
	if raw == "" || raw != strings.TrimSpace(raw) {
		return "", false
	}

	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") ||
		parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawPath != "" ||
		parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return "", false
	}

	hostname := strings.ToLower(parsed.Hostname())
	if hostname == "" {
		return "", false
	}

	port := parsed.Port()
	if port == "" {
		if parsed.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	return parsed.Scheme + "://" + net.JoinHostPort(hostname, port), true
}
