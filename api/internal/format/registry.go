package format

import (
	"errors"
	"fmt"
	"slices"
)

var (
	ErrInvariant          = errors.New("format registry invariant")
	ErrConflictingMatches = fmt.Errorf("%w: conflicting authoritative matches", ErrInvariant)
	ErrAmbiguousMatches   = fmt.Errorf("%w: ambiguous format matches", ErrInvariant)
	ErrInvalidMatch       = fmt.Errorf("%w: invalid format match", ErrInvariant)
	ErrUnsupportedFormat  = errors.New("unsupported format")
)

type Resolution struct {
	Module Reader
	Match  Match
}

type Registry struct {
	modules map[string]Module
}

func NewRegistry() *Registry {
	return &Registry{modules: make(map[string]Module)}
}

func (r *Registry) Empty() bool { return len(r.modules) == 0 }

func (r *Registry) Register(m Module) error {
	if _, taken := r.modules[m.ID()]; taken {
		return fmt.Errorf("format module %q is already registered", m.ID())
	}
	r.modules[m.ID()] = m
	return nil
}

func (r *Registry) ValidateDeclarations() error {
	ids := make([]string, 0, len(r.modules))
	for id := range r.modules {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		module := r.modules[id]
		declaration := module.Declaration()
		if declaration.ID != id {
			return fmt.Errorf("module %q declares identity %q: %w", id, declaration.ID, ErrInvariant)
		}
		if err := ValidateDeclaration(declaration); err != nil {
			return fmt.Errorf("module %q declaration: %w", id, err)
		}
		if declaration.Direction.Read && declaration.Input == InputFile {
			if _, ok := module.(Reader); !ok {
				return fmt.Errorf("module %q declares read support without a reader: %w", id, ErrInvariant)
			}
		}
		if declaration.Direction.Read && declaration.Input == InputDatabaseRow {
			if _, ok := module.(DatabaseReader); !ok {
				return fmt.Errorf("module %q declares row read support without a database reader: %w", id, ErrInvariant)
			}
		}
		if declaration.Direction.Write {
			if _, ok := module.(Writer); !ok {
				return fmt.Errorf("module %q declares write support without a writer: %w", id, ErrInvariant)
			}
		}
	}
	for i, firstID := range ids {
		first := r.modules[firstID].Declaration()
		for _, secondID := range ids[i+1:] {
			second := r.modules[secondID].Declaration()
			if shapesOverlap(first.Recognition, second.Recognition) {
				return fmt.Errorf(
					"modules %q and %q have overlapping shapes: %w",
					firstID, secondID, ErrInvariant,
				)
			}
		}
	}
	return nil
}

func shapesOverlap(first, second []Recognition) bool {
	for _, a := range first {
		if a.Type != RecognitionShape {
			continue
		}
		for _, b := range second {
			if b.Type != RecognitionShape || incompatibleContainers(a.Containers, b.Containers) {
				continue
			}
			if shadows(a.Required, b.Required) || shadows(b.Required, a.Required) {
				return true
			}
		}
	}
	return false
}

func shadows(looser, stricter map[string]ValueType) bool {
	for key, wanted := range looser {
		if held, present := stricter[key]; !present || held != wanted {
			return false
		}
	}
	return true
}

func incompatibleContainers(first, second []Container) bool {
	if len(first) == 0 || len(second) == 0 {
		return false
	}
	for _, container := range first {
		if slices.Contains(second, container) {
			return false
		}
	}
	return true
}

func (r *Registry) ByID(id string) (Module, bool) {
	m, ok := r.modules[id]
	return m, ok
}

func (r *Registry) Declaration(id string) (Declaration, bool) {
	module, ok := r.modules[id]
	if !ok {
		return Declaration{}, false
	}
	return module.Declaration(), true
}

