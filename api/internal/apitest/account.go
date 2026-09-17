package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
)

const CreatorHandle = "moon.creator"

func SignUp(t *testing.T, r http.Handler, email, handle string) *http.Cookie {
	t.Helper()
	rec := SignUpRequest(t, r, email, handle)
	if rec.Code != http.StatusCreated {
		t.Fatalf("sign up %s status = %d, want 201. body: %s", handle, rec.Code, rec.Body.String())
	}
	return rec.Result().Cookies()[0]
}

func SignUpRequest(t *testing.T, r http.Handler, email, handle string) *httptest.ResponseRecorder {
	t.Helper()
	return SendJSON(t, r, http.MethodPost, "/v1/auth/sign-up", `{
		"email":"`+email+`",
		"password":"correct horse battery staple",
		"handle":"`+handle+`"
	}`)
}

func VerifiedSignUp(
	t *testing.T,
	setupRouter *gin.Engine,
	outbox *VerificationOutbox,
	email, handle string,
) *http.Cookie {
	t.Helper()
	session := SignUp(t, setupRouter, email, handle)
	verificationURL, err := url.Parse(outbox.Messages[len(outbox.Messages)-1].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	rec := SendJSON(t, setupRouter, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("verify %s: %d %s", email, rec.Code, rec.Body.String())
	}
	return session
}

type AccountState struct {
	Handle        string  `json:"handle"`
	Email         *string `json:"email"`
	EmailVerified bool    `json:"emailVerified"`
	DiscordLinked bool    `json:"discordLinked"`
	HasPassword   bool    `json:"hasPassword"`
}

func SessionState(t *testing.T, r http.Handler, session *http.Cookie) AccountState {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req.AddCookie(session)
	rec := Send(t, r, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session status = %d, want 200. body: %s", rec.Code, rec.Body.String())
	}
	var state struct {
		User AccountState `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &state); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	return state.User
}

type VerificationMessage struct {
	Address string
	Link    string
}

type VerificationOutbox struct {
	Messages       []VerificationMessage
	PasswordResets []VerificationMessage
}

func (o *VerificationOutbox) SendVerification(_ context.Context, address, link string) error {
	o.Messages = append(o.Messages, VerificationMessage{Address: address, Link: link})
	return nil
}

func (o *VerificationOutbox) SendPasswordReset(_ context.Context, address, link string) error {
	o.PasswordResets = append(o.PasswordResets, VerificationMessage{Address: address, Link: link})
	return nil
}
