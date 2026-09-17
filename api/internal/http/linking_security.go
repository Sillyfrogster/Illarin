package http

import (
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

const maxLinkBodyBytes = 4 << 10

func readLinkJSON(c *gin.Context, destination any) bool {
	return api.ReadBoundedJSON(c, destination, maxLinkBodyBytes, "The link request is too large.")
}

func noStoreLink(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
	c.Header("Pragma", "no-cache")
}

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
			noStoreLink(c)
		}
		c.Next()
	}
}
