package account

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/connect"
	"github.com/gin-gonic/gin"
)

const (
	oauthStateCookieName  = "illarin_discord_state"
	oauthReturnCookieName = "illarin_discord_return"

	accountSignUpLimit                = 20
	accountSignInLimit                = 10
	accountPasswordResetLimit         = 5
	accountPasswordResetCompleteLimit = 10
)

func (h *Handlers) SignUp(c *gin.Context) {
	if !h.allowAccountAttempt(c, "account-sign-up", accountSignUpLimit, time.Hour) {
		return
	}
	var request SignUpRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send an email, password and handle as JSON.")
		return
	}

	created, token, expires, err := h.accounts.SignUp(c.Request.Context(), SignUpInput{
		Email:    string(request.Email),
		Password: request.Password,
		Handle:   request.Handle,
	})
	if err != nil {
		h.accountError(c, err)
		return
	}
	api.SetSession(c, token, expires)
	c.JSON(http.StatusCreated, toAPIAccount(created))
}

func (h *Handlers) SignIn(c *gin.Context) {
	if !h.allowAccountAttempt(c, "account-sign-in", accountSignInLimit, 15*time.Minute) {
		return
	}
	var request SignInRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusUnauthorized, "Email or password is not correct.")
		return
	}
	current, token, expires, err := h.accounts.SignIn(
		c.Request.Context(), string(request.Email), request.Password,
	)
	if errors.Is(err, ErrCredentials) {
		api.Refuse(c, http.StatusUnauthorized, "Email or password is not correct.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not sign in.")
		return
	}
	api.SetSession(c, token, expires)
	c.JSON(http.StatusOK, toAPIAccount(current))
}

func (h *Handlers) BeginDiscord(c *gin.Context) {
	q := api.ReadQuery(c)
	params := BeginDiscordParams{
		Intent:   api.QueryText[BeginDiscordParamsIntent](q, "intent"),
		ReturnTo: api.QueryText[string](q, "returnTo"),
	}
	if q.Refused(c) {
		return
	}
	intent := DiscordSignIn
	if params.Intent != nil && *params.Intent == BeginDiscordParamsIntentAttach {
		intent = DiscordAttach
	}
	token := api.SessionToken(c)
	authorization, err := h.accounts.BeginDiscord(c.Request.Context(), token, intent)
	if errors.Is(err, ErrDiscordUnavailable) {
		api.Refuse(c, http.StatusServiceUnavailable, "Discord sign-in is not available.")
		return
	}
	if errors.Is(err, ErrUnauthorized) {
		api.Refuse(c, http.StatusUnauthorized, "Sign in before attaching Discord.")
		return
	}
	if errors.Is(err, ErrEmailUnverified) {
		api.Refuse(c, http.StatusForbidden, "Verify your email before attaching Discord.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not begin Discord sign-in.")
		return
	}
	setOAuthStateCookie(c, authorization.State, authorization.Expires)
	returnTo := ""
	if intent == DiscordSignIn {
		returnTo = safeInternalPath(params.ReturnTo)
	}
	setOAuthReturnCookie(c, returnTo, authorization.Expires)
	c.Redirect(http.StatusSeeOther, authorization.URL)
}

func (h *Handlers) CompleteDiscord(c *gin.Context) {
	q := api.ReadQuery(c)
	params := CompleteDiscordParams{
		State: api.QueryRequired(q, "state"),
		Code:  api.QueryText[string](q, "code"),
		Error: api.QueryText[string](q, "error"),
	}
	if q.Refused(c) {
		return
	}
	browserState, err := c.Cookie(oauthStateCookieName)
	if err != nil || subtle.ConstantTimeCompare([]byte(browserState), []byte(params.State)) != 1 {
		c.Redirect(http.StatusSeeOther, "/sign-in?discord=failed")
		return
	}
	returnTo := readOAuthReturnCookie(c)
	clearOAuthStateCookie(c)
	clearOAuthReturnCookie(c)
	if params.Error != nil || params.Code == nil {
		c.Redirect(http.StatusSeeOther, discordSignInDestination("cancelled", returnTo))
		return
	}
	completion, err := h.accounts.CompleteDiscord(
		c.Request.Context(), params.State, *params.Code,
	)
	attached := completion.Intent == DiscordAttach
	if err != nil {
		destination := discordSignInDestination("failed", returnTo)
		if attached {
			destination = "/settings?discord=failed"
		}
		switch {
		case errors.Is(err, ErrDiscordEmailConflict):
			if attached {
				destination = "/settings?discord=email-conflict"
			} else {
				destination = discordSignInDestination("email-conflict", returnTo)
			}
		case errors.Is(err, ErrDiscordClaimed):
			destination = "/settings?discord=claimed"
		}
		c.Redirect(http.StatusSeeOther, destination)
		return
	}
	if attached {
		c.Redirect(http.StatusSeeOther, "/settings?discord=attached")
		return
	}
	api.SetSession(c, completion.SessionToken, completion.SessionExpires)
	if returnTo == "" {
		returnTo = "/browse"
	}
	c.Redirect(http.StatusSeeOther, returnTo)
}

