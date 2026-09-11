package asset

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/google/uuid"
)

func TestPublishedAssetKeepsPrivateEditsOutOfPublicReads(t *testing.T) {
	registry := format.NewRegistry()
	for _, module := range character.Modules() {
		if err := registry.Register(module); err != nil {
			t.Fatal(err)
		}
	}
	svc, pool := newTestServiceWithRegistry(t, registry)
	ctx := context.Background()
	owner, id := startedDraft(t, svc)
	saveDescription(t, svc, owner, id, pool, "Published description")
	saveGreeting(t, svc, owner, id, pool, "Published greeting")
	adult := false
	if err := svc.SetIdentity(ctx, Identity{OwnerID: owner, AssetID: id, Name: "Published name", Blurb: "Published pitch", IsNSFW: &adult}, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Publish(ctx, owner, id, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	generation := contentGeneration(t, pool, id)
	saveDescription(t, svc, owner, id, pool, "Private description")
	if err := svc.SetIdentity(ctx, Identity{OwnerID: owner, AssetID: id, Name: "Private name", Blurb: "Private pitch", IsNSFW: &adult}, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	page, err := svc.Detail(ctx, id, nil, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	core := blockFor(t, page.Blocks, "character_core")
	if page.Name != "Published name" || page.Blurb != "Published pitch" || core.Elements[0].Content.(block.Prose).Text != "Published description" {
		t.Fatal("private edits reached the published page")
	}
	if got := contentGeneration(t, pool, id); got != generation {
		t.Fatalf("private save advanced generation from %d to %d", generation, got)
	}
	working, err := svc.WorkingCopy(ctx, id, &owner, ContentShown)
	if err != nil || working.Name != "Private name" || working.Blurb != "Private pitch" {
		t.Fatalf("owner working copy identity = %q / %q, error = %v", working.Name, working.Blurb, err)
	}
	other := uuid.New()
	for _, viewer := range []*uuid.UUID{nil, &other} {
		if _, err := svc.WorkingCopy(ctx, id, viewer, ContentShown); !errors.Is(err, ErrNotFound) {
			t.Fatalf("non-owner working copy read = %v", err)
		}
	}
	page, err = svc.Detail(ctx, id, &owner, ContentShown)
	if err != nil || page.Name != "Published name" {
		t.Fatalf("owner public page = %q, error = %v", page.Name, err)
	}
	listed, err := svc.Browse(ctx, ListFilter{Query: "Private name"}, ContentShown)
	if err != nil || len(listed.Items) != 0 {
		t.Fatalf("private name in browse = %+v, error = %v", listed, err)
	}
	listed, err = svc.Browse(ctx, ListFilter{Query: "Private pitch"}, ContentShown)
	if err != nil || len(listed.Items) != 0 {
		t.Fatalf("private blurb in browse = %+v, error = %v", listed, err)
	}
	listed, err = svc.Browse(ctx, ListFilter{Query: "Published pitch"}, ContentShown)
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("published blurb in browse = %+v, error = %v", listed, err)
	}
	exported, err := svc.OpenExport(ctx, id, nil, "chara_card_v2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported.Body), "Private") || !strings.Contains(string(exported.Body), "Published description") {
		t.Fatal("export did not retain the published content")
	}
	restarted := NewService(pool, registry, svc.store)
	working, err = restarted.WorkingCopy(ctx, id, &owner, ContentShown)
	if err != nil || working.Name != "Private name" {
		t.Fatalf("working copy after service restart = %q, error = %v", working.Name, err)
	}
	checkPublished := func() {
		t.Helper()
		current, err := svc.Detail(ctx, id, nil, ContentShown)
		if err != nil || !reflect.DeepEqual(current.Blocks, page.Blocks) || !reflect.DeepEqual(current.Downloads, page.Downloads) {
			t.Fatalf("private block operation changed the published page: %v", err)
		}
	}
	added, err := svc.AddBlock(ctx, owner, id, block.AuthorNotes, block.TypeProse, currentCandidate(t, svc, id))
	if err != nil {
		t.Fatal(err)
	}
	checkPublished()
	update := updateOf(added.Block)
	update.Elements[0].Content = block.Prose{Text: "Private author notes"}
	if _, err := svc.SaveBlock(ctx, owner, id, added.Block.ID, update, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	checkPublished()
	blocks := draftBlocks(t, pool, id)
	arrangement := make([]BlockArrangement, len(blocks))
	for i, holder := range blocks {
		arrangement[len(blocks)-1-i] = BlockArrangement{ID: holder.ID, Width: block.Full, Hidden: holder.ID == added.Block.ID}
	}
	if _, err := svc.ArrangeBlocks(ctx, owner, id, arrangement, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	checkPublished()
	messages := blockFor(t, draftBlocks(t, pool, id), "messages")
	update = updateOf(messages)
	update.Layout = block.Stack3
	if _, err := svc.SaveBlock(ctx, owner, id, messages.ID, update, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.MoveBlockContent(ctx, owner, id, added.Block.ID, messages.ID, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	checkPublished()
	added, err = svc.AddBlock(ctx, owner, id, block.CustomBlock, block.TypeProse, currentCandidate(t, svc, id))
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.RemoveBlock(ctx, owner, id, added.Block.ID, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	checkPublished()
	if err := svc.SetDiscovery(ctx, owner, id, DiscoveryUnlisted); err != nil {
		t.Fatal(err)
	}
	page, err = restarted.Detail(ctx, id, nil, ContentShown)
	if err != nil || page.Discovery != DiscoveryUnlisted || page.Name != "Published name" {
		t.Fatalf("immediate unlisting = %s, name = %q, error = %v", page.Discovery, page.Name, err)
	}
}

func TestPublishedSnapshotIsolatesEveryStoredHeaderAndPreservedValue(t *testing.T) {
	svc, pool := newTestService(t)
	ctx := context.Background()
	owner, id := startedDraft(t, svc)
	saveDescription(t, svc, owner, id, pool, "Recorded description")
	saveGreeting(t, svc, owner, id, pool, "Recorded greeting")
	snapshotExec(t, pool, `update assets set name = 'Recorded', blurb = 'Recorded blurb',
		tags = array['recorded'], is_nsfw = false, asset_version = '1',
		credited_author = 'Recorded credit', nickname = 'Recorded nickname', origin_format = 'test_opaque'
		where id = $1`, id)
	const preserved = `{ "extension": 1, "extension": 2 }`
	snapshotExec(t, pool, `insert into asset_preserved_data (id, asset_id, owner_kind, owner_id, namespace, payload)
		values (gen_random_uuid(), $1, 'asset', $1, 'synthetic', $2::json)`, id, preserved)
	if _, err := svc.Publish(ctx, owner, id, currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	snapshotExec(t, pool, `update assets set name = 'Private', blurb = 'Private blurb',
		tags = array['private'], is_nsfw = true, asset_version = '2',
		credited_author = 'Private credit', nickname = 'Private nickname', origin_format = 'private'
		where id = $1`, id)
	if err := svc.DeletePreservedNamespace(ctx, owner, id, "synthetic", currentCandidate(t, svc, id)); err != nil {
		t.Fatal(err)
	}
	var intact bool
	err := pool.QueryRow(ctx, `select name = 'Recorded' and blurb = 'Recorded blurb'
		and tags = array['recorded'] and not is_nsfw and asset_version = '1'
		and credited_author = 'Recorded credit' and nickname = 'Recorded nickname'
		and origin_format = 'test_opaque' from asset_public.assets where id = $1`, id).Scan(&intact)
	if err != nil || !intact {
		t.Fatalf("published header intact = %v, error = %v", intact, err)
	}
	var kept string
	if err := pool.QueryRow(ctx, `select payload::text from asset_public.asset_preserved_data where asset_id = $1`, id).Scan(&kept); err != nil || kept != preserved {
		t.Fatalf("published preservation = %q, error = %v", kept, err)
	}
}
