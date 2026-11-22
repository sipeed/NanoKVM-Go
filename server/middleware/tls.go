package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

const (
	contentSecurityPolicy = "default-src 'self'; base-uri 'self'; connect-src 'self'; font-src 'self' data:; form-action 'self'; frame-ancestors 'none'; frame-src 'none'; img-src 'self' data: blob:; manifest-src 'self'; media-src 'self' blob:; object-src 'none'; script-src 'self'; style-src 'self' 'unsafe-inline'; worker-src 'self' blob:"
	permissionsPolicy     = "accelerometer=(), autoplay=(self), camera=(self), clipboard-read=(), clipboard-write=(self), display-capture=(), encrypted-media=(), fullscreen=(self), geolocation=(), gyroscope=(), magnetometer=(), microphone=(self), midi=(), payment=(), picture-in-picture=(self), publickey-credentials-get=(self), screen-wake-lock=(self), usb=()"
)

func Tls() gin.HandlerFunc {
	secureMiddleware := secure.New(secure.Options{
		SSLRedirect:          false,
		STSSeconds:           31536000,
		STSIncludeSubdomains: true,
	})

	return func(c *gin.Context) {
		if err := secureMiddleware.Process(c.Writer, c.Request); err != nil {
			c.Abort()
		}
	}
}

// SecurityHeaders applies browser protections that are safe on both HTTP and
// HTTPS. Transport-specific HSTS remains in Tls so plaintext development mode
// cannot accidentally assert a policy browsers should honor only over TLS.
func SecurityHeaders() gin.HandlerFunc {
	secureMiddleware := secure.New(secure.Options{
		FrameDeny:               true,
		ContentTypeNosniff:      true,
		ContentSecurityPolicy:   contentSecurityPolicy,
		ReferrerPolicy:          "no-referrer",
		PermissionsPolicy:       permissionsPolicy,
		CrossOriginOpenerPolicy: "same-origin",
	})

	return func(c *gin.Context) {
		err := secureMiddleware.Process(c.Writer, c.Request)
		if err != nil {
			c.Abort()
			return
		}

		// The SPA embeds only same-origin resources; its cross-origin URLs are
		// external navigations. Blob workers and media are permitted explicitly
		// by the CSP, so cross-origin isolation does not break the application.
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		// Check if redirect response was set
		if status := c.Writer.Status(); status >= 300 && status < 400 {
			c.Abort()
		}
	}
}
