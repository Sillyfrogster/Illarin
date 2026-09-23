package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	SessionCookie = "illarin_session"
	sessionKey    = "illarin.session"
)

// Lookup finds the account a session token belongs to, or nil when it belongs to none
type Lookup func(ctx context.Context, token string) (*Account, error)

type session struct {
	once    sync.Once
	lookup  func() (*Account, error)
	account *Account
	err     error
}

// SessionToken reads the session cookie and is empty when there is none
func SessionToken(c *gin.Context) string {
	token, _ := c.Cookie(SessionCookie)
	return token
}

func SetSession(c *gin.Context, token string, expires time.Time) {
	SetCookie(c, SessionCookie, token, expires)
}

func ClearSession(c *gin.Context) {
	ClearCookie(c, SessionCookie)
}

// Sessions lets a handler ask for the signed-in account and reads it at most once per request
func Sessions(lookup Lookup) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(SessionCookie)
		if err == nil {
			c.Set(sessionKey, &session{lookup: func() (*Account, error) {
				return lookup(c.Request.Context(), token)
			}})
		}
		c.Next()
	}
}

// Current is the signed-in account, or nil when the request carries no session
func Current(c *gin.Context) (*Account, error) {
	value, found := c.Get(sessionKey)
	if !found {
		return nil, nil
	}
	held := value.(*session)
	held.once.Do(func() { held.account, held.err = held.lookup() })
	return held.account, held.err
}

// SignedIn answers 401 when nobody is signed in, naming the action they tried
func SignedIn(c *gin.Context, action string) (Account, bool) {
	current, err := Current(c)
	if err != nil {
		Refuse(c, http.StatusInternalServerError, "Could not check the signed-in account.")
		return Account{}, false
	}
	if current == nil {
		Refuse(c, http.StatusUnauthorized, "Sign in before "+action+".")
		return Account{}, false
	}
	return *current, true
}

// Verified answers 403 when the signed-in account has not verified its email
func Verified(c *gin.Context, action string) (Account, bool) {
	current, ok := SignedIn(c, action)
	if !ok {
		return Account{}, false
	}
	if !current.EmailVerified {
		Refuse(c, http.StatusForbidden, "Verify your email before "+action+".")
		return Account{}, false
	}
	return current, true
}

// Admin answers 403 unless the signed-in account is a verified admin
func Admin(c *gin.Context, action string) (Account, bool) {
	current, ok := Verified(c, action)
	if !ok {
		return Account{}, false
	}
	if current.Role != RoleAdmin {
		Refuse(c, http.StatusForbidden, "Only an admin can "+action+".")
		return Account{}, false
	}
	return current, true
}

// Staff answers 403 unless the signed-in account is a verified moderator or admin
func Staff(c *gin.Context, action string) (Account, bool) {
	current, ok := Verified(c, action)
	if !ok {
		return Account{}, false
	}
	if current.Role != RoleAdmin && current.Role != RoleModerator {
		Refuse(c, http.StatusForbidden, "Only staff can "+action+".")
		return Account{}, false
	}
	return current, true
}
