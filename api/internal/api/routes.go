package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Deadlines are how long each kind of route may run before it gives up
type Deadlines struct {
	JSON     time.Duration
	Upload   time.Duration
	Download time.Duration
	Deliver  time.Duration
	Verify   time.Duration
}

func DefaultDeadlines() Deadlines {
	return Deadlines{
		JSON:     5 * time.Second,
		Upload:   15 * time.Minute,
		Download: 15 * time.Minute,
		Deliver:  45 * time.Second,
		Verify:   20 * time.Second,
	}
}

// Check refuses a deadline that is not set, since a route would then run as long as it likes
func (d Deadlines) Check() error {
	for name, limit := range map[string]time.Duration{
		"JSON": d.JSON, "Upload": d.Upload, "Download": d.Download,
		"Deliver": d.Deliver, "Verify": d.Verify,
	} {
		if limit <= 0 {
			return fmt.Errorf("the %s deadline is not set", name)
		}
	}
	return nil
}

// Routes registers handlers on gin and holds each one to a deadline
type Routes struct {
	router    gin.IRouter
	Deadlines Deadlines
}

func NewRoutes(router gin.IRouter, deadlines Deadlines) Routes {
	return Routes{router: router, Deadlines: deadlines}
}

func (r Routes) Handle(method, path string, limit time.Duration, handler gin.HandlerFunc) {
	r.router.Handle(method, path, withDeadline(limit), handler)
}

func withDeadline(limit time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), limit)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)

		_ = http.NewResponseController(c.Writer).SetWriteDeadline(time.Now().Add(limit))

		c.Next()
	}
}