func (h *Handlers) SignOut(c *gin.Context) {
	token := api.SessionToken(c)
	if err := h.accounts.SignOut(c.Request.Context(), token); err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not sign out.")
		return
	}
	api.ClearSession(c)
	c.Status(http.StatusNoContent)
}

func (h *Handlers) GetSession(c *gin.Context) {
	current, err := api.Current(c)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the signed-in account.")
		return
	}
	if current == nil {
		c.JSON(http.StatusOK, SessionState{User: nil})
		return
	}
	writer, err := h.blog.IsWriter(c.Request.Context(), current.ID)
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the signed-in account.")
		return
	}
	user := toAPIAccount(*current)
	c.JSON(http.StatusOK, SessionState{User: &user, Writer: writer})
}

func (h *Handlers) VerifyEmail(c *gin.Context) {
	var request VerifyEmailRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Token == "" {
		api.Refuse(c, http.StatusBadRequest, "The verification link is incomplete.")
		return
	}
	verified, err := h.accounts.VerifyEmail(c.Request.Context(), request.Token)
	if err != nil {
		switch {
		case errors.Is(err, ErrVerification):
			api.Refuse(c, http.StatusBadRequest, "This verification link is invalid or has expired.")
		case errors.Is(err, ErrEmailUnavailable):
			api.Refuse(c, http.StatusConflict, "Another account verified this email first.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not verify the email.")
		}
		return
	}
	c.JSON(http.StatusOK, toAPIAccount(verified))
}

func (h *Handlers) RequestPasswordReset(c *gin.Context) {
	if !h.allowAccountAttempt(c, "account-password-reset", accountPasswordResetLimit, time.Hour) {
		return
	}
	var request PasswordResetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the account email as JSON.")
		return
	}
	if err := h.accounts.RequestPasswordReset(c.Request.Context(), string(request.Email)); err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not request a password reset.")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) CompletePasswordReset(c *gin.Context) {
	if !h.allowAccountAttempt(
		c,
		"account-password-reset-complete",
		accountPasswordResetCompleteLimit,
		time.Hour,
	) {
		return
	}
	var request CompletePasswordResetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the reset token and new password as JSON.")
		return
	}
	err := h.accounts.CompletePasswordReset(
		c.Request.Context(), request.Token, request.Password,
	)
	if err != nil {
		var field FieldError
		switch {
		case errors.As(err, &field):
			api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
		case errors.Is(err, ErrPasswordReset):
			api.Refuse(c, http.StatusBadRequest, "This password reset link is invalid or has expired.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not reset the password.")
		}
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) allowAccountAttempt(
	c *gin.Context,
	action string,
	limit int32,
	window time.Duration,
) bool {
	err := h.apps.Throttle(c.Request.Context(), action, api.RequestSource(c), limit, window)
	if err == nil {
		return true
	}
	var limited *connect.RateLimitError
	if errors.As(err, &limited) {
		seconds := int((limited.After + time.Second - 1) / time.Second)
		c.Header("Retry-After", strconv.Itoa(seconds))
		api.Refuse(c, http.StatusTooManyRequests, "Too many attempts. Try again later.")
		return false
	}
	api.Refuse(c, http.StatusInternalServerError, "Could not check the request limit.")
	return false
}

func (h *Handlers) RenameHandle(c *gin.Context) {
	var request RenameHandleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the new handle as JSON.")
		return
	}
	token := api.SessionToken(c)
	updated, err := h.accounts.RenameHandle(c.Request.Context(), token, request.Handle)
	if err != nil {
		var field FieldError
		switch {
		case errors.As(err, &field):
			api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
		case errors.Is(err, ErrUnauthorized):
			api.Refuse(c, http.StatusUnauthorized, "Sign in before changing your handle.")
		case errors.Is(err, ErrEmailUnverified):
			api.Refuse(c, http.StatusForbidden, "Verify your email before changing your handle.")
		case errors.Is(err, ErrHandleUnavailable):
			api.Refuse(c, http.StatusConflict, "That handle is not available.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not change the handle.")
		}
		return
	}
	c.JSON(http.StatusOK, toAPIAccount(updated))
}

func (h *Handlers) ChangeUnverifiedEmail(c *gin.Context) {
	var request ChangeEmailRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the corrected email as JSON.")
		return
	}
	token := api.SessionToken(c)
	updated, err := h.accounts.ChangeUnverifiedEmail(
		c.Request.Context(), token, string(request.Email),
	)
	if err != nil {
		var field FieldError
		switch {
		case errors.As(err, &field):
			api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
		case errors.Is(err, ErrUnauthorized):
			api.Refuse(c, http.StatusUnauthorized, "Sign in before changing your email.")
		case errors.Is(err, ErrEmailUnavailable):
			api.Refuse(c, http.StatusConflict, "An account already uses that email. Sign in instead.")
		case errors.Is(err, ErrEmailVerified):
			api.Refuse(c, http.StatusConflict, "This email is already verified.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not change the email.")
		}
		return
	}
	c.JSON(http.StatusOK, toAPIAccount(updated))
}

