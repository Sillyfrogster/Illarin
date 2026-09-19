package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAnIntegrationStillArrivesAndAnswersUnderTheOldNameForItsType(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	creator := apitest.VerifiedSignUp(t, stack.router, stack.outbox, "creator@example.com", "work.creator")

	response := stack.integrationRequest(t, creator, http.MethodPost, integrationsPath,
		fmt.Sprintf(`{"name":"Creator updates","kind":"webhook","address":%q}`, stack.to.address()))

	if response.Code != http.StatusCreated {
		t.Fatalf("create with the old name = %d, want 201: %s", response.Code, response.Body.String())
	}
	var made struct {
		Integration map[string]any `json:"integration"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode the integration: %v", err)
	}
	if made.Integration["kind"] != "webhook" || made.Integration["type"] != "webhook" {
		t.Fatalf("integration = %v, want kind and type both webhook", made.Integration)
	}
}

func TestACreatorsIntegrationStillAnswersUnderItsOldPathAndFieldNames(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	creator := apitest.VerifiedSignUp(t, stack.router, stack.outbox, "old.names@example.com", "old.names")
	const oldPath = "/v1/account/update-destinations"

	made := stack.integrationRequest(t, creator, http.MethodPost, oldPath,
		fmt.Sprintf(`{"name":"Creator updates","type":"webhook","address":%q}`, stack.to.address()))
	if made.Code != http.StatusCreated {
		t.Fatalf("create on the old path = %d, want 201: %s", made.Code, made.Body.String())
	}
	var added struct {
		Destination map[string]any `json:"destination"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &added); err != nil {
		t.Fatalf("decode what the old name returned: %v", err)
	}
	if added.Destination["id"] == nil {
		t.Fatalf("the answer carries no destination: %s", made.Body.String())
	}

	listed := stack.integrationRequest(t, creator, http.MethodGet, oldPath, "")
	var held struct {
		Destinations []map[string]any `json:"destinations"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &held); err != nil {
		t.Fatalf("decode the old listing: %v", err)
	}
	if len(held.Destinations) != 1 {
		t.Fatalf("the old listing held %d destinations, want 1: %s", len(held.Destinations), listed.Body.String())
	}
}

func TestTheBlogsIntegrationsStillAnswerUnderTheirOldPathAndFieldNames(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	made := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/destinations",
		fmt.Sprintf(`{"name":"Old names","address":%q,"events":["publication.post.published.v1"]}`,
			stack.to.address()),
	), stack.authority))
	if made.Code != http.StatusCreated {
		t.Fatalf("create on the old path = %d, want 201: %s", made.Code, made.Body.String())
	}
	var added struct {
		Destination struct {
			Id     string   `json:"id"`
			Events []string `json:"events"`
		} `json:"destination"`
	}
	if err := json.Unmarshal(made.Body.Bytes(), &added); err != nil {
		t.Fatalf("decode what the old name returned: %v", err)
	}
	if added.Destination.Id == "" || len(added.Destination.Events) != 1 {
		t.Fatalf("the answer lost its old names: %s", made.Body.String())
	}

	sent := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/deliveries", nil,
	), stack.authority))
	var queued struct {
		Deliveries []map[string]any `json:"deliveries"`
	}
	if err := json.Unmarshal(sent.Body.Bytes(), &queued); err != nil {
		t.Fatalf("decode the old delivery listing: %v", err)
	}
	if queued.Deliveries == nil {
		t.Fatalf("the old delivery listing is gone: %s", sent.Body.String())
	}
}
