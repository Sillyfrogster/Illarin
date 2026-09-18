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

func sealPresetBlock(
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
		t.Fatalf("write a sealed block payload: %v", err)
	}
	_, err = pool.Exec(t.Context(), `
		insert into migration_preserved_records
			(id, source_table, source_id, work_id, owner_id, payload)
		values ($1, 'preset_sealed_blocks', $2, $3, $4, $5)
	`, uuid.New(), uuid.NewString(), workID, ownerID, payload)
	if err != nil {
		t.Fatalf("preserve a sealed block: %v", err)
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

type sealedStack struct {
	router   *gin.Engine
	session  *http.Cookie
	stranger *http.Cookie
	workID   string
	pool     *pgxpool.Pool
}

func newSealedStack(t *testing.T) sealedStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, _ := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "sealed@example.com", "sealed.creator")
	stranger := apitest.VerifiedSignUp(t, router, outbox, "other@example.com", "other.creator")
	started := apitest.StartPreset(t, router, session, "sillytavern")
	return sealedStack{
		router: router, session: session, stranger: stranger,
		workID: started.ID, pool: pool,
	}
}

func (stack sealedStack) seal(t *testing.T, version, key, content string) {
	t.Helper()
	sealPresetBlock(
		t, stack.pool, stack.workID,
		ownerOfWork(t, stack.pool, stack.workID), version, key, content,
	)
}

func TestAnOwnerExportsEverySealedBlockTheirPresetPreserves(t *testing.T) {
	t.Parallel()
	stack := newSealedStack(t)
	stack.seal(t, "1.0.0", "jailbreak", "The withheld one.")
	stack.seal(t, "1.0.0", "authors_note", "The other one.")
	stack.seal(t, "0.9.0", "jailbreak", "An older take.")

	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+stack.workID+"/sealed", nil,
	), stack.session))
	if response.Code != http.StatusOK {
		t.Fatalf("export sealed content: status = %d: %s", response.Code, response.Body.String())
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
		WorkID string `json:"asset_id"`
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
	for _, sealed := range exported.Blocks {
		order = append(order, sealed.Version+"/"+sealed.Key)
	}
	want := []string{"0.9.0/jailbreak", "1.0.0/authors_note", "1.0.0/jailbreak"}
	for index, expected := range want {
		if order[index] != expected {
			t.Fatalf("export order = %v, want %v", order, want)
		}
	}
	if exported.Blocks[0].Content != "An older take." {
		t.Errorf("a sealed block came back as %q", exported.Blocks[0].Content)
	}
}

func TestSealedContentAnswersNobodyButItsOwner(t *testing.T) {
	t.Parallel()
	stack := newSealedStack(t)
	stack.seal(t, "1.0.0", "jailbreak", "The withheld one.")

	signedOut := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+stack.workID+"/sealed", nil,
	))
	if signedOut.Code != http.StatusUnauthorized {
		t.Errorf("a signed-out reader asked for sealed content and got %d", signedOut.Code)
	}

	stranger := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+stack.workID+"/sealed", nil,
	), stack.stranger))
	if stranger.Code != http.StatusNotFound {
		t.Errorf("another creator asked for sealed content and got %d, want 404", stranger.Code)
	}
	if strings.Contains(stranger.Body.String(), "The withheld one.") {
		t.Error("the refusal carried the sealed content")
	}
}

func TestAnWorkHoldingNothingSealedHasNoExport(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartPreset(t, r, session, "sillytavern")

	response := apitest.Send(t, r, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+started.ID+"/sealed", nil,
	), session))
	if response.Code != http.StatusNotFound {
		t.Errorf("a work with nothing sealed exported %d, want 404", response.Code)
	}
}

func TestTheSealedCountStandsOnlyForTheOwner(t *testing.T) {
	t.Parallel()
	stack := newSealedStack(t)
	stack.seal(t, "1.0.0", "jailbreak", "The withheld one.")

	owner := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+stack.workID, nil,
	), stack.session))
	var page struct {
		SealedBlocks *int `json:"sealedBlocks"`
	}
	if err := json.Unmarshal(owner.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the owner's page: %v", err)
	}
	if page.SealedBlocks == nil || *page.SealedBlocks != 1 {
		t.Errorf("the owner's page counts %v sealed blocks, want 1", page.SealedBlocks)
	}
	if strings.Contains(owner.Body.String(), "The withheld one.") {
		t.Error("the page rendered sealed content")
	}

	stranger := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/assets/"+stack.workID, nil,
	), stack.stranger))
	if strings.Contains(stranger.Body.String(), "sealedBlocks") {
		t.Error("a stranger's page says the work is withholding something")
	}
}
