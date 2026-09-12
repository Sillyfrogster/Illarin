package asset

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBundledLumiverseScriptsChangeThroughAJSONReplacement(t *testing.T) {
	const initial = `{
		"schemaVersion":1,
		"name":"Scripted preset",
		"blocks":[{
			"id":"prompt","name":"Prompt","role":"system",
			"content":"Stay in character.","enabled":true
		}],
		"extensions":{"regex_scripts":[{
			"name":"Formatter","find_regex":"/before/g","replace_string":"after",
			"placement":["ai_output"],"target":["display"],"disabled":false
		}]}
	}`

	svc, pool := newTestServiceWithRegistry(t, registryWithModule(t, preset.LumiverseModule{}))
	owner := revisionOwner(t, svc, "bundled.scripts.update")
	created := ingestOne(t, svc, owner, "preset.json", []byte(initial))
	publishImported(t, svc, owner, created)
	generation := contentGeneration(t, pool, created.ID)

	replacement := strings.Replace(initial, "/before/g", "/updated/g", 1)
	operation := addRevision(t, svc, owner, created.ID, "preset.json", []byte(replacement))
	if operation.Status != IngestSuccess {
		t.Fatalf("replacement = %+v, want success", operation)
	}
	working, err := svc.WorkingCopy(t.Context(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	scriptBlock := blockFor(t, working.Blocks, block.PresetScripts)
	if len(scriptBlock.Elements) != 1 {
		t.Fatalf("script block elements = %d, want 1", len(scriptBlock.Elements))
	}
	scripts := scriptBlock.Elements[0].Content.(block.ScriptList).Scripts
	if len(scripts) != 1 || scripts[0].Find != "/updated/g" {
		t.Fatalf("working scripts = %+v", scripts)
	}
	updated, _, err := svc.PublishUpdate(t.Context(), UpdateRequest{
		OwnerID: owner, AssetID: created.ID, Summary: "Updated the bundled script",
	}, currentCandidate(t, svc, created.ID))
	if err != nil || !updated.ContentChanged {
		t.Fatalf("publish JSON replacement = %+v, %v", updated, err)
	}
	if got := contentGeneration(t, pool, created.ID); got != generation+1 {
		t.Fatalf("content generation = %d, want %d", got, generation+1)
	}
}

func TestAnIdenticalReuploadCannotPublishAnUpdate(t *testing.T) {
	parsed := format.Parsed{Kind: "character", Format: "replacing", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: uuid.New(), Text: "Hello"}}}},
		}}
	svc, pool := newTestServiceWithRegistry(t, registryWithModule(t, replacingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "unchanged.upload")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)
	generation := contentGeneration(t, pool, created.ID)
	parsed.Elements[1].Content = block.TextSet{Texts: []block.TextItem{{ID: uuid.New(), Text: "Hello"}}}
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(t.Context(), RevisionInput{OwnerID: owner, AssetID: created.ID,
		Filename: "wren.json", File: bytes.NewBufferString(`{"payload":true}`)}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(t.Context()); err != nil || !processed {
		t.Fatalf("ingest = %v, %v", processed, err)
	}
	if _, err := svc.AcceptReplacement(t.Context(), owner, created.ID, operation.ID, candidate, nil, false); err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.PublishUpdate(t.Context(), UpdateRequest{OwnerID: owner, AssetID: created.ID, Summary: "No change"}, currentCandidate(t, svc, created.ID))
	if !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("identical upload = %v", err)
	}
	if recordedUpdates(t, pool, created.ID) != 1 || contentGeneration(t, pool, created.ID) != generation {
		t.Fatal("identical upload published a version")
	}
	saveDescription(t, svc, owner, created.ID, pool, "A real change")
	updated, _, err := svc.PublishUpdate(t.Context(), UpdateRequest{OwnerID: owner, AssetID: created.ID, Summary: "Changed description"}, currentCandidate(t, svc, created.ID))
	if err != nil || !updated.ContentChanged {
		t.Fatalf("real change = %+v, %v", updated, err)
	}
}

func publishedAsset(t *testing.T, svc *Service, pool *pgxpool.Pool, handle string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := revisionOwner(t, svc, handle)
	id, err := svc.StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatalf("start a draft: %v", err)
	}
	saveDescription(t, svc, owner, id, pool, "Published description")
	saveGreeting(t, svc, owner, id, pool, "Published greeting")
	adult := false
	if err := svc.SetIdentity(context.Background(), Identity{
		OwnerID: owner, AssetID: id, Name: "Published name", IsNSFW: &adult,
	}, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("save the header: %v", err)
	}
	if _, err := svc.Publish(context.Background(), owner, id, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the asset: %v", err)
	}
	return owner, id
}

