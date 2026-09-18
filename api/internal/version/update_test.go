package version_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/upload"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestBundledLumiverseScriptsChangeThroughAJSONReplacement(t *testing.T) {
	t.Parallel()
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

	svc, pool := apitest.WorksWithRegistry(t, apitest.RegistryWith(t, preset.LumiverseModule{}))
	owner := apitest.Owner(t, svc, "bundled.scripts.update")
	created := apitest.IngestOne(t, svc, owner, "preset.json", []byte(initial))
	apitest.PublishImported(t, svc, owner, created)
	generation := apitest.ContentGeneration(t, pool, created.ID.String())

	replacement := strings.Replace(initial, "/before/g", "/updated/g", 1)
	operation := apitest.AddRevision(t, svc, owner, created.ID, "preset.json", []byte(replacement))
	if operation.Status != upload.IngestSuccess {
		t.Fatalf("replacement = %+v, want success", operation)
	}
	working, err := apitest.Pages(svc).WorkingCopy(t.Context(), created.ID, &owner, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	scriptBlock := apitest.BlockFor(t, working.Blocks, block.PresetScripts)
	if len(scriptBlock.Elements) != 1 {
		t.Fatalf("script block elements = %d, want 1", len(scriptBlock.Elements))
	}
	scripts := scriptBlock.Elements[0].Content.(block.ScriptList).Scripts
	if len(scripts) != 1 || scripts[0].Find != "/updated/g" {
		t.Fatalf("working scripts = %+v", scripts)
	}
	updated, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(t.Context(), version.UpdateRequest{
		OwnerID: owner, WorkID: created.ID, Summary: "Updated the bundled script",
	}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil || !updated.ContentChanged {
		t.Fatalf("publish JSON replacement = %+v, %v", updated, err)
	}
	if got := apitest.ContentGeneration(t, pool, created.ID.String()); got != generation+1 {
		t.Fatalf("content generation = %d, want %d", got, generation+1)
	}
}

func TestAnIdenticalReuploadCannotPublishAnUpdate(t *testing.T) {
	t.Parallel()
	parsed := format.Parsed{Type: "character", Format: "replacing", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: uuid.New(), Text: "Hello"}}}},
		}}
	svc, pool := apitest.WorksWithRegistry(t, apitest.RegistryWith(t, apitest.ReplacingModule{Parsed: &parsed}))
	owner := apitest.Owner(t, svc, "unchanged.upload")
	created := apitest.IngestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	apitest.PublishImported(t, svc, owner, created)
	generation := apitest.ContentGeneration(t, pool, created.ID.String())
	parsed.Elements[1].Content = block.TextSet{Texts: []block.TextItem{{ID: uuid.New(), Text: "Hello"}}}
	candidate := apitest.CurrentCandidate(t, svc, created.ID)
	operation, err := apitest.Uploads(svc).AcceptRevision(t.Context(), upload.RevisionInput{OwnerID: owner, WorkID: created.ID,
		Filename: "wren.json", File: bytes.NewBufferString(`{"payload":true}`)}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := apitest.Uploads(svc).ProcessNextIngest(t.Context()); err != nil || !processed {
		t.Fatalf("ingest = %v, %v", processed, err)
	}
	if _, err := apitest.Uploads(svc).AcceptReplacement(t.Context(), owner, created.ID, operation.ID, candidate, nil, false); err != nil {
		t.Fatal(err)
	}
	_, _, err = version.NewService(svc.Pool(), svc).PublishUpdate(t.Context(), version.UpdateRequest{OwnerID: owner, WorkID: created.ID, Summary: "No change"}, apitest.CurrentCandidate(t, svc, created.ID))
	if !errors.Is(err, version.ErrNothingToPublish) {
		t.Fatalf("identical upload = %v", err)
	}
	if recordedUpdates(t, pool, created.ID) != 1 || apitest.ContentGeneration(t, pool, created.ID.String()) != generation {
		t.Fatal("identical upload published a version")
	}
	apitest.SaveDescription(t, svc, owner, created.ID, "A real change")
	updated, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(t.Context(), version.UpdateRequest{OwnerID: owner, WorkID: created.ID, Summary: "Changed description"}, apitest.CurrentCandidate(t, svc, created.ID))
	if err != nil || !updated.ContentChanged {
		t.Fatalf("real change = %+v, %v", updated, err)
	}
}

