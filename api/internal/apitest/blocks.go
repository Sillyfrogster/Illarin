package apitest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func AddBlock(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	definition string,
	elementType string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"definition": definition, "elementType": elementType,
	})
	if err != nil {
		t.Fatalf("encode the block to add: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPost, "/v1/works/"+workID+"/blocks", strings.NewReader(string(body)),
	)
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

func AddedBlock(t *testing.T, response *httptest.ResponseRecorder) StartedBlock {
	t.Helper()
	if response.Code != http.StatusCreated {
		t.Fatalf("add a block: status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var added StartedBlock
	if err := json.Unmarshal(response.Body.Bytes(), &added); err != nil {
		t.Fatalf("decode the new block: %v", err)
	}
	return added
}

type ArrangedBlock struct {
	ID     string `json:"id"`
	Hidden bool   `json:"hidden"`
	Width  string `json:"width"`
}

func ArrangeBlocks(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
	blocks []ArrangedBlock,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{"blocks": blocks})
	if err != nil {
		t.Fatalf("encode arrangement: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPut, "/v1/works/"+workID+"/blocks", strings.NewReader(string(body)),
	)
	request.Header.Set("Content-Type", "application/json")
	return Send(t, r, Authorized(request, session))
}

func FetchStartedWork(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workID string,
) StartedWork {
	t.Helper()
	response := Send(t, r, Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+workID+"?draftedChanges=true", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read saved asset status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var saved StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &saved); err != nil {
		t.Fatalf("decode saved asset: %v", err)
	}
	return saved
}

func ProtectedCounts(t *testing.T, pool *pgxpool.Pool, workID string) (int, int) {
	t.Helper()
	var payloads, policies int
	err := pool.QueryRow(context.Background(), `
		SELECT
			(SELECT count(*) FROM protected_content WHERE work_id = $1),
			(SELECT count(*) FROM protected_delivery_apps WHERE work_id = $1)
	`, workID).Scan(&payloads, &policies)
	if err != nil {
		t.Fatalf("count protected rows: %v", err)
	}
	return payloads, policies
}
