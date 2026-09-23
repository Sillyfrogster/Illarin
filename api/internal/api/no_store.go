package api

import (
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// noStoreStarts and noStoreEnds pick out the routes whose answers carry credentials or a connected app's state
var (
	noStoreStarts = []string{
		"/v1/connect/", "/v1/connected-apps", "/v1/sends", "/v1/library/sync", "/v1/account/integrations",
		"/v1/link/", "/v1/instances", "/v1/deliveries", "/v1/account/update-destinations",
	}
	noStoreEnds = []string{"/connected-apps", "/sends", "/integrations", "/update-destinations", "/deliveries", "/instances"}
)

func NoStoreCredentialResponses() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.FullPath()
		if slices.ContainsFunc(noStoreStarts, func(start string) bool { return strings.HasPrefix(path, start) }) ||
			slices.ContainsFunc(noStoreEnds, func(end string) bool { return strings.HasSuffix(path, end) }) {
			c.Header("Cache-Control", "no-store")
			c.Header("Pragma", "no-cache")
		}
		c.Next()
	}
}