func (h *Handlers) SetPassword(c *gin.Context) {
	var request PasswordRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the new password as JSON.")
		return
	}
	token := api.SessionToken(c)
	updated, err := h.accounts.SetPassword(c.Request.Context(), token, request.Password)
	if err != nil {
		var field FieldError
		switch {
		case errors.As(err, &field):
			api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
		case errors.Is(err, ErrUnauthorized):
			api.Refuse(c, http.StatusUnauthorized, "Sign in before setting a password.")
		case errors.Is(err, ErrPasswordAlreadySet):
			api.Refuse(c, http.StatusConflict, "This account already has a password. Use password recovery to replace it.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not set the password.")
		}
		return
	}
	c.JSON(http.StatusOK, toAPIAccount(updated))
}

func (h *Handlers) SetNsfwPreference(c *gin.Context) {
	var request NsfwPreferenceRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil || !request.Preference.Valid() {
		api.Refuse(c, http.StatusBadRequest, "Choose hidden, blurred or shown.")
		return
	}
	token := api.SessionToken(c)
	err := h.accounts.SetNSFWPreference(
		c.Request.Context(), token, NSFWPreference(request.Preference),
	)
	if errors.Is(err, ErrUnauthorized) {
		api.Refuse(c, http.StatusUnauthorized, "Sign in before saving a content preference.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not save the content preference.")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) DetachDiscord(c *gin.Context) {
	token := api.SessionToken(c)
	updated, err := h.accounts.DetachDiscord(c.Request.Context(), token)
	if err != nil {
		switch {
		case errors.Is(err, ErrUnauthorized):
			api.Refuse(c, http.StatusUnauthorized, "Sign in before detaching Discord.")
		case errors.Is(err, ErrLastSignInMethod):
			api.Refuse(c, http.StatusConflict, "Verify an email and set a password before detaching Discord.")
		case errors.Is(err, ErrDiscordNotLinked):
			api.Refuse(c, http.StatusConflict, "Discord is not attached to this account.")
		default:
			api.Refuse(c, http.StatusInternalServerError, "Could not detach Discord.")
		}
		return
	}
	c.JSON(http.StatusOK, toAPIAccount(updated))
}

func (h *Handlers) accountError(c *gin.Context, err error) {
	var field FieldError
	switch {
	case errors.As(err, &field):
		api.RefuseField(c, http.StatusBadRequest, field.Field, field.Message)
	case errors.Is(err, ErrHandleUnavailable):
		api.Refuse(c, http.StatusConflict, "That handle is not available.")
	case errors.Is(err, ErrEmailBelongsDiscord):
		api.Refuse(c, http.StatusConflict, "That email belongs to a Discord account. Sign in with Discord and set a password.")
	case errors.Is(err, ErrEmailUnavailable):
		api.Refuse(c, http.StatusConflict, "An account already uses that email. Sign in instead.")
	default:
		api.Refuse(c, http.StatusInternalServerError, "Could not create the account.")
	}
}

func setOAuthStateCookie(c *gin.Context, state string, expires time.Time) {
	api.SetCookie(c, oauthStateCookieName, state, expires)
}

func clearOAuthStateCookie(c *gin.Context) {
	api.ClearCookie(c, oauthStateCookieName)
}

func setOAuthReturnCookie(c *gin.Context, destination string, expires time.Time) {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(destination))
	api.SetCookie(c, oauthReturnCookieName, encoded, expires)
}

func readOAuthReturnCookie(c *gin.Context) string {
	encoded, err := c.Cookie(oauthReturnCookieName)
	if err != nil || encoded == "" {
		return ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	destination := string(decoded)
	return safeInternalPath(&destination)
}

func clearOAuthReturnCookie(c *gin.Context) {
	api.ClearCookie(c, oauthReturnCookieName)
}

func safeInternalPath(candidate *string) string {
	if candidate == nil || !strings.HasPrefix(*candidate, "/") ||
		strings.HasPrefix(*candidate, "//") || strings.Contains(*candidate, "\\") ||
		strings.IndexFunc(*candidate, func(character rune) bool {
			return character < ' ' || character == '\u007f'
		}) >= 0 {
		return ""
	}
	return *candidate
}

func discordSignInDestination(status, returnTo string) string {
	query := url.Values{"discord": {status}}
	if returnTo != "" {
		query.Set("returnTo", returnTo)
	}
	return "/sign-in?" + query.Encode()
}

func toAPIAccount(value api.Account) Account {
	result := Account{
		Id:            value.ID,
		Handle:        value.Handle,
		EmailVerified: value.EmailVerified,
		DiscordLinked: value.DiscordLinked,
		HasPassword:   value.HasPassword,
		Role:          AccountRole(value.Role),
	}
	result.Email = value.Email
	return result
}
