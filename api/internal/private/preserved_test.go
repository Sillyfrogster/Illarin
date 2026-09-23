package private_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func preservePresetBlock(
	t *testing.T,
	pool *pgxpool.Pool,
	workID string,
	ownerID uuid.UUID,
	version, key, content string,
) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"id": uuid.NewString(), "preset_id": workID, "version": version,
		"block_key": key, "content": content,
	})
	if err != nil {
		t.Fatalf("write a preserved prompt payload: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		insert into migration_preserved_records
			(id, source_table, source_id, work_id, owner_id, payload)
		values ($1, 'preserved_prompts', $2, $3, $4, $5)
	`, uuid.New(), uuid.NewString(), workID, ownerID, payload)
	if err != nil {
		t.Fatalf("preserve a preserved prompt: %v", err)
	}
}

func ownerOfWork(t *testing.T, pool *pgxpool.Pool, workID string) uuid.UUID {
	t.Helper()
	var ownerID uuid.UUID
	if err := pool.QueryRow(t.Context(),
		`select owner_id from works where id = $1`, workID).Scan(&ownerID); err != nil {
		t.Fatalf("read the work owner: %v", err)
	}
	return ownerID
}

type preservedStack struct {
	router   *gin.Engine
	session  *http.Cookie
	stranger *http.Cookie
	workID   string
	pool     *pgxpool.Pool
}

func newPreservedStack(t *testing.T) preservedStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, _ := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "preserved@example.com", "preserved.creator")
	stranger := apitest.VerifiedSignUp(t, router, outbox, "other@example.com", "other.creator")
	started := apitest.StartPreset(t, router, session, "sillytavern")
	return preservedStack{
		router: router, session: session, stranger: stranger,
		workID: started.ID, pool: pool,
	}
}

func (stack preservedStack) preserve(t *testing.T, version, key, content string) {
	t.Helper()
	preservePresetBlock(
		t, stack.pool, stack.workID,
		ownerOfWork(t, stack.pool, stack.workID), version, key, content,
	)
}

func TestAnOwnerExportsEveryPreservedPromptTheirPresetHolds(t *testing.T) {
	t.Parallel()
	stack := newPreservedStack(t)
	stack.preserve(t, "1.0.0", "jailbreak", "The taken down one.")
	stack.preserve(t, "1.0.0", "authors_note", "The other one.")
	stack.preserve(t, "0.9.0", "jailbreak", "An older take.")

	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID+"/preserved-prompts", nil,
	), stack.session))
	if response.Code != http.StatusOK {
		t.Fatalf("export preserved prompts: status = %d: %s", response.Code, response.Body.String())
	}
	if disposition := response.Header().Get("Content-Disposition"); !strings.Contains(
		disposition, "attachment",
	) || !strings.Contains(disposition, ".json") {
		t.Errorf("Content-Disposition = %q, want a named json attachment", disposition)
	}
	if cache := response.Header().Get("Cache-Control"); cache != "private, no-store" {
		t.Errorf("Cache-Control = %q, want the export uncached", cache)
	}

	var exported struct {
		WorkID string `json:"work_id"`
		Blocks []struct {
			Version string `json:"version"`
			Key     string `json:"block_key"`
			Content string `json:"content"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &exported); err != nil {
		t.Fatalf("decode the export: %v", err)
	}
	if exported.WorkID != stack.workID {
		t.Errorf("the export names work %s, want %s", exported.WorkID, stack.workID)
	}
	if len(exported.Blocks) != 3 {
		t.Fatalf("the export holds %d blocks, want all three", len(exported.Blocks))
	}
	order := make([]string, 0, 3)
	for _, preserved := range exported.Blocks {
		order = append(order, preserved.Version+"/"+preserved.Key)
	}
	want := []string{"0.9.0/jailbreak", "1.0.0/authors_note", "1.0.0/jailbreak"}
	for index, expected := range want {
		if order[index] != expected {
			t.Fatalf("export order = %v, want %v", order, want)
		}
	}
	if exported.Blocks[0].Content != "An older take." {
		t.Errorf("a preserved prompt came back as %q", exported.Blocks[0].Content)
	}
}

func TestPreservedPromptsAnswerNobodyButTheirOwner(t *testing.T) {
	t.Parallel()
	stack := newPreservedStack(t)
	stack.preserve(t, "1.0.0", "jailbreak", "The taken down one.")

	signedOut := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID+"/preserved-prompts", nil,
	))
	if signedOut.Code != http.StatusUnauthorized {
		t.Errorf("a signed-out reader asked for preserved prompts and got %d", signedOut.Code)
	}

	stranger := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID+"/preserved-prompts", nil,
	), stack.stranger))
	if stranger.Code != http.StatusNotFound {
		t.Errorf("another creator asked for preserved prompts and got %d, want 404", stranger.Code)
	}
	if strings.Contains(stranger.Body.String(), "The taken down one.") {
		t.Error("the refusal carried the preserved prompts")
	}
}

func TestAWorkHoldingNoPreservedPromptsHasNoExport(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartPreset(t, r, session, "sillytavern")

	response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/preserved-prompts", nil,
	), session))
	if response.Code != http.StatusNotFound {
		t.Errorf("a work with no preserved prompts exported %d, want 404", response.Code)
	}
}

func TestThePreservedPromptCountStandsOnlyForTheOwner(t *testing.T) {
	t.Parallel()
	stack := newPreservedStack(t)
	stack.preserve(t, "1.0.0", "jailbreak", "The taken down one.")

	owner := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID, nil,
	), stack.session))
	var page struct {
		PreservedPrompts *int `json:"preservedPrompts"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the owner's page: %v", err)
	}
	if page.PreservedPrompts == nil || *page.PreservedPrompts != 1 {
		t.Errorf("the owner's page counts %v preserved prompts, want 1", page.PreservedPrompts)
	}
	if strings.Contains(owner.Body.String(), "The taken down one.") {
		t.Error("the page rendered preserved prompts")
	}

	stranger := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+stack.workID, nil,
	), stack.stranger))
	if strings.Contains(stranger.Body.String(), "preservedPrompts") {
		t.Error("a stranger's page says the work is hiding something")
	}
}
