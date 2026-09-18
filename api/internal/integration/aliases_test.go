package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAnIntegrationStillArrivesAndAnswersUnderTheOldNameForItsType(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	creator := apitest.VerifiedSignUp(t, stack.router, stack.outbox, "creator@example.com", "work.creator")

	response := stack.updateDestinationRequest(t, creator, http.MethodPost, updateDestinationsPath,
		fmt.Sprintf(`{"name":"Creator updates","kind":"webhook","address":%q}`, stack.to.address()))

	if response.Code != http.StatusCreated {
		t.Fatalf("create with the old name = %d, want 201: %s", response.Code, response.Body.String())
	}
	var made struct {
		Destination map[string]any `json:"destination"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode the integration: %v", err)
	}
	if made.Destination["kind"] != "webhook" || made.Destination["type"] != "webhook" {
		t.Fatalf("integration = %v, want kind and type both webhook", made.Destination)
	}
}
