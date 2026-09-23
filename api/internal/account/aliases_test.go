package account_test

import (
	"net/http"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestTheNSFWPreferenceStillArrivesUnderItsOldPathAndFieldName(t *testing.T) {
	t.Parallel()
	router, session := harness.NewVerifiedRouter(t)

	saved := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/nsfw-visibility", `{"visibility":"hidden"}`, session,
	))

	if saved.Code != http.StatusNoContent {
		t.Fatalf("save status = %d, want 204: %s", saved.Code, saved.Body.String())
	}
}
