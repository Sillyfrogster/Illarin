package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SetCookie stores a credential the page's scripts cannot read
func SetCookie(c *gin.Context, name, value string, expires time.Time) {
	cookie := credentialCookie(c, name)
	cookie.Value = value
	cookie.Expires = expires
	cookie.MaxAge = int(time.Until(expires).Seconds())
	http.SetCookie(c.Writer, cookie)
}

func ClearCookie(c *gin.Context, name string) {
	cookie := credentialCookie(c, name)
	cookie.MaxAge = -1
	http.SetCookie(c.Writer, cookie)
}

func credentialCookie(c *gin.Context, name string) *http.Cookie {
	secure := c.Request.TLS != nil || strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
	return &http.Cookie{
		Name:     name,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
