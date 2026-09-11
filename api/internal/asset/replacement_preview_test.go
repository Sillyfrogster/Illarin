package asset

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/google/uuid"
)

func TestReplacementPreviewLeavesThePublishedAssetAloneUntilAccepted(t *testing.T) {
	parsed := format.Parsed{
		Kind: "character", Format: "replacing",
		Header: format.Header{Name: "Wren"},
		Elements: []block.Element{{
			Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"},
		}, {
			Type: block.TypeTextSet, Role: block.RoleGreetings,
			Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}},
		}},
	}
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, replacingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "preview.owner")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)

	parsed.Elements[0].Content = block.Prose{Text: "After"}
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "wren.json",
		File: bytes.NewBufferString(`{"payload":true,"replacement":true}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	preview, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Status != IngestPreview || preview.Preview == nil {
		t.Fatalf("preview = %+v", preview)
	}
	public, err := svc.Detail(context.Background(), created.ID, nil, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	if text := blockFor(t, public.Blocks, block.CharacterCore).Elements[0].Content.(block.Prose).Text; text != "Before" {
		t.Fatalf("published description = %q, want Before", text)
	}
	accepted, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID, candidate, nil)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != IngestSuccess {
		t.Fatalf("accepted replacement = %+v", accepted)
	}
	working, err := svc.WorkingCopy(context.Background(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	if text := blockFor(t, working.Blocks, block.CharacterCore).Elements[0].Content.(block.Prose).Text; text != "After" {
		t.Fatalf("working description = %q, want After", text)
	}
}

func TestReplacementPreviewRefusesAStaleAcceptance(t *testing.T) {
	parsed := format.Parsed{
		Kind: "character", Format: "replacing", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}}},
	}
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, replacingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "stale.preview.owner")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "wren.json", File: bytes.NewBufferString(`{"payload":true}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	adult := false
	if err := svc.SetIdentity(context.Background(), Identity{
		OwnerID: owner, AssetID: created.ID, Name: "Newer", IsNSFW: &adult,
	}, currentCandidate(t, svc, created.ID)); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID, candidate, nil)
	var conflict *VersionConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("stale acceptance = %v, want version conflict", err)
	}
}

