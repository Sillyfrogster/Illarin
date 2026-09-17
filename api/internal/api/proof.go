package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BrowserHeader is the header the site sends on every change it asks for
const BrowserHeader = "X-Illarin-Request"

const openFromIllarin = "Open this action from Illarin and try again."

// GuardBrowserMutations refuses a change made with a session cookie unless the site sent it
func GuardBrowserMutations(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			c.Next()
			return
		}

		_, err := c.Cookie(SessionCookie)
		if errors.Is(err, http.ErrNoCookie) {
			c.Next()
			return
		}
		if err != nil || !FromBrowser(c, origin) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": openFromIllarin})
			return
		}
		c.Next()
	}
}

// FromBrowser reports whether the site at origin sent the request
func FromBrowser(c *gin.Context, origin string) bool {
	return c.GetHeader(BrowserHeader) == "1" && c.GetHeader("Origin") == origin
}

// RequireBrowser answers 403 unless the site at origin sent the request
func RequireBrowser(c *gin.Context, origin string) bool {
	if FromBrowser(c, origin) {
		return true
	}
	Refuse(c, http.StatusForbidden, openFromIllarin)
	return false
}

// FromIllarin answers 403 unless the request carries exactly one browser header
func FromIllarin(c *gin.Context) bool {
	if len(c.Request.Header.Values(BrowserHeader)) == 1 {
		return true
	}
	Refuse(c, http.StatusForbidden, openFromIllarin)
	return false
}
