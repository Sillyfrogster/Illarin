package block

import (
	"fmt"

	"github.com/google/uuid"
)

type Offer struct {
	Definition DefinitionID
	Title      string
	Summary    string
	Group      Group
	Repeatable bool
	Choices    []Choice
}

type Choice struct {
	Type  Type
	Label string
}

func Offers(workType string) ([]Offer, bool) {
	definitions, ok := Definitions(workType)
	if !ok {
		return nil, false
	}
	offers := make([]Offer, 0, len(definitions))
	for _, definition := range definitions {
		if definition.Required {
			continue
		}
		starts := definition.starts()
		choices := make([]Choice, 0, len(starts))
		for _, option := range starts {
			choices = append(choices, Choice{Type: option.Type, Label: option.Label})
		}
		offers = append(offers, Offer{
			Definition: definition.ID,
			Title:      definition.Title,
			Summary:    definition.Summary,
			Group:      definition.Group,
			Repeatable: definition.Repeatable,
			Choices:    choices,
		})
	}
	return offers, true
}

func NewBlock(workType string, id DefinitionID, elementType Type, page []Block) (Block, error) {
	definition, ok := id.Definition(workType)
	if !ok {
		return Block{}, fmt.Errorf("a %s has no %s block", workType, id)
	}
	if definition.Required {
		return Block{}, fmt.Errorf("%s is on every %s already", definition.Title, workType)
	}
	if !definition.Repeatable {
		for _, holder := range page {
			if holder.Definition == id {
				return Block{}, fmt.Errorf("%s is already on this page", definition.Title)
			}
		}
	}
	var chosen start
	found := false
	for _, option := range definition.starts() {
		if option.Type == elementType {
			chosen, found = option, true
			break
		}
	}
	if !found {
		return Block{}, fmt.Errorf("%s does not hold %s content", definition.Title, elementType)
	}
	layout, ok := definition.layoutFor(len(chosen.Elements))
	if !ok {
		return Block{}, fmt.Errorf(
			"%s has no layout with room for %d elements", id, len(chosen.Elements),
		)
	}
	elements := make([]Element, 0, len(chosen.Elements))
	for i, defined := range chosen.Elements {
		content, err := defined.Type.Empty()
		if err != nil {
			return Block{}, err
		}
		elements = append(elements, Element{
			ID:      uuid.New(),
			Type:    defined.Type,
			Role:    defined.Role,
			Slot:    layout.Slots()[i],
			Options: defined.Options,
			Content: content,
		})
	}
	return Block{
		ID:         uuid.New(),
		Definition: id,
		Position:   len(page),
		Layout:     layout,
		Width:      definition.Width,
		Elements:   elements,
	}, nil
}