func recordedUpdates(t *testing.T, pool *pgxpool.Pool, assetID uuid.UUID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from asset_snapshots where asset_id = $1`, assetID).Scan(&count); err != nil {
		t.Fatalf("count the recorded updates: %v", err)
	}
	return count
}

func TestPublishingAnUpdateRecordsTheReviewedCandidateAndMovesTheGeneration(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "update.owner")
	var madeAt, kept any
	if err := pool.QueryRow(ctx, `select created_at, id from assets where id = $1`, id).Scan(&madeAt, &kept); err != nil {
		t.Fatal(err)
	}
	generation := contentGeneration(t, pool, id)
	saveDescription(t, svc, owner, id, pool, "Second description")

	recorded, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Rewrote the description",
		Notes: "The longer explanation.", VersionLabel: "v2",
	}, currentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	if recorded.Number != 2 || recorded.Summary != "Rewrote the description" ||
		recorded.Notes != "The longer explanation." || recorded.VersionLabel != "v2" ||
		!recorded.ContentChanged || recorded.RecordedAt.IsZero() {
		t.Fatalf("recorded update = %+v", recorded)
	}
	if got := contentGeneration(t, pool, id); got != generation+1 {
		t.Fatalf("content generation = %d, want %d", got, generation+1)
	}
	page, err := svc.Detail(ctx, id, nil, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	core := blockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Second description" {
		t.Fatal("readers still have the earlier description")
	}
	exported, err := svc.OpenExport(ctx, id, nil, "test_opaque", nil)
	if err != nil || !strings.Contains(string(exported.Body), "Second description") {
		t.Fatalf("published download = %q, error = %v", exported.Body, err)
	}
	var stable bool
	err = pool.QueryRow(ctx, `select created_at = $2 and id = $3 and content_generation = $4
		from assets where id = $1`, id, madeAt, kept, recorded.ContentGeneration).Scan(&stable)
	if err != nil || !stable {
		t.Fatalf("asset identity and generation stable = %v, error = %v", stable, err)
	}
}

func TestAnUpdateWithoutAChangeOrASummaryIsRefused(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "unchanged.owner")

	_, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "  ",
	}, currentCandidate(t, svc, id))
	if !errors.Is(err, ErrSummaryRequired) {
		t.Fatalf("publishing without a summary = %v, want ErrSummaryRequired", err)
	}
	_, _, err = svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Nothing has changed",
		Notes: "But the notes are new.",
	}, currentCandidate(t, svc, id))
	if !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("publishing an unchanged candidate = %v, want ErrNothingToPublish", err)
	}
	saveDescription(t, svc, owner, id, pool, "Second description")
	if _, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Rewrote the description",
	}, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	_, _, err = svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Same content, fresh notes",
	}, currentCandidate(t, svc, id))
	if !errors.Is(err, ErrNothingToPublish) {
		t.Fatalf("republishing the same candidate = %v, want ErrNothingToPublish", err)
	}
	if got := recordedUpdates(t, pool, id); got != 2 {
		t.Fatalf("recorded updates = %d, want 2", got)
	}
}

func TestAPresentationChangePublishesWithoutMovingTheGeneration(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "presentation.owner")
	generation := contentGeneration(t, pool, id)
	arrangement := make([]BlockArrangement, 0)
	for _, holder := range draftBlocks(t, pool, id) {
		arrangement = append(arrangement, BlockArrangement{
			ID: holder.ID, Hidden: holder.Definition == "character_core", Width: holder.Width,
		})
	}
	if _, err := svc.ArrangeBlocks(ctx, owner, id, arrangement, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("rearrange the page: %v", err)
	}

	recorded, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Hid the character block",
	}, currentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	if recorded.Number != 2 || recorded.ContentChanged {
		t.Fatalf("recorded update = %+v, want a second update that left the file alone", recorded)
	}
	if got := contentGeneration(t, pool, id); got != generation {
		t.Fatalf("content generation = %d, want %d", got, generation)
	}
}

func TestAnUploadWaitingForADecisionRefusesPublicationAndKeepsThePublicAsset(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "waiting.owner")
	saveDescription(t, svc, owner, id, pool, "Second description")
	stored, err := svc.store.Put(ctx, strings.NewReader("synthetic replacement upload"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		insert into ingest_operations (id, owner_id, blob_id, filename, status,
		    target_asset_id, replacement_preview)
		values ($1, $2, $3, 'replacement.json', 'preview', $4, '{}'::jsonb)
	`, uuid.New(), owner, stored.ID, id)
	if err != nil {
		t.Fatal(err)
	}

	_, items, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Rewrote the description",
	}, currentCandidate(t, svc, id))
	if !errors.Is(err, ErrPublishFloor) {
		t.Fatalf("publishing over a waiting upload = %v, want ErrPublishFloor", err)
	}
	for _, item := range items {
		if item.ID == uploadRequirement && item.Met {
			t.Fatal("the waiting upload was reported as reviewed")
		}
	}
	if got := recordedUpdates(t, pool, id); got != 1 {
		t.Fatalf("recorded updates = %d, want the published version alone", got)
	}
	page, err := svc.Detail(ctx, id, nil, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	core := blockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Published description" {
		t.Fatal("the refused publication reached readers")
	}
}

