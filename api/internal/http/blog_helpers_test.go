package http

import (
	"context"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type publicationStack struct {
	router    *gin.Engine
	pool      *pgxpool.Pool
	handlers  apitest.Services
	outbox    *apitest.VerificationOutbox
	authority *http.Cookie
}

func (s publicationStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func holdsAuthority(t *testing.T, pool *pgxpool.Pool, handle string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into publication_authorities (user_id)
		select id from users where username = $1
	`, handle)
	if err != nil {
		t.Fatalf("assign publication authority: %v", err)
	}
}

func setRole(t *testing.T, pool *pgxpool.Pool, handle, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		update users set role = $2 where username = $1
	`, handle, role); err != nil {
		t.Fatalf("set %s role: %v", handle, err)
	}
}

func jsonRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}
