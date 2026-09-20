package work_test

import (
	"context"
	"crypto/sha256"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func versionExec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestVersionMediaBelongsToItsWorkAndKeepsItsBytes(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	id, other, mediaID := uuid.New(), uuid.New(), uuid.New()
	stored, err := svc.Store().Put(ctx, strings.NewReader("synthetic historical image"))
	if err != nil {
		t.Fatal(err)
	}
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values
		($1, 'character', 'History', 'published'), ($2, 'character', 'Other', 'draft')`, id, other)
	versionExec(t, pool, `insert into work_media (id, work_id, role, width, height, blob_id)
		values ($1, $2, 'avatar', 12, 34, $3)`, mediaID, id, stored.ID)
	versionExec(t, pool, `update works set cover_media_id = $2 where id = $1`, id, mediaID)
	versionExec(t, pool, `select record_initial_work_version($1, true)`, id)
	versionExec(t, pool, `update works set cover_media_id = null where id = $1`, id)
	versionExec(t, pool, `update work_media set is_current = false where id = $1`, mediaID)
	for _, sql := range []string{
		`update work_media set width = 99 where id = $1`,
		`update work_media set blob_id = null where id = $1`,
		`delete from work_media where id = $1`,
	} {
		if _, err := pool.Exec(ctx, sql, mediaID); err == nil {
			t.Fatalf("historical media mutation succeeded: %s", sql)
		}
	}
	if _, err := pool.Exec(ctx, `update works set published_version_id =
		(select published_version_id from works where id = $1) where id = $2`, id, other); err == nil {
		t.Fatal("another work selected the version")
	}
	if _, err := pool.Exec(ctx, `update work_version_media set work_id = $2 where media_id = $1`, mediaID, other); err == nil {
		t.Fatal("historical media reference changed ownership")
	}
	now := time.Now().Add(48 * time.Hour)
	cleanup := apitest.CleanupAt(svc, func() time.Time { return now })
	if _, err := cleanup.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	now = now.Add(48 * time.Hour)
	if _, err := cleanup.Cleanup(ctx); err != nil {
		t.Fatal(err)
	}
	var retained bool
	if err := pool.QueryRow(ctx, `select exists(select 1 from work_version_media r
		join work_media m on m.id = r.media_id join blobs b on b.id = m.blob_id
		where r.work_id = $1 and m.id = $2 and m.width = 12 and m.height = 34)`, id, mediaID).Scan(&retained); err != nil || !retained {
		t.Fatalf("historical media retained = %v, error = %v", retained, err)
	}
}

func TestFirstPublicationRecordsTheDraftInTheSameTransaction(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	owner, draft := apitest.StartedDraft(t, svc)
	apitest.SaveDescription(t, svc, owner, draft, "Original description")
	apitest.SaveGreeting(t, svc, owner, draft, "Original greeting")
	nsfw := false
	if err := apitest.Pages(svc).SetDetails(context.Background(), page.Details{
		OwnerID: owner, WorkID: draft, Name: "First publication", IsNSFW: &nsfw,
	}, apitest.CurrentCandidate(t, svc, draft)); err != nil {
		t.Fatal(err)
	}
	if _, err := apitest.Pages(svc).Publish(context.Background(), owner, draft, apitest.CurrentCandidate(t, svc, draft)); err != nil {
		t.Fatal(err)
	}
	var captured bool
	err := pool.QueryRow(context.Background(), `select exists(select 1 from works a
		join work_versions s on s.id = a.published_version_id and s.work_id = a.id
		where a.id = $1 and a.lifecycle = 'published' and s.number = 1 and not s.initial_recorded
		and s.payload->>'name' = 'First publication'
		and s.recorded_at = a.updated_at)`, draft).Scan(&captured)
	if err != nil || !captured {
		t.Fatalf("first publication captured = %v, error = %v", captured, err)
	}
}

func TestAVersionRejectsForeignMediaAndPublicationRollsBack(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	owner, draft := apitest.StartedDraft(t, svc)
	apitest.SaveDescription(t, svc, owner, draft, "Description")
	apitest.SaveGreeting(t, svc, owner, draft, "Greeting")
	nsfw := false
	if err := apitest.Pages(svc).SetDetails(context.Background(), page.Details{OwnerID: owner, WorkID: draft, Name: "Candidate", IsNSFW: &nsfw}, apitest.CurrentCandidate(t, svc, draft)); err != nil {
		t.Fatal(err)
	}
	other, media := uuid.New(), uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values ($1, 'character', 'Private', 'draft')`, other)
	versionExec(t, pool, `insert into work_media (id, work_id, role, width, height) values ($1, $2, 'avatar', 1, 1)`, media, other)
	versionExec(t, pool, `update works set cover_media_id = $2 where id = $1`, draft, media)
	if _, err := apitest.Pages(svc).Publish(context.Background(), owner, draft, apitest.CurrentCandidate(t, svc, draft)); err == nil {
		t.Fatal("publication accepted another work's private media")
	}
	var private bool
	err := pool.QueryRow(context.Background(), `select lifecycle = 'draft' and published_version_id is null
		and not exists(select 1 from work_versions where work_id = $1) from works where id = $1`, draft).Scan(&private)
	if err != nil || !private {
		t.Fatalf("failed publication rolled back = %v, error = %v", private, err)
	}
}

