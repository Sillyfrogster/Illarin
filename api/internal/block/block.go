package block

import "github.com/google/uuid"

type Block struct {
	ID         uuid.UUID
	Definition DefinitionID
	Title      *string
	Position   int
	Hidden     bool
	Layout     Layout
	Width      Width
	Elements   []Element
}

func (b Block) Empty() bool {
	for _, element := range b.Elements {
		if element.Content != nil && !element.Content.Empty() {
			return false
		}
	}
	return true
}
