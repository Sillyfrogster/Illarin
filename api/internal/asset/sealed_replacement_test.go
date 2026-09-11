package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
)

type sealingModule struct {
	claimsFirstPayload
	parsed *format.Parsed
}

func (sealingModule) ID() string { return "sealing" }

func (sealingModule) Declaration() format.Declaration {
	declaration := testReaderDeclaration("sealing", "preset")
	declaration.Label = "Sealing format"
	declaration.Direction.Write = true
	declaration.Header = []format.HeaderField{format.HeaderName}
	declaration.TestedOrigins = append(declaration.TestedOrigins, format.OriginIllarin)
	declaration.Roles = map[block.Role]format.DirectionalRoleSupport{
		block.RolePromptFragments: {
			Read:  format.RoleSupport{Grade: format.SupportFull},
			Write: format.RoleSupport{Grade: format.SupportFull},
		},
	}
	declaration.Preservation = format.PreservationDeclaration{Body: "sealing", Container: []string{sealingNamespace}}
	return declaration
}

func (module sealingModule) Parse(context.Context, probe.Inspection, format.Claim) (format.Parsed, error) {
	return *module.parsed, nil
}

func (sealingModule) Write(context.Context, format.ExportAsset) (format.Artifact, error) {
	return format.Artifact{MediaType: "text/plain", Extension: ".txt"}, nil
}

const sealingNamespace = "sealing_block"

func sealingRemainder(fragmentID uuid.UUID, sourceID string) format.Remainder {
	payload, err := json.Marshal(map[string]string{"id": sourceID})
	if err != nil {
		panic(err)
	}
	return format.Remainder{
		Owner: format.OwnerItem, OwnerID: fragmentID,
		Namespace: sealingNamespace, Payload: payload,
	}
}

func promptListParsed(fragment block.PromptFragment, sourceID string) format.Parsed {
	return format.Parsed{
		Kind: "preset", Format: "sealing", Header: format.Header{Name: "Sample preset"},
		Elements: []block.Element{{
			Type: block.TypePromptList, Role: block.RolePromptFragments,
			Content: block.PromptList{Fragments: []block.PromptFragment{fragment}},
		}},
		Remainder: []format.Remainder{sealingRemainder(fragment.ID, sourceID)},
	}
}

func TestASealedPlaceholderTakesTheWordingTheAssetAlreadyHolds(t *testing.T) {
	held := block.NewItemID()
	parsed := promptListParsed(block.PromptFragment{
		ID: held, Name: "Setup", Text: "The wording only this asset holds", Enabled: true,
	}, "setup")
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, sealingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "sealing.owner")
	created := ingestOne(t, svc, owner, "loom.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)

	arriving := block.NewItemID()
	replacement := promptListParsed(block.PromptFragment{
		ID: arriving, Name: "Setup", Protected: true, Enabled: true,
	}, "setup")
	replacement.Protected = format.ProtectedImport{
		Prompts: []format.ProtectedPrompt{{
			FragmentID: arriving, SourceKey: "setup", ReuseExisting: true,
		}},
		Apps: []string{protected.AppLumiverse},
	}
	parsed = replacement

	operation := stageReplacementFile(t, svc, owner, created.ID)
	if operation.Status != IngestPreview {
		t.Fatalf("staged = %+v", operation)
	}
	if operation.Preview.Seals != 1 {
		t.Fatalf("seals = %d, want 1", operation.Preview.Seals)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID,
		currentCandidate(t, svc, created.ID), nil); err != nil {
		t.Fatalf("AcceptReplacement: %v", err)
	}

	working, err := svc.WorkingCopy(context.Background(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	fragment := onlyFragment(t, working.Blocks)
	if !fragment.Protected {
		t.Fatalf("fragment = %+v, want a sealed fragment", fragment)
	}
	if fragment.Text != "The wording only this asset holds" {
		t.Fatalf("sealed wording = %q, want the wording the asset already held", fragment.Text)
	}
}

func TestASealedPlaceholderWithNoWordingAnywhereCanBeReviewedByName(t *testing.T) {
	parsed := promptListParsed(block.PromptFragment{
		ID: block.NewItemID(), Name: "Setup", Text: "Present", Enabled: true,
	}, "setup")
	svc, _ := newTestServiceWithRegistry(t, registryWithModule(t, sealingModule{parsed: &parsed}))
	owner := revisionOwner(t, svc, "unfillable.owner")
	created := ingestOne(t, svc, owner, "loom.json", []byte(`{"payload":true}`))
	publishImported(t, svc, owner, created)

	arriving := block.NewItemID()
	replacement := promptListParsed(block.PromptFragment{
		ID: arriving, Name: "Late addition", Protected: true, Enabled: true,
	}, "late")
	replacement.Protected = format.ProtectedImport{
		Prompts: []format.ProtectedPrompt{{
			FragmentID: arriving, SourceKey: "late", ReuseExisting: true,
		}},
		Apps: []string{protected.AppLumiverse},
	}
	parsed = replacement

	operation := stageReplacementFile(t, svc, owner, created.ID)
	if operation.Status != IngestPreview || operation.Preview == nil {
		t.Fatalf("staged = %+v, want a replacement preview", operation)
	}
	if !slices.Equal(operation.Preview.MissingWording, []string{"Late addition"}) {
		t.Fatalf("missing wording = %+v", operation.Preview.MissingWording)
	}
	if _, err := svc.AcceptReplacement(context.Background(), owner, created.ID, operation.ID,
		currentCandidate(t, svc, created.ID), nil); err != nil {
		t.Fatalf("AcceptReplacement: %v", err)
	}

	working, err := svc.WorkingCopy(context.Background(), created.ID, &owner, ContentShown)
	if err != nil {
		t.Fatal(err)
	}
	fragment := onlyFragment(t, working.Blocks)
	if !fragment.Protected || fragment.Text != "" {
		t.Fatalf("fragment = %+v, want a sealed prompt awaiting its wording", fragment)
	}
}

func stageReplacementFile(t *testing.T, svc *Service, owner, assetID uuid.UUID) IngestOperation {
	t.Helper()
	operation, err := svc.AcceptRevision(context.Background(), RevisionInput{
		OwnerID: owner, AssetID: assetID, Filename: "loom.json",
		File: bytes.NewBufferString(`{"payload":true,"replacement":true}`),
	}, currentCandidate(t, svc, assetID))
	if err != nil {
		t.Fatalf("AcceptRevision: %v", err)
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