func TestVersionHistoryEndsWithExpiredDeletion(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	id := uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values ($1, 'preset', 'Temporary history', 'published')`, id)
	versionExec(t, pool, `select record_initial_work_version($1, true)`, id)
	versionExec(t, pool, `update works set deleted_at = now() - interval '40 days',
		recoverable_until = now() - interval '10 days' where id = $1`, id)
	if _, err := apitest.Cleanup(svc).Cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from work_versions where work_id = $1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("expired versions = %d, error = %v", count, err)
	}
}

func TestVersionMediaStillObeysPurgeAndWorkDeletion(t *testing.T) {
	t.Parallel()
	svc, pool := apitest.Works(t)
	ctx := context.Background()
	id, mediaID := uuid.New(), uuid.New()
	content := "synthetic image to purge"
	stored, err := svc.Store().Put(ctx, strings.NewReader(content))
	if err != nil {
		t.Fatal(err)
	}
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values ($1, 'character', 'Purge history', 'published')`, id)
	versionExec(t, pool, `insert into work_media (id, work_id, role, width, height, blob_id)
		values ($1, $2, 'avatar', 1, 1, $3)`, mediaID, id, stored.ID)
	versionExec(t, pool, `select record_initial_work_version($1, true)`, id)
	if err := apitest.Cleanup(svc).Purge(ctx, sha256.Sum256([]byte(content)), "test_purge", uuid.New()); err != nil {
		t.Fatal(err)
	}
	var removed bool
	err = pool.QueryRow(ctx, `select blob_id is null and not exists(select 1 from blobs where id = $2)
		from work_media where id = $1`, mediaID, stored.ID).Scan(&removed)
	if err != nil || !removed {
		t.Fatalf("historical bytes purged = %v, error = %v", removed, err)
	}
	versionExec(t, pool, `delete from works where id = $1`, id)
	var count int
	if err := pool.QueryRow(ctx, `select count(*) from work_versions where work_id = $1`, id).Scan(&count); err != nil || count != 0 {
		t.Fatalf("orphan versions = %d, error = %v", count, err)
	}
}

func TestConcurrentBaselineCreatesOnlyOneRecordedVersion(t *testing.T) {
	t.Parallel()
	_, pool := apitest.Works(t)
	id := uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values ($1, 'preset', 'Concurrent baseline', 'published')`, id)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			_, err := pool.Exec(context.Background(), `select record_initial_work_version($1, true)`, id)
			results <- err
		}()
	}
	for range 2 {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from work_versions where work_id = $1`, id).Scan(&count); err != nil || count != 1 {
		t.Fatalf("concurrent versions = %d, error = %v", count, err)
	}
}

func TestAVersionRetainsAnImageReferencedOnlyByABlock(t *testing.T) {
	t.Parallel()
	_, pool := apitest.Works(t)
	id, media := uuid.New(), uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle) values ($1, 'character', 'Gallery history', 'published')`, id)
	versionExec(t, pool, `insert into work_media (id, work_id, role, width, height, is_current)
		values ($1, $2, 'gallery', 10, 20, false)`, media, id)
	versionExec(t, pool, `insert into work_blocks (id, work_id, definition, position, layout, width, elements)
		values ($1, $2, 'gallery', 0, 'single', 'full', jsonb_build_array(jsonb_build_object(
		'type', 'image_set', 'content', jsonb_build_object('images', jsonb_build_array(jsonb_build_object('mediaId', $3::text))))))`, uuid.New(), id, media)
	versionExec(t, pool, `select record_initial_work_version($1, true)`, id)
	var retained bool
	err := pool.QueryRow(context.Background(), `select exists(select 1 from work_version_media
		where work_id = $1 and media_id = $2)`, id, media).Scan(&retained)
	if err != nil || !retained {
		t.Fatalf("block image retained = %v, error = %v", retained, err)
	}
}

func TestTheInitialVersionIsRepeatableAndLeavesDraftsPrivate(t *testing.T) {
	t.Parallel()
	_, pool := apitest.Works(t)
	published, draft := uuid.New(), uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle, work_version)
		values ($1, 'character', 'Recorded character', 'published', 'same label'),
		       ($2, 'character', 'Private character', 'draft', '')`, published, draft)
	for range 2 {
		versionExec(t, pool, `select record_initial_work_version(id, true) from works`)
	}
	var count, number int
	var label, name string
	var initial bool
	err := pool.QueryRow(context.Background(), `select count(*) over (), s.number,
		s.version_label, s.payload->>'name', s.initial_recorded
		from work_versions s join works a on a.published_version_id = s.id
		where a.id = $1`, published).Scan(&count, &number, &label, &name, &initial)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 || number != 1 || label != "same label" || name != "Recorded character" || !initial {
		t.Fatalf("unexpected initial version: %d %d %q %q %v", count, number, label, name, initial)
	}
	var private bool
	if err := pool.QueryRow(context.Background(), `select lifecycle = 'draft' and published_version_id is null
		and not exists (select 1 from work_versions where work_id = $1) from works where id = $1`, draft).Scan(&private); err != nil || !private {
		t.Fatalf("draft stayed private = %v, error = %v", private, err)
	}
}

