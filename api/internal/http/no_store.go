package http

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func noStoreCredentialResponses() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if strings.HasPrefix(path, "/v1/link/") ||
			path == "/v1/instances" || strings.HasPrefix(path, "/v1/instances/") ||
			strings.HasPrefix(path, "/v1/deliveries") ||
			path == "/v1/library/sync" ||
			strings.HasPrefix(path, "/v1/account/update-destinations") ||
			strings.HasSuffix(path, "/instances") ||
			strings.HasSuffix(path, "/update-destinations") ||
			strings.HasSuffix(path, "/deliveries") {
			c.Header("Cache-Control", "no-store")
			c.Header("Pragma", "no-cache")
		}
		c.Next()
	}
}
