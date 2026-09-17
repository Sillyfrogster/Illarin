package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/gin-gonic/gin"
)

var harness = apitest.Harness{Register: register}

func register(r *gin.Engine, s apitest.Services, d api.Deadlines) error {
	handlers := NewHandlers(
		s.Assets, s.Works, s.Blocks, s.Accounts, s.Links, s.Deliveries, s.Publications,
		s.UpdateDestinations, s.Notifications, s.MaxUploadBytes,
	)
	return Register(r, handlers, d, apitest.Ready)
}

func TestFormatRegistryInvariantIsNotAnUploaderRefusal(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	new(Handlers).refuse(ctx, format.ErrConflictingClaims)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "claim") {
		t.Fatalf("response exposed registry details: %s", recorder.Body.String())
	}
}