func TestAVersionRetainsExactContentAndCannotBeRewritten(t *testing.T) {
	t.Parallel()
	_, pool := apitest.Works(t)
	id, blockID, promptID := uuid.New(), uuid.New(), uuid.New()
	versionExec(t, pool, `insert into works (id, type, name, lifecycle, blurb, tags, is_nsfw,
		work_version, credited_author, nickname, original_format)
		values ($1, 'preset', 'Recorded preset', 'published', 'Original blurb', '{one,two}', false,
		'v free text', 'Original author', 'Original nickname', 'test-preset')`, id)
	versionExec(t, pool, `insert into work_blocks (id, work_id, definition, title, position, hidden, layout, width, elements)
		values ($1, $2, 'changelog', 'My changelog', 0, true, 'single', 'half',
		'[{"id":"retained-item","type":"prose","content":{"text":"Handwritten history"}}]')`, blockID, id)
	opaque := `{ "z": 1, "a":2, "z": 3 }`
	versionExec(t, pool, `insert into work_preserved_data (id, work_id, owner_type, owner_id, namespace, payload)
		values ($1, $2, 'asset', $2, 'test.opaque', $3::json)`, uuid.New(), id, opaque)
	versionExec(t, pool, `with policy as (insert into private_prompt_apps (work_id, app) values ($1, 'lumiverse'))
		insert into private_prompts (work_id, owner_type, owner_id, payload_type, payload, source_key, digest)
		values ($1, 'prompt_fragment', $2, 'prompt_fragment_text', '{"text":"Private original"}', 'original-key', $3)`, id, promptID, make([]byte, 32))
	versionExec(t, pool, `select record_initial_work_version($1, true)`, id)
	versionExec(t, pool, `update works set name = 'Edited', blurb = 'Edited', tags = '{}' where id = $1`, id)
	versionExec(t, pool, `delete from work_blocks where work_id = $1`, id)
	versionExec(t, pool, `delete from work_preserved_data where work_id = $1`, id)
	versionExec(t, pool, `with policy as (delete from private_prompt_apps where work_id = $1)
		delete from private_prompts where work_id = $1`, id)
	var retained bool
	err := pool.QueryRow(context.Background(), `select
		payload->>'name' = 'Recorded preset' and payload->>'blurb' = 'Original blurb'
		and payload->'tags' = '["one","two"]'::jsonb and payload->>'is_nsfw' = 'false'
		and payload->>'work_version' = 'v free text' and payload->>'credited_author' = 'Original author'
		and payload->>'nickname' = 'Original nickname' and payload->>'original_format' = 'test-preset'
		and payload#>>'{blocks,0,title}' = 'My changelog' and payload#>>'{blocks,0,hidden}' = 'true'
		and payload#>>'{blocks,0,width}' = 'half' and payload#>>'{blocks,0,layout}' = 'single'
		and payload#>>'{blocks,0,elements,0,content,text}' = 'Handwritten history'
		and payload#>>'{preserved_data,0,payload}' = $2
		and private_prompts#>>'{0,payload,text}' = 'Private original'
		and private_prompts#>>'{0,source_key}' = 'original-key'
		from work_versions where work_id = $1`, id, opaque).Scan(&retained)
	if err != nil || !retained {
		t.Fatalf("exact content retained = %v, error = %v", retained, err)
	}
	for _, sql := range []string{
		`update work_versions set payload = '{}' where work_id = $1`,
		`update work_versions set private_prompts = '[]' where work_id = $1`,
		`update work_versions set number = 2 where work_id = $1`,
		`delete from work_versions where work_id = $1`,
	} {
		if _, err := pool.Exec(context.Background(), sql, id); err == nil {
			t.Fatalf("version mutation succeeded: %s", sql)
		}
	}
}
