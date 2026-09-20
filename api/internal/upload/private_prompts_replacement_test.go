package upload

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type privatePromptModule struct {
	matchesFirstPayload
	parsed *format.Parsed
}

func (privatePromptModule) ID() string { return "preset_lumiverse" }

func (privatePromptModule) Declaration() format.Declaration {
	declaration := testReaderDeclaration("preset_lumiverse", "preset")
	declaration.KeepsPrivatePrompts = true
	declaration.Label = "Private prompt format"
	declaration.Direction.Write = true
	declaration.Header = []format.HeaderField{format.HeaderName}
	declaration.TestedOrigins = append(declaration.TestedOrigins, format.OriginIllarin)
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RolePromptFragments: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
	}
	declaration.Preservation = format.PreservationDeclaration{Body: "private_prompts", Container: []string{privatePromptNamespace}}
	return declaration
}

func (module privatePromptModule) Parse(context.Context, format.Inspection, format.Match) (format.Parsed, error) {
	return *module.parsed, nil
}

func (privatePromptModule) Write(context.Context, format.ExportWork) (format.MainFile, error) {
	return format.MainFile{MediaType: "text/plain", Extension: ".txt"}, nil
}

const privatePromptNamespace = "private_prompt_block"

func privatePromptRemainder(fragmentID uuid.UUID, sourceID string) format.Remainder {
	payload, err := json.Marshal(map[string]string{"id": sourceID})
	if err != nil {
		panic(err)
	}
	return format.Remainder{
		Owner: format.OwnerItem, OwnerID: fragmentID,
		Namespace: privatePromptNamespace, Payload: payload,
	}
}

func promptListParsed(fragment block.PromptFragment, sourceID string) format.Parsed {
	return format.Parsed{
		Type: "preset", Format: "preset_lumiverse", Header: format.Header{Name: "Sample preset"},
		Elements: []block.Element{{
			Type: block.TypePromptList, Role: block.RolePromptFragments,
			Content: block.PromptList{Fragments: []block.PromptFragment{fragment}},
		}},
		Remainder: []format.Remainder{privatePromptRemainder(fragment.ID, sourceID)},
	}
}

func TestAPrivatePlaceholderTakesTheWordingTheWorkAlreadyHolds(t *testing.T) {
	t.Parallel()
	held := block.NewItemID()
	parsed := promptListParsed(block.PromptFragment{
		ID: held, Name: "Setup", Text: "The wording only this asset holds", Enabled: true,
	}, "setup")
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, privatePromptModule{parsed: &parsed}))
	owner := originalFileOwner(t, svc, "private.owner")
	created := ingestOne(t, svc, owner, "loom.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)

	arriving := block.NewItemID()
	replacement := promptListParsed(block.PromptFragment{
		ID: arriving, Name: "Setup", Private: true, Enabled: true,
	}, "setup")
	replacement.PrivatePrompts = []format.PrivatePrompt{{
		FragmentID: arriving, SourceKey: "setup", ReuseExisting: true,
	}}
	parsed = replacement

	operation := stageReplacementFile(t, svc, owner, created.ID)
	if operation.Status != IngestPreview {
		t.Fatalf("staged = %+v", operation)
	}
	if operation.Preview.PrivatePrompts != 1 {
		t.Fatalf("private prompts = %d, want 1", operation.Preview.PrivatePrompts)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID,
		currentCandidate(t, svc, created.ID), nil, false); err != nil {
		t.Fatalf("AcceptReplacement: %v", err)
	}

	working, err := works(svc).DraftedChanges(context.Background(), created.ID, &owner, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	fragment := onlyFragment(t, working.Blocks)
	if !fragment.Private {
		t.Fatalf("fragment = %+v, want a private fragment", fragment)
	}
	if fragment.Text != "The wording only this asset holds" {
		t.Fatalf("private wording = %q, want the wording the work already held", fragment.Text)
	}
}

func TestAPrivatePlaceholderWithNoWordingAnywhereCanBeReviewedByName(t *testing.T) {
	t.Parallel()
	parsed := promptListParsed(block.PromptFragment{
		ID: block.NewItemID(), Name: "Setup", Text: "Present", Enabled: true,
	}, "setup")
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, privatePromptModule{parsed: &parsed}))
	owner := originalFileOwner(t, svc, "unfillable.owner")
	created := ingestOne(t, svc, owner, "loom.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)

	arriving := block.NewItemID()
	replacement := promptListParsed(block.PromptFragment{
		ID: arriving, Name: "Late addition", Private: true, Enabled: true,
	}, "late")
	replacement.PrivatePrompts = []format.PrivatePrompt{{
		FragmentID: arriving, SourceKey: "late", ReuseExisting: true,
	}}
	parsed = replacement

	operation := stageReplacementFile(t, svc, owner, created.ID)
	if operation.Status != IngestPreview || operation.Preview == nil {
		t.Fatalf("staged = %+v, want a replacement preview", operation)
	}
	if !slices.Equal(operation.Preview.MissingWording, []string{"Late addition"}) {
		t.Fatalf("missing wording = %+v", operation.Preview.MissingWording)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID,
		currentCandidate(t, svc, created.ID), nil, false); err != nil {
		t.Fatalf("AcceptReplacement: %v", err)
	}

	working, err := works(svc).DraftedChanges(context.Background(), created.ID, &owner, work.NSFWShown)
	if err != nil {
		t.Fatal(err)
	}
	fragment := onlyFragment(t, working.Blocks)
	if !fragment.Private || fragment.Text != "" {
		t.Fatalf("fragment = %+v, want a private prompt awaiting its wording", fragment)
	}
}

func stageReplacementFile(t *testing.T, svc *Service, owner, workID uuid.UUID) Operation {
	t.Helper()
	operation, err := svc.AcceptOriginalFile(context.Background(), OriginalFileInput{
		OwnerID: owner, WorkID: workID, Filename: "loom.json",
		File: bytes.NewBufferString(`{"payload":true,"replacement":true}`),
	}, currentCandidate(t, svc, workID))
	if err != nil {
		t.Fatalf("AcceptOriginalFile: %v", err)
	}
	if processed, err := svc.ProcessNextIngest(context.Background()); err != nil || !processed {
		t.Fatalf("ProcessNextIngest = %v, %v", processed, err)
	}
	staged, err := svc.GetIngest(context.Background(), owner, operation.ID)
	if err != nil {
		t.Fatal(err)
	}
	return staged
}

func onlyFragment(t *testing.T, blocks []block.Block) block.PromptFragment {
	t.Helper()
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if ok && len(list.Fragments) == 1 {
				return list.Fragments[0]
			}
		}
	}
	t.Fatalf("no prompt fragment in %+v", blocks)
	return block.PromptFragment{}
}