func (r *Registry) ReadableLabels() []string {
	ids := make([]string, 0, len(r.modules))
	for id := range r.modules {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	labels := make([]string, 0, len(ids))
	for _, id := range ids {
		declaration := r.modules[id].Declaration()
		if declaration.Direction.Read && declaration.Input == InputFile {
			labels = append(labels, declaration.Label)
		}
	}
	return labels
}

func (r *Registry) Resolve(file Inspection) (Resolution, bool, error) {
	var candidates []Resolution
	for _, module := range r.modules {
		declaration := module.Declaration()
		if !declaration.Direction.Read {
			continue
		}
		reader, ok := module.(Reader)
		if !ok {
			continue
		}
		match, ok := reader.Match(file)
		if !ok {
			continue
		}
		if err := validateMatch(file, reader, match); err != nil {
			return Resolution{}, false, fmt.Errorf("module %q: %w", module.ID(), err)
		}
		candidates = append(candidates, Resolution{Module: reader, Match: match})
	}
	if len(candidates) == 0 {
		if err := r.unsupportedMarker(file); err != nil {
			return Resolution{}, false, err
		}
		return Resolution{}, false, nil
	}

	for i, first := range candidates {
		if first.Match.strength != authoritative {
			continue
		}
		for _, second := range candidates[i+1:] {
			if second.Match.strength == authoritative && second.Match.payloadID == first.Match.payloadID {
				return Resolution{}, false, fmt.Errorf(
					"modules %q and %q matched payload %d: %w",
					first.Module.ID(), second.Module.ID(), first.Match.payloadID, ErrConflictingMatches,
				)
			}
		}
	}

	strongest := candidates[0].Match.strength
	for _, candidate := range candidates[1:] {
		strongest = max(strongest, candidate.Match.strength)
	}
	var winners []Resolution
	for _, candidate := range candidates {
		if candidate.Match.strength == strongest {
			winners = append(winners, candidate)
		}
	}
	if len(winners) > 1 {
		return Resolution{}, false, fmt.Errorf(
			"modules %q and %q made equally strong matches: %w",
			winners[0].Module.ID(), winners[1].Module.ID(), ErrAmbiguousMatches,
		)
	}
	return winners[0], true, nil
}

func (r *Registry) unsupportedMarker(file Inspection) error {
	type observation struct {
		workType string
		path     string
		value    string
		formats  []string
	}
	observations := make(map[string]*observation)
	ids := make([]string, 0, len(r.modules))
	for id := range r.modules {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		declaration := r.modules[id].Declaration()
		for _, recognition := range declaration.Recognition {
			if recognition.Type != RecognitionMarker {
				continue
			}
			for _, payload := range file.Payloads {
				if len(recognition.Containers) > 0 &&
					!slices.Contains(recognition.Containers, payload.Location.Container) {
					continue
				}
				value, present := payloadValue(payload.Root, recognition.Path)
				if !present || slices.Contains(recognition.Values, value) {
					continue
				}
				path := slices.Concat(recognition.Path)
				joined := ""
				for i, part := range path {
					if i > 0 {
						joined += "."
					}
					joined += part
				}
				key := declaration.Type + "\x00" + joined + "\x00" + value
				found := observations[key]
				if found == nil {
					found = &observation{workType: declaration.Type, path: joined, value: value}
					observations[key] = found
				}
				found.formats = append(found.formats, declaration.ID)
			}
		}
	}
	if len(observations) == 0 {
		return nil
	}
	keys := make([]string, 0, len(observations))
	for key := range observations {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	found := observations[keys[0]]
	if len(found.formats) == 1 {
		return fmt.Errorf(
			"format %q recognises marker %q but cannot read value %q: %w",
			found.formats[0], found.path, found.value, ErrUnsupportedFormat,
		)
	}
	return fmt.Errorf(
		"formats for type %q recognise marker %q but cannot read value %q: %w",
		found.workType, found.path, found.value, ErrUnsupportedFormat,
	)
}

func validateMatch(file Inspection, module Reader, match Match) error {
	if match.strength != compatibility && match.strength != authoritative {
		return fmt.Errorf("strength %d: %w", match.strength, ErrInvalidMatch)
	}
	if _, ok := match.Payload(file); !ok {
		return fmt.Errorf("payload %d does not exist: %w", match.payloadID, ErrInvalidMatch)
	}
	if match.strength == authoritative && !ownsSpec(module, match.formatID) {
		return fmt.Errorf("payload names format %q, module is %q: %w", match.formatID, module.ID(), ErrInvalidMatch)
	}
	if match.strength == compatibility && match.formatID != "" {
		return fmt.Errorf("compatibility match names format %q: %w", match.formatID, ErrInvalidMatch)
	}
	return nil
}

// Declarations lists every module's declaration, sorted by id
func (r *Registry) Declarations() []Declaration {
	ids := make([]string, 0, len(r.modules))
	for id := range r.modules {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	declarations := make([]Declaration, 0, len(ids))
	for _, id := range ids {
		declarations = append(declarations, r.modules[id].Declaration())
	}
	return declarations
}
