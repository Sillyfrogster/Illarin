package connect

import (
	"net/http"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

func registerAppRoute(routes api.Routes, path string, limit time.Duration, handlers map[string]gin.HandlerFunc) {
	methods := make([]string, 0, len(handlers))
	for method, handler := range handlers {
		methods = append(methods, method)
		routes.Handle(method, path, limit, func(c *gin.Context) {
			allowAppOrigin(c)
			handler(c)
		})
	}
	routes.Handle(http.MethodOptions, path, routes.Deadlines.JSON, func(c *gin.Context) {
		allowAppOrigin(c)
		c.Header("Access-Control-Allow-Methods", strings.Join(methods, ", "))
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Status(http.StatusNoContent)
	})
}

func allowAppOrigin(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Expose-Headers", "Retry-After, Content-Disposition, X-Illarin-Format, X-Illarin-Export-Target")
}
