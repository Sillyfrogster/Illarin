package block

import (
	"fmt"

	"github.com/google/uuid"
)

func Place(kind string, tagged []Element) ([]Block, error) {
	definitions, ok := Catalog(kind)
	if !ok {
		return nil, fmt.Errorf("no block catalog for kind %q", kind)
	}

	placed := make([]bool, len(tagged))
	blocks := make([]Block, 0, len(definitions))
	for _, definition := range definitions {
		elements, err := definition.fill(tagged, placed)
		if err != nil {
			return nil, err
		}
		if len(elements) == 0 && !definition.Required {
			continue
		}
		layout, ok := definition.layoutFor(len(elements))
		if !ok {
			return nil, fmt.Errorf(
				"%s has %d elements and no layout with room for them",
				definition.ID, len(elements),
			)
		}
		for i := range elements {
			elements[i].Slot = layout.Slots()[i]
		}
		blocks = append(blocks, Block{
			ID:         uuid.New(),
			Definition: definition.ID,
			Position:   len(blocks),
			Layout:     layout,
			Width:      definition.Width,
			Elements:   elements,
		})
	}

	for i, element := range tagged {
		if !placed[i] {
			return nil, fmt.Errorf(
				"kind %q has nowhere to put a %s element", kind, element.Role,
			)
		}
	}
	return blocks, nil
}

func (d Definition) fill(tagged []Element, placed []bool) ([]Element, error) {
	elements := make([]Element, 0, len(d.Elements))
	for _, defined := range d.Elements {
		found, err := d.take(defined, tagged, placed)
		if err != nil {
			return nil, err
		}
		if found < 0 {
			if !defined.Pinned {
				continue
			}
			content, err := defined.Type.Empty()
			if err != nil {
				return nil, err
			}
			elements = append(elements, Element{
				ID: uuid.New(), Type: defined.Type, Role: defined.Role,
				Options: defined.Options, Content: content,
			})
			continue
		}
		element := tagged[found]
		if element.Type != defined.Type {
			return nil, fmt.Errorf(
				"%s carries a %s element and %s takes a %s",
				defined.Role, element.Type, d.ID, defined.Type,
			)
		}
		placed[found] = true
		if element.ID == uuid.Nil {
			element.ID = uuid.New()
		}
		element.Options = defined.Options
		elements = append(elements, element)
	}
	return elements, nil
}

func (d Definition) take(defined DefinedElement, tagged []Element, placed []bool) (int, error) {
	found := -1
	for i, candidate := range tagged {
		if placed[i] || candidate.Role != defined.Role || candidate.Type != defined.Type {
			continue
		}
		if found >= 0 {
			return 0, fmt.Errorf(
				"%s carries more than one %s element and the block has one place for it",
				d.ID, defined.Type,
			)
		}
		found = i
	}
	return found, nil
}

func (b Block) Pinned(role Role, kind string) bool {
	definition, ok := b.Definition.Definition(kind)
	if !ok {
		return false
	}
	defined, ok := definition.element(role)
	return ok && defined.Pinned
}