func publishedWork(t *testing.T, svc *work.Service, pool *pgxpool.Pool, handle string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	owner := apitest.Owner(t, svc, handle)
	id, err := apitest.Uploads(svc).StartFromNothing(context.Background(), owner, "character", "")
	if err != nil {
		t.Fatalf("start a draft: %v", err)
	}
	apitest.SaveDescription(t, svc, owner, id, "Published description")
	apitest.SaveGreeting(t, svc, owner, id, "Published greeting")
	nsfw := false
	if err := apitest.Pages(svc).SetIdentity(context.Background(), page.Identity{
		OwnerID: owner, WorkID: id, Name: "Published name", IsNSFW: &nsfw,
	}, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("save the header: %v", err)
	}
	if _, err := apitest.Pages(svc).Publish(context.Background(), owner, id, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the asset: %v", err)
	}
	return owner, id
}

func recordedUpdates(t *testing.T, pool *pgxpool.Pool, workID uuid.UUID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from work_snapshots where work_id = $1`, workID).Scan(&count); err != nil {
		t.Fatalf("count the recorded updates: %v", err)
	}
	return count
}

func TestPublishingAnUpdateRecordsTheReviewedCandidateAndMovesTheGeneration(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "update.owner")
	var madeAt, kept any
	if err := pool.QueryRow(ctx, `select created_at, id from works where id = $1`, id).Scan(&madeAt, &kept); err != nil {
		t.Fatal(err)
	}
	generation := apitest.ContentGeneration(t, pool, id.String())
	apitest.SaveDescription(t, svc, owner, id, "Second description")

	recorded, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Rewrote the description",
		Notes: "The longer explanation.", VersionLabel: "v2",
	}, apitest.CurrentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	if recorded.Number != 2 || recorded.Summary != "Rewrote the description" ||
		recorded.Notes != "The longer explanation." || recorded.VersionLabel != "v2" ||
		!recorded.ContentChanged || recorded.RecordedAt.IsZero() {
		t.Fatalf("recorded update = %+v", recorded)
	}
	if got := apitest.ContentGeneration(t, pool, id.String()); got != generation+1 {
		t.Fatalf("content generation = %d, want %d", got, generation+1)
	}
	page, err := apitest.Pages(svc).Detail(ctx, id, nil, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	core := apitest.BlockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Second description" {
		t.Fatal("readers still have the earlier description")
	}
	exported, err := apitest.Downloads(svc).OpenExport(ctx, id, nil, "test_opaque", nil)
	if err != nil || !strings.Contains(string(exported.Body), "Second description") {
		t.Fatalf("published download = %q, error = %v", exported.Body, err)
	}
	var stable bool
	err = pool.QueryRow(ctx, `select created_at = $2 and id = $3 and content_generation = $4
		from works where id = $1`, id, madeAt, kept, recorded.ContentGeneration).Scan(&stable)
	if err != nil || !stable {
		t.Fatalf("asset identity and generation stable = %v, error = %v", stable, err)
	}
}

func TestAnUpdateWithoutAChangeOrASummaryIsRefused(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "unchanged.owner")

	_, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "  ",
	}, apitest.CurrentCandidate(t, svc, id))
	if !errors.Is(err, version.ErrSummaryRequired) {
		t.Fatalf("publishing without a summary = %v, want ErrSummaryRequired", err)
	}
	_, _, err = version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Nothing has changed",
		Notes: "But the notes are new.",
	}, apitest.CurrentCandidate(t, svc, id))
	if !errors.Is(err, version.ErrNothingToPublish) {
		t.Fatalf("publishing an unchanged candidate = %v, want ErrNothingToPublish", err)
	}
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	if _, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Rewrote the description",
	}, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	_, _, err = version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Same content, fresh notes",
	}, apitest.CurrentCandidate(t, svc, id))
	if !errors.Is(err, version.ErrNothingToPublish) {
		t.Fatalf("republishing the same candidate = %v, want ErrNothingToPublish", err)
	}
	if got := recordedUpdates(t, pool, id); got != 2 {
		t.Fatalf("recorded updates = %d, want 2", got)
	}
}

func TestAPresentationChangePublishesWithoutMovingTheGeneration(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "presentation.owner")
	generation := apitest.ContentGeneration(t, pool, id.String())
	arrangement := make([]edit.BlockArrangement, 0)
	for _, holder := range apitest.DraftBlocks(t, pool, id) {
		arrangement = append(arrangement, edit.BlockArrangement{
			ID: holder.ID, Hidden: holder.Definition == "character_core", Width: holder.Width,
		})
	}
	if _, err := apitest.Blocks(svc).ArrangeBlocks(ctx, owner, id, arrangement, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("rearrange the page: %v", err)
	}

	recorded, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Hid the character block",
	}, apitest.CurrentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the update: %v", err)
	}
	if recorded.Number != 2 || recorded.ContentChanged {
		t.Fatalf("recorded update = %+v, want a second update that left the file alone", recorded)
	}
	if got := apitest.ContentGeneration(t, pool, id.String()); got != generation {
		t.Fatalf("content generation = %d, want %d", got, generation)
	}
}

func TestAnUploadWaitingForADecisionRefusesPublicationAndKeepsThePublicWork(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "waiting.owner")
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	stored, err := svc.Store().Put(ctx, strings.NewReader("synthetic replacement upload"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(ctx, `
		insert into ingest_operations (id, owner_id, blob_id, filename, status,
		    target_work_id, replacement_preview)
		values ($1, $2, $3, 'replacement.json', 'preview', $4, '{}'::jsonb)
	`, uuid.New(), owner, stored.ID, id)
	if err != nil {
		t.Fatal(err)
	}

	_, items, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Rewrote the description",
	}, apitest.CurrentCandidate(t, svc, id))
	if !errors.Is(err, work.ErrPublishFloor) {
		t.Fatalf("publishing over a waiting upload = %v, want ErrPublishFloor", err)
	}
	for _, item := range items {
		if item.ID == "upload" && item.Met {
			t.Fatal("the waiting upload was reported as reviewed")
		}
	}
	if got := recordedUpdates(t, pool, id); got != 1 {
		t.Fatalf("recorded updates = %d, want the published version alone", got)
	}
	page, err := apitest.Pages(svc).Detail(ctx, id, nil, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	core := apitest.BlockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Published description" {
		t.Fatal("the refused publication reached readers")
	}
}

func TestARefusedAnnouncementRollsTheWholePublicationBack(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	refused := errors.New("delivery refused this update")
	versions := apitest.Versions(svc)
	versions.OnUpdatePublished(func(context.Context, pgx.Tx, version.Update, version.UpdateAnnouncement) error { return refused })
	owner, id := apitest.PublishedWork(t, svc, "rollback.owner")
	generation := apitest.ContentGeneration(t, pool, id.String())
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	working := apitest.CurrentCandidate(t, svc, id).Version

	_, _, err := versions.PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Rewrote the description",
	}, &work.Candidate{Version: working})
	if !errors.Is(err, refused) {
		t.Fatalf("publishing with a refused announcement = %v", err)
	}
	if got := recordedUpdates(t, pool, id); got != 1 {
		t.Fatalf("recorded updates = %d, want the published version alone", got)
	}
	if got := apitest.ContentGeneration(t, pool, id.String()); got != generation {
		t.Fatalf("content generation = %d, want %d", got, generation)
	}
	if got := apitest.CurrentCandidate(t, svc, id).Version; got != working {
		t.Fatalf("working-copy version = %d, want %d", got, working)
	}
}

func TestSimultaneousPublicationRecordsOneUpdate(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "simultaneous.owner")
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	working := apitest.CurrentCandidate(t, svc, id).Version

	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
				OwnerID: owner, WorkID: id, Summary: "Rewrote the description",
			}, &work.Candidate{Version: working})
			results <- err
		}()
	}
	published := 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			published++
		case errors.Is(err, version.ErrNothingToPublish):
		default:
			var conflict *work.VersionConflict
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
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	owner, id := apitest.PublishedWork(t, svc, "restore.owner")
	generation := apitest.ContentGeneration(t, pool, id.String())
	apitest.SaveDescription(t, svc, owner, id, "Second description")
	if _, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Rewrote the description", VersionLabel: "the same label",
	}, apitest.CurrentCandidate(t, svc, id)); err != nil {
		t.Fatalf("publish the second version: %v", err)
	}
	apitest.SaveDescription(t, svc, owner, id, "Published description")

	recorded, _, err := version.NewService(svc.Pool(), svc).PublishUpdate(ctx, version.UpdateRequest{
		OwnerID: owner, WorkID: id, Summary: "Put the first description back",
		VersionLabel: "the same label",
	}, apitest.CurrentCandidate(t, svc, id))
	if err != nil {
		t.Fatalf("publish the restored version: %v", err)
	}
	if recorded.Number != 3 || !recorded.ContentChanged || recorded.VersionLabel != "the same label" {
		t.Fatalf("restored update = %+v, want a third update that moved the file", recorded)
	}
	if got := apitest.ContentGeneration(t, pool, id.String()); got != generation+2 {
		t.Fatalf("content generation = %d, want %d", got, generation+2)
	}
	page, err := apitest.Pages(svc).Detail(ctx, id, nil, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	core := apitest.BlockFor(t, page.Blocks, "character_core")
	if core.Elements[0].Content.(block.Prose).Text != "Published description" {
		t.Fatal("the restored description did not reach readers")
	}
	if got := recordedUpdates(t, pool, id); got != 3 {
		t.Fatalf("recorded updates = %d, want 3", got)
	}
}
