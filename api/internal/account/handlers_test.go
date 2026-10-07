package account

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestSignUpLogsTheFailureWithoutPrivateDetails(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })

	graph := microsoftResponseError("send Microsoft 365 message", &http.Response{
		StatusCode: http.StatusForbidden,
		Body:       io.NopCloser(strings.NewReader("private@example.com")),
	})
	for _, test := range []struct {
		name, stage, cause string
		err                error
	}{
		{"database", "create account", "SQLSTATE 23505", &pgconn.PgError{Code: "23505", Message: "private@example.com"}},
		{"smtp", "send verification", "SMTP 535", &textproto.Error{Code: 535, Msg: "private@example.com"}},
		{"microsoft", "send verification", "HTTP 403", graph},
		{"timeout", "send verification", "context deadline exceeded", context.DeadlineExceeded},
	} {
		t.Run(test.name, func(t *testing.T) {
			output.Reset()
			response := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(response)
			(&Handlers{}).accountError(c, fmt.Errorf("%s: %w", test.stage, test.err))
			if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), test.cause) {
				t.Fatalf("unexpected public response: %d %s", response.Code, response.Body.String())
			}
			if !strings.Contains(output.String(), test.stage) || !strings.Contains(output.String(), test.cause) {
				t.Fatalf("missing failure diagnostic: %q", output.String())
			}
			if strings.Contains(output.String(), "private@example.com") {
				t.Fatal("diagnostic contains private details")
			}
		})
	}
}
