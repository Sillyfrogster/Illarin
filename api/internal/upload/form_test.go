package upload

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/gin-gonic/gin"
)

func TestFormatRegistryInvariantIsNotAnUploaderRefusal(t *testing.T) {
	t.Parallel()
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	RefuseFile(ctx, format.ErrConflictingClaims, 0)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "claim") {
		t.Fatalf("response exposed registry details: %s", recorder.Body.String())
	}
}
