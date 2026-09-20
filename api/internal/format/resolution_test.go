package format

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type matchingModule struct {
	id            string
	spec          string
	authoritative bool
}

func (m matchingModule) ID() string { return m.id }
func (m matchingModule) Declaration() Declaration {
	declaration := testReaderDeclaration(m.id, "character")
	declaration.Recognition = []Recognition{{
		Type: RecognitionMarker, Path: []string{"spec"}, Values: []string{m.spec},
		Containers: []Container{PNG},
	}}
	return declaration
}
func (m matchingModule) Parse(context.Context, Inspection, Match) (Parsed, error) {
	return Parsed{Format: m.id}, nil
}
func (m matchingModule) Match(file Inspection) (Match, bool) {
	for _, payload := range file.Payloads {
		if spec, ok := payload.String("spec"); ok && spec == m.spec {
			if m.authoritative {
				return AuthoritativeMatch(payload, "spec")
			}
			return CompatibilityMatch(payload), true
		}
	}
	return Match{}, false
}

func TestResolveReturnsNoModuleWhenNothingMatchesTheFile(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()

	_, ok, err := registry.Resolve(Inspection{Container: Unknown})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if ok {
		t.Fatal("an unmatched file resolved to a module")
	}
}

func TestResolveRejectsAPayloadThatNamesAnUnsupportedFormat(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	if err := registry.Register(matchingModule{
		id: "chara_card_v3", spec: "chara_card_v3", authoritative: true,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, ok, err := registry.Resolve(probedPayload("future_card", "card"))
	if ok {
		t.Fatal("unsupported format resolved to a module")
	}
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("Resolve error = %v, want ErrUnsupportedFormat", err)
	}
}

func TestResolvePrefersAnAuthoritativeMatchRegardlessOfRegistrationOrder(t *testing.T) {
	t.Parallel()
	file := probedPayload("chara_card_v3", "ccv3")
	compatible := matchingModule{id: "compatible_reader", spec: "chara_card_v3"}
	authority := matchingModule{id: "chara_card_v3", spec: "chara_card_v3", authoritative: true}

	for _, modules := range [][]Module{
		{compatible, authority},
		{authority, compatible},
	} {
		registry := NewRegistry()
		for _, module := range modules {
			if err := registry.Register(module); err != nil {
				t.Fatalf("Register: %v", err)
			}
		}

		got, ok, err := registry.Resolve(file)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if !ok || got.Module.ID() != "chara_card_v3" {
			t.Fatalf("resolved module = %v, %v; want chara_card_v3, true", got.Module, ok)
		}
	}
}

func TestResolveRejectsTwoAuthoritativeMatchesOnOnePayload(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	for _, id := range []string{"first", "second"} {
		if err := registry.Register(forcedAuthoritativeModule{id: id}); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	_, _, err := registry.Resolve(probedPayload("chara_card_v2", "chara"))
	if !errors.Is(err, ErrConflictingMatches) {
		t.Fatalf("Resolve error = %v, want ErrConflictingMatches", err)
	}
}

func TestResolveUsesThePayloadMarkerRatherThanItsLocation(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	for _, module := range []Module{
		matchingModule{id: "chara_card_v3", spec: "chara_card_v3", authoritative: true},
		matchingModule{id: "chara_card_v2", spec: "chara_card_v2", authoritative: true},
	} {
		if err := registry.Register(module); err != nil {
			t.Fatalf("Register: %v", err)
		}
	}

	got, ok, err := registry.Resolve(probedPayload("chara_card_v2", "ccv3"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !ok || got.Module.ID() != "chara_card_v2" {
		t.Fatalf("resolved module = %v, %v; want chara_card_v2, true", got.Module, ok)
	}
}

type forcedAuthoritativeModule struct{ id string }

func (m forcedAuthoritativeModule) ID() string { return m.id }
func (m forcedAuthoritativeModule) Declaration() Declaration {
	return testReaderDeclaration(m.id, "character")
}
func (m forcedAuthoritativeModule) Match(file Inspection) (Match, bool) {
	return Match{
		payloadID: file.Payloads[0].ID,
		strength:  authoritative,
		formatID:  m.id,
	}, true
}
func (m forcedAuthoritativeModule) Parse(context.Context, Inspection, Match) (Parsed, error) {
	return Parsed{Format: m.id}, nil
}

func TestResolveRejectsAuthorityForADifferentMarker(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	if err := registry.Register(matchingModule{
		id: "chara_card_v3", spec: "chara_card_v2", authoritative: true,
	}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, err := registry.Resolve(probedPayload("chara_card_v2", "chara"))
	if !errors.Is(err, ErrInvalidMatch) {
		t.Fatalf("Resolve error = %v, want ErrInvalidMatch", err)
	}
}

func probedPayload(spec, location string) Inspection {
	return Inspection{
		Container: PNG,
		Payloads: []Payload{{
			ID:       0,
			Location: PayloadLocation{Container: PNG, Name: location},
			Root: map[string]json.RawMessage{
				"spec": json.RawMessage(`"` + spec + `"`),
			},
		}},
	}
}

type containerModule struct{ matchingModule }

func (containerModule) OwnedSpecs() []string { return []string{"chara_card_v3"} }

func TestResolveAcceptsAuthorityForASpecTheModuleOwns(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	if err := registry.Register(containerModule{matchingModule{
		id: "charx", spec: "chara_card_v3", authoritative: true,
	}}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	got, ok, err := registry.Resolve(probedPayload("chara_card_v3", "card.json"))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !ok || got.Module.ID() != "charx" {
		t.Fatalf("resolved module = %v, %v; want charx, true", got.Module, ok)
	}
}

func TestResolveStillRejectsAuthorityForASpecNobodyOwns(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	if err := registry.Register(containerModule{matchingModule{
		id: "charx", spec: "chara_card_v2", authoritative: true,
	}}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, err := registry.Resolve(probedPayload("chara_card_v2", "card.json"))
	if !errors.Is(err, ErrInvalidMatch) {
		t.Fatalf("Resolve error = %v, want ErrInvalidMatch", err)
	}
}
