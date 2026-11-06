package router

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/contrib/static"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

func Init(r *gin.Engine) {
	web(r)
	server(r)
	log.Debugf("router init done")
}

func web(r *gin.Engine) {
	execPath, err := os.Executable()
	if err != nil {
		panic("invalid executable path")
	}

	execDir := filepath.Dir(execPath)
	webPath := fmt.Sprintf("%s/web", execDir)

	r.Use(webCacheControl())
	r.Use(static.Serve("/", static.LocalFile(webPath, true)))

	r.GET("/kvm", func(c *gin.Context) {
		c.Redirect(302, "/")
	})
}

// webCacheControl keeps the entry document revalidating while allowing the
// content-hashed assets to be cached. Without this, browsers can keep serving
// an old index.html and its old locale chunks after a Web redeploy.
func webCacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		switch {
		case path == "/" || strings.HasSuffix(path, ".html"):
			c.Header("Cache-Control", "no-cache, must-revalidate")
		case strings.HasPrefix(path, "/assets/"):
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}

func server(r *gin.Engine) {
	routers := []func(c *gin.Engine){
		authRouter,
		applicationRouter,
		appsRouter,
		vmRouter,
		streamRouter,
		storageRouter,
		networkRouter,
		hidRouter,
		aiRouter,
		wsRouter,
		localRouter,
		extensionsRouter,
		speedtestRouter,
	}

	for _, fn := range routers {
		fn(r)
	}
}