func TestARefusedAnnouncementRollsTheWholePublicationBack(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	refused := errors.New("delivery refused this update")
	svc.OnUpdatePublished(func(context.Context, pgx.Tx, Update, UpdateAnnouncement) error { return refused })
	owner, id := publishedAsset(t, svc, pool, "rollback.owner")
	generation := contentGeneration(t, pool, id)
	saveDescription(t, svc, owner, id, pool, "Second description")
	version := currentCandidate(t, svc, id).Version

	_, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Rewrote the description",
	}, &Candidate{Version: version})
	if !errors.Is(err, refused) {
		t.Fatalf("publishing with a refused announcement = %v", err)
	}
	if got := recordedUpdates(t, pool, id); got != 1 {
		t.Fatalf("recorded updates = %d, want the published version alone", got)
	}
	if got := contentGeneration(t, pool, id); got != generation {
		t.Fatalf("content generation = %d, want %d", got, generation)
	}
	if got := currentCandidate(t, svc, id).Version; got != version {
		t.Fatalf("working-copy version = %d, want %d", got, version)
	}
}

func TestSimultaneousPublicationRecordsOneUpdate(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "simultaneous.owner")
	saveDescription(t, svc, owner, id, pool, "Second description")
	version := currentCandidate(t, svc, id).Version

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, _, err := svc.PublishUpdate(ctx, UpdateRequest{
				OwnerID: owner, AssetID: id, Summary: "Rewrote the description",
			}, &Candidate{Version: version})
			results <- err
		}()
	}
	published := 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			published++
		case errors.Is(err, ErrNothingToPublish):
		default:
			var conflict *VersionConflict
			if !errors.As(err, &conflict) {
				t.Fatalf("second publication = %v, want a conflict", err)
			}
		}
	}
	if published != 1 {
		t.Fatalf("publications that succeeded = %d, want 1", published)
	}
	if got := recordedUpdates(t, pool, id); got != 2 {
		t.Fatalf("recorded updates = %d, want 2", got)
	}
}

func TestRestoringEarlierContentPublishesAFurtherUpdate(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := publishedAsset(t, svc, pool, "restore.owner")
	generation := contentGeneration(t, pool, id)
	saveDescription(t, svc, owner, id, pool, "Second description")
	if _, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Rewrote the description", VersionLabel: "the same label",
	}, currentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the second version: %v", err)
	}
	saveDescription(t, svc, owner, id, pool, "Published description")

	recorded, _, err := svc.PublishUpdate(ctx, UpdateRequest{
		OwnerID: owner, AssetID: id, Summary: "Put the first description back",
		VersionLabel: "the same label",
	}, currentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the restored version: %v", err)
	}
	if recorded.Number != 3 || !recorded.ContentChanged || recorded.VersionLabel != "the same label" {
		t.Fatalf("restored update = %+v, want a third update that moved the file", recorded)
	}
	if got := contentGeneration(t, pool, id); got != generation+2 {
		t.Fatalf("content generation = %d, want %d", got, generation+2)
	}
	page, err := svc.Detail(ctx, id, nil, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	core := blockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Published description" {
		t.Fatal("the restored description did not reach readers")
	}
	if got := recordedUpdates(t, pool, id); got != 3 {
		t.Fatalf("recorded updates = %d, want 3", got)
	}
}