func TestReplacementPreviewRequiresAChoiceForUnrepresentableContent(t *testing.T) {
	parsed := format.Parsed{
		Kind: "character", Format: "replacing", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}},
			{Type: block.TypeProse, Role: block.RolePersonality, Content: block.Prose{Text: "Patient"}},
		},
	}
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, replacingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "unrepresentable.preview.owner")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	parsed.Elements = parsed.Elements[:1]
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "wren.json", File: bytes.NewBufferString(`{"payload":true,"replacement":true}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	preview, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(preview.Preview.Unrepresentable, string(block.RolePersonality)) {
		t.Fatalf("unrepresentable content = %+v", preview.Preview)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID, candidate, nil); !errors.Is(err, ErrReplacementDecision) {
		t.Fatalf("accept without a decision = %v", err)
	}
	decisions := make(map[string]string, len(preview.Preview.Unrepresentable))
	for _, role := range preview.Preview.Unrepresentable {
		decisions[role] = "keep"
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID, currentCandidate(t, svc, created.ID), decisions); err != nil {
		t.Fatal(err)
	}
	working, err := svc.WorkingCopy(context.Background(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRole(blockFor(t, working.Blocks, block.CharacterCore).Elements, block.RolePersonality) {
		t.Fatal("kept personality disappeared from the candidate")
	}
}

func hasRole(elements []block.Element, role block.Role) bool {
	for _, element := range elements {
		if element.Role == role {
			return true
		}
	}
	return false
}

func TestCancellingAReplacementPreviewLeavesTheCandidateAlone(t *testing.T) {
	parsed := format.Parsed{
		Kind: "character", Format: "replacing", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}}},
	}
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, replacingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "cancel.preview.owner")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"payload":true}`))
	parsed.Elements[0].Content = block.Prose{Text: "After"}
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "wren.json", File: bytes.NewBufferString(`{"payload":true,"replacement":true}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	if err := svc.CancelReplacement(context.Background(), owner, created.ID, operation.ID); err != nil {
		t.Fatal(err)
	}
	cancelled, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil || cancelled.Status != IngestCancelled {
		t.Fatalf("cancelled operation = %+v, error = %v", cancelled, err)
	}
	working, err := svc.WorkingCopy(context.Background(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	if text := blockFor(t, working.Blocks, block.CharacterCore).Elements[0].Content.(block.Prose).Text; text != "Before" {
		t.Fatalf("working description after cancellation = %q, want Before", text)
	}
}

func TestReplacementPreviewUsesStableItemIDs(t *testing.T) {
	shared, removed, added := block.NewItemID(), block.NewItemID(), block.NewItemID()
	working := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: shared, Text: "Same"}, {ID: removed, Text: "Old"}}},
	}}}}
	incoming := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: shared, Text: "Updated"}, {ID: added, Text: "New"}}},
	}}}}
	groups := compareContent(working, incoming)
	var additions, removals, updates int
	for _, group := range groups {
		if group.Subject != string(block.RoleGreetings) {
			continue
		}
		for _, change := range group.Changes {
			switch change.Kind {
			case ChangeAdded:
				additions++
			case ChangeRemoved:
				removals++
			case ChangeEdited:
				updates++
			}
		}
	}
	if additions != 1 || removals != 1 || updates != 1 {
		t.Fatalf("stable-item changes = %+v", groups)
	}
}

func TestReplacementPreviewShowsTheWordingOnBothSides(t *testing.T) {
	item := block.NewItemID()
	working := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: item, Name: "Opening", Text: "Old wording"}}},
	}}}}
	incoming := []block.Block{{Elements: []block.Element{{
		Role: block.RoleGreetings, Type: block.TypeTextSet,
		Content: block.TextSet{Texts: []block.TextItem{{ID: item, Name: "Opening", Text: "New wording"}}},
	}}}}
	groups := compareContent(working, incoming)
	if len(groups) != 1 || len(groups[0].Changes) != 1 {
		t.Fatalf("groups = %+v", groups)
	}
	change := groups[0].Changes[0]
	if change.Kind != ChangeEdited || change.Before != "Old wording" || change.After != "New wording" {
		t.Fatalf("change = %+v, want the wording on both sides", change)
	}
}

func TestFilteringSuppliedRolesDoesNotAlterTheStagedBlocks(t *testing.T) {
	blocks := []block.Block{{Elements: []block.Element{
		{Role: block.RolePersonality, Type: block.TypeProse, Content: block.Prose{Text: "Keep"}},
		{Role: block.RoleDescription, Type: block.TypeProse, Content: block.Prose{Text: "Replace"}},
	}}}
	filtered := blocksWithSuppliedRoles(blocks, []block.Role{block.RoleDescription})
	if len(filtered[0].Elements) != 1 || filtered[0].Elements[0].Role != block.RoleDescription {
		t.Fatalf("filtered blocks = %+v", filtered)
	}
	if len(blocks[0].Elements) != 2 || blocks[0].Elements[0].Role != block.RolePersonality {
		t.Fatalf("staged blocks changed = %+v", blocks)
	}
}

func TestReplacementPreviewReportsConflictsWhenEitherSideOmitsContent(t *testing.T) {
	baseline := []block.Block{{Elements: []block.Element{{
		Role: block.RoleDescription, Type: block.TypeProse, Content: block.Prose{Text: "Published"},
	}}}}
	incoming := []block.Block{{Elements: []block.Element{{
		Role: block.RoleDescription, Type: block.TypeProse, Content: block.Prose{Text: "From file"},
	}}}}
	conflicts := replacementConflicts(nil, baseline, incoming, nil, nil, nil, nil, nil, nil)
	if !slices.Contains(conflicts, string(block.RoleDescription)) {
		t.Fatalf("missing-content conflict = %+v", conflicts)
	}
}

func TestReplacementPreviewReportsAConflictingImageReplacement(t *testing.T) {
	public, local, incoming := block.NewItemID(), block.NewItemID(), block.NewItemID()
	changes := comparePictureSets([]uuid.UUID{local}, []uuid.UUID{incoming})
	if len(changes) != 2 {
		t.Fatalf("same-count image replacement = %+v", changes)
	}
	conflicts := replacementConflicts(nil, nil, nil, nil, nil, nil,
		[]uuid.UUID{local}, []uuid.UUID{public}, []uuid.UUID{incoming})
	if !slices.Contains(conflicts, picturesSubject) {
		t.Fatalf("image conflict = %+v", conflicts)
	}
}

func TestReplacementPreviewReportsConflictingOpaqueData(t *testing.T) {
	current := []format.Remainder{{Owner: format.OwnerAsset, OwnerID: uuid.New(), Namespace: "extension", Payload: []byte(`{"local":true}`)}}
	public := []format.Remainder{{Owner: format.OwnerAsset, OwnerID: current[0].OwnerID, Namespace: "extension", Payload: []byte(`{"published":true}`)}}
	incoming := []format.Remainder{{Owner: format.OwnerAsset, OwnerID: current[0].OwnerID, Namespace: "extension", Payload: []byte(`{"file":true}`)}}
	changes := comparePreserved(asVersionPreserved(current), asVersionPreserved(incoming))
	if len(changes) != 1 || changes[0].Kind != ChangeEdited || changes[0].Name != "extension" {
		t.Fatalf("opaque replacement = %+v", changes)
	}
	conflicts := replacementConflicts(nil, nil, nil, current, public, incoming, nil, nil, nil)
	if !slices.Contains(conflicts, preservedSubject) {
		t.Fatalf("opaque conflict = %+v", conflicts)
	}
}

func TestReplacementPreviewAcceptsACharacterFormatChange(t *testing.T) {
	old := namedReplacementModule{id: "old_character", parsed: format.Parsed{
		Kind: "character", Format: "old_character", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "Before"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
		},
	}}
	updated := namedReplacementModule{id: "new_character", parsed: format.Parsed{
		Kind: "character", Format: "new_character", Header: format.Header{Name: "Wren"},
		Elements: []block.Element{
			{Type: block.TypeProse, Role: block.RoleDescription, Content: block.Prose{Text: "After"}},
			{Type: block.TypeTextSet, Role: block.RoleGreetings, Content: block.TextSet{Texts: []block.TextItem{{ID: block.NewItemID(), Text: "Hello"}}}},
		},
	}}
	registry := format.NewRegistry()
	for _, module := range []format.Module{old, updated} {
		if err := registry.Register(module); err != nil {
			t.Fatal(err)
		}
	}
	svc, pool := newTestServiceWithRegistry(t, registry)
	owner := revisionOwner(t, svc, "format.change.owner")
	created := ingestOne(t, svc, owner, "wren.json", []byte(`{"spec":"old_character"}`))
	candidate := currentCandidate(t, svc, created.ID)
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: created.ID, Filename: "wren.json", File: bytes.NewBufferString(`{"spec":"new_character"}`),
	}, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	preview, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil || preview.Preview == nil || preview.Preview.Format != "new_character" {
		t.Fatalf("format-change preview = %+v, error = %v", preview, err)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID, candidate, nil); err != nil {
		t.Fatal(err)
	}
	var origin string
	if err := pool.QueryRow(context.Background(), `select origin_format from assets where id = $1`, created.ID).Scan(&origin); err != nil || origin != "new_character" {
		t.Fatalf("replacement origin = %q, error = %v", origin, err)
	}
}

func TestReplacementPreviewsEveryBuildableKind(t *testing.T) {
	for _, kind := range []string{"character", "lorebook", "preset", "theme", "pack"} {
		t.Run(kind, func(t *testing.T) {
			module := kindModule{id: "preview_" + kind, kind: kind}
			svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, module))
			owner := revisionOwner(t, svc, "preview."+kind)
			created := ingestOne(t, svc, owner, "asset.json", []byte(`{"spec":"preview_`+kind+`"}`))
			operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
				OwnerID: owner, AssetID: created.ID, Filename: "asset.json",
				File: bytes.NewBufferString(`{"spec":"preview_` + kind + `"}`),
			}, currentCandidate(t, svc, created.ID))
			if err != nil {
				t.Fatal(err)
			}
			if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
				t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
			}
			found, err := svc.GetIngest(context.Background(), owner, operation.ID)
			if err != nil || found.Status != IngestPreview || found.Preview == nil {
				t.Fatalf("preview = %+v, error = %v", found, err)
			}
		})
	}
}

type namedReplacementModule struct {
	id     string
	parsed format.Parsed
}

func (m namedReplacementModule) ID() string { return m.id }

func (m namedReplacementModule) Declaration() format.Declaration {
	declaration := testReaderDeclaration(m.id, "character")
	declaration.Direction.Write = true
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RoleDescription: {Read: format.RoleSupport{Grade: format.SupportFull}, Write: format.RoleSupport{Grade: format.SupportFull}},
		block.RoleGreetings:   {Read: format.RoleSupport{Grade: format.SupportFull}, Write: format.RoleSupport{Grade: format.SupportFull}},
	}
	return declaration
}

func (m namedReplacementModule) Claim(file probe.Inspection) (format.Claim, bool) {
	for _, payload := range file.Payloads {
		if spec, _ := payload.String("spec"); spec == m.id {
			return format.AuthoritativeClaim(payload, "spec")
		}
	}
	return format.Claim{}, false
}

func (m namedReplacementModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return m.parsed, nil
}

func (m namedReplacementModule) Write(context.Context, format.ExportAsset) (format.Artifact, error) {
	return format.Artifact{MediaType: "text/plain", Extension: ".txt"}, nil
}
