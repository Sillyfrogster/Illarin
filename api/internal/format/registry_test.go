package format

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type stubModule struct{ id string }

func testReaderDeclaration(id, workType string) Declaration {
	return Declaration{
		ID: id, Type: workType, Direction: Direction{Read: true},
		Recognition: []Recognition{{
			Type: RecognitionShape, Containers: []Container{JSON},
			Required: map[string]ValueType{"payload": ValueBoolean},
		}},
		Limits:        ContentLimits{PayloadBytes: 1024, CollectionItems: 100, ItemBytes: 100},
		ConsumedKeys:  []string{"payload"},
		Preservation:  PreservationDeclaration{Body: "test"},
		TestedOrigins: []string{id},
	}
}

func (s stubModule) ID() string               { return s.id }
func (s stubModule) Declaration() Declaration { return testReaderDeclaration(s.id, "character") }
func (stubModule) Match(Inspection) (Match, bool) {
	return Match{}, false
}

type declaredMarkerModule struct{}

func (declaredMarkerModule) ID() string { return "theme_lumiverse" }
func (declaredMarkerModule) Declaration() Declaration {
	declaration := testReaderDeclaration("theme_lumiverse", "theme")
	declaration.Recognition = []Recognition{{
		Type: RecognitionMarker, Containers: []Container{JSON},
		Path: []string{"format"}, Values: []string{"3"},
	}}
	return declaration
}
func (m declaredMarkerModule) Match(file Inspection) (Match, bool) {
	return MatchByDeclaration(file, m.Declaration())
}
func (declaredMarkerModule) Parse(context.Context, Inspection, Match) (Parsed, error) {
	return Parsed{}, nil
}

func TestMarkerOutsideTheAcceptedSetNamesTheFormatAndValue(t *testing.T) {
	t.Parallel()
	registry := NewRegistry()
	if err := registry.Register(declaredMarkerModule{}); err != nil {
		t.Fatalf("register module: %v", err)
	}
	file := Inspection{Payloads: []Payload{{
		ID: 0, Location: PayloadLocation{Container: JSON},
		Root: map[string]json.RawMessage{"format": json.RawMessage(`4`)},
	}}}
	_, matched, err := registry.Resolve(file)
	if matched || !errors.Is(err, ErrUnsupportedFormat) ||
		!strings.Contains(err.Error(), "theme_lumiverse") || !strings.Contains(err.Error(), `"4"`) {
		t.Fatalf("Resolve = matched %v, error %v; want named unsupported marker", matched, err)
	}
}
func (s stubModule) Parse(context.Context, Inspection, Match) (Parsed, error) {
	return Parsed{Format: s.id}, nil
}

type declarationModule struct {
	stubModule
	declaration Declaration
}

func (m declarationModule) Declaration() Declaration { return m.declaration }

func registerShapes(t *testing.T, shapes map[string]map[string]ValueType) *Registry {
	t.Helper()
	registry := NewRegistry()
	for id, required := range shapes {
		declaration := testReaderDeclaration(id, "character")
		declaration.Recognition[0].Required = required
		if err := registry.Register(declarationModule{
			stubModule: stubModule{id: id}, declaration: declaration,
		}); err != nil {
			t.Fatalf("register %s: %v", id, err)
		}
	}
	return registry
}

func TestValidationRejectsAShapeThatShadowsAnother(t *testing.T) {
	t.Parallel()
	registry := registerShapes(t, map[string]map[string]ValueType{
		"looser":   {"alpha": ValueString},
		"stricter": {"alpha": ValueString, "beta": ValueBoolean},
	})
	if err := registry.ValidateDeclarations(); !errors.Is(err, ErrInvariant) {
		t.Fatalf("validation error = %v, want the shadowed shape rejected", err)
	}
}

func TestValidationAcceptsShapesThatEachRequireWhatTheOtherDoesNot(t *testing.T) {
	t.Parallel()
	registry := registerShapes(t, map[string]map[string]ValueType{
		"first":  {"alpha": ValueString},
		"second": {"beta": ValueBoolean},
	})
	if err := registry.ValidateDeclarations(); err != nil {
		t.Fatalf("validation error = %v, want shapes with disjoint keys accepted", err)
	}
}

func TestValidationRejectsTwoModulesDeclaringOneShape(t *testing.T) {
	t.Parallel()
	registry := registerShapes(t, map[string]map[string]ValueType{
		"first":  {"alpha": ValueString},
		"second": {"alpha": ValueString},
	})
	if err := registry.ValidateDeclarations(); !errors.Is(err, ErrInvariant) {
		t.Fatalf("validation error = %v, want the repeated shape rejected", err)
	}
}

func TestRegisterRejectsDuplicateIDs(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register(stubModule{id: "same"}); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := r.Register(stubModule{id: "same"}); err == nil {
		t.Fatal("expected an error registering the same id twice")
	}
}

func TestValidationRequiresAFileLimitOfAModuleThatReadsArchives(t *testing.T) {
	t.Parallel()
	declaration := testReaderDeclaration("archived", "character")
	declaration.Recognition = []Recognition{{
		Type: RecognitionEntry, Containers: []Container{ZIP}, Entry: "card.json",
	}}
	registry := NewRegistry()
	if err := registry.Register(declarationModule{stubModule: stubModule{id: "archived"}, declaration: declaration}); err != nil {
		t.Fatalf("register: %v", err)
	}
	if err := registry.ValidateDeclarations(); err == nil || !strings.Contains(err.Error(), "limit on their files") {
		t.Fatalf("validation error = %v, want an archive reader without a file limit rejected", err)
	}
}
