package http

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func insertEmptyGallery(t *testing.T, pool *pgxpool.Pool, assetID string) string {
	t.Helper()
	id := uuid.New()
	elementID := uuid.New()
	elements := `[{"id":"` + elementID.String() + `","type":"image_set","role":"gallery","slot":"main","version":1,"options":{"itemSize":"medium"},"content":{"images":[]}}]`
	if _, err := pool.Exec(t.Context(), `
		insert into asset_blocks
		  (id, asset_id, definition, position, hidden, layout, width, elements)
		values ($1, $2, 'gallery', 2, false, 'single', 'half', $3)
	`, id, assetID, elements); err != nil {
		t.Fatalf("insert gallery: %v", err)
	}
	return id.String()
}

func TestCreatorReordersAndHidesBlocksAsOneArrangement(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	core := apitest.BlockNamed(t, started.Blocks, "character_core")
	messages := apitest.BlockNamed(t, started.Blocks, "messages")

	response := apitest.ArrangeBlocks(t, r, session, started.ID, []apitest.ArrangedBlock{
		{ID: messages.ID, Width: "half"},
		{ID: core.ID, Hidden: true, Width: "half"},
	})

	if response.Code != http.StatusOK {
		t.Fatalf("arrange status = %d, want 200: %s", response.Code, response.Body.String())
	}
	saved := apitest.FetchStartedAsset(t, r, session, started.ID)
	if saved.Blocks[0].ID != messages.ID || saved.Blocks[0].Position != 0 || saved.Blocks[0].Width != "half" {
		t.Errorf("first arranged block = %+v, want Messages at half width", saved.Blocks[0])
	}
	if saved.Blocks[1].ID != core.ID || saved.Blocks[1].Position != 1 || !saved.Blocks[1].Hidden {
		t.Errorf("second arranged block = %+v, want hidden character core", saved.Blocks[1])
	}
}

func TestArrangementRefusesToHideTheAlwaysShownBlock(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	core := apitest.BlockNamed(t, started.Blocks, "character_core")
	messages := apitest.BlockNamed(t, started.Blocks, "messages")

	response := apitest.ArrangeBlocks(t, r, session, started.ID, []apitest.ArrangedBlock{
		{ID: core.ID, Width: core.Width},
		{ID: messages.ID, Hidden: true, Width: messages.Width},
	})

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "always shown") {
		t.Fatalf("hide Messages = %d, want an always-shown refusal: %s", response.Code, response.Body.String())
	}
}

func TestArrangementRequiresEveryCurrentBlockExactlyOnce(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	core := apitest.BlockNamed(t, started.Blocks, "character_core")

	response := apitest.ArrangeBlocks(t, r, session, started.ID, []apitest.ArrangedBlock{
		{ID: core.ID, Width: core.Width},
		{ID: core.ID, Width: core.Width},
	})

	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "each block once") {
		t.Fatalf("duplicate arrangement = %d, want exact-membership refusal: %s", response.Code, response.Body.String())
	}
}

func TestCreatorRemovesAnOptionalBlockAndRequiredBlocksStay(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	galleryID := insertEmptyGallery(t, pool, started.ID)

	request := httptest.NewRequest(
		http.MethodDelete, "/v1/assets/"+started.ID+"/blocks/"+galleryID, nil,
	)
	response := apitest.Send(t, r, apitest.Authorized(request, session))
	if response.Code != http.StatusNoContent {
		t.Fatalf("remove Gallery status = %d, want 204: %s", response.Code, response.Body.String())
	}
	saved := apitest.FetchStartedAsset(t, r, session, started.ID)
	if len(saved.Blocks) != 2 || saved.Blocks[0].Position != 0 || saved.Blocks[1].Position != 1 {
		t.Errorf("blocks after remove = %+v, want two required blocks in gapless order", saved.Blocks)
	}

	core := apitest.BlockNamed(t, saved.Blocks, "character_core")
	request = httptest.NewRequest(
		http.MethodDelete, "/v1/assets/"+started.ID+"/blocks/"+core.ID, nil,
	)
	response = apitest.Send(t, r, apitest.Authorized(request, session))
	if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), "required") {
		t.Fatalf("remove required block = %d, want required refusal: %s", response.Code, response.Body.String())
	}
}

func TestSavingAnOptionalBlockEmptyKeepsItUntilExplicitRemoval(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	insertEmptyGallery(t, pool, started.ID)
	gallery := apitest.BlockNamed(t, apitest.FetchStartedAsset(t, r, session, started.ID).Blocks, "gallery")

	update := apitest.EditableBlock(gallery)
	update.Width = "full"
	response := apitest.SaveBlock(t, r, session, started.ID, gallery.ID, update)

	if response.Code != http.StatusOK {
		t.Fatalf("save empty Gallery status = %d, want 200: %s", response.Code, response.Body.String())
	}
	saved := apitest.FetchStartedAsset(t, r, session, started.ID)
	kept := apitest.BlockNamed(t, saved.Blocks, "gallery")
	if len(saved.Blocks) != 3 || kept.ID != gallery.ID || kept.Width != "full" {
		t.Errorf("blocks after empty save = %+v, want the full-width Gallery kept", saved.Blocks)
	}
}

func TestCreatorMovesUnpinnedContentBeforeRemovingItsBlock(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)
	galleryID := insertEmptyGallery(t, pool, started.ID)
	messagesBlock := apitest.BlockNamed(t, started.Blocks, "messages")
	messages := apitest.EditableBlock(messagesBlock)
	messages.Layout = "stack-3"
	messages.Elements[0].Slot = "top"
	messages.Elements[1].Slot = "middle"
	response := apitest.SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages)
	if response.Code != http.StatusOK {
		t.Fatalf("prepare destination status = %d, want 200: %s", response.Code, response.Body.String())
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/assets/"+started.ID+"/blocks/"+galleryID+"/move-and-remove",
		strings.NewReader(`{"destinationBlockId":"`+messagesBlock.ID+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response = apitest.Send(t, r, apitest.Authorized(request, session))

	if response.Code != http.StatusOK {
		t.Fatalf("move Gallery content status = %d, want 200: %s", response.Code, response.Body.String())
	}
	saved := apitest.FetchStartedAsset(t, r, session, started.ID)
	if len(saved.Blocks) != 2 {
		t.Fatalf("blocks after move = %d, want the source removed", len(saved.Blocks))
	}
	messagesBlock = apitest.BlockNamed(t, saved.Blocks, "messages")
	if len(messagesBlock.Elements) != 3 || messagesBlock.Elements[2].Role != "gallery" || messagesBlock.Elements[2].Slot != "bottom" {
		t.Errorf("Messages after move = %+v, want Gallery in the free bottom slot", messagesBlock.Elements)
	}
}
