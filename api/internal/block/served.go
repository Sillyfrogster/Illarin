package block

func ToBlocks(workType string, blocks []Block) ([]WorkBlock, error) {
	out := make([]WorkBlock, 0, len(blocks))
	for _, b := range blocks {
		definition, _ := b.Definition.Definition(workType)
		title, isDefault := definition.Title, true
		if b.Title != nil {
			title, isDefault = *b.Title, false
		}
		elements, err := toAPIElements(workType, b)
		if err != nil {
			return nil, err
		}
		out = append(out, WorkBlock{
			Id:             b.ID,
			Definition:     string(b.Definition),
			Title:          title,
			TitleIsDefault: isDefault,
			Position:       b.Position,
			Hidden:         b.Hidden,
			Layout:         WorkBlockLayout(b.Layout),
			Width:          WorkBlockWidth(b.Width),
			AllowedLayouts: apiLayouts(definition.Layouts),
			Required:       definition.Required,
			Hideable:       !definition.Required || definition.Hideable,
			IsEmpty:        b.Empty(),
			Elements:       elements,
		})
	}
	return out, nil
}

func apiLayouts(layouts []Layout) []WorkBlockAllowedLayouts {
	out := make([]WorkBlockAllowedLayouts, len(layouts))
	for i, layout := range layouts {
		out[i] = WorkBlockAllowedLayouts(layout)
	}
	return out
}

func toAPIElements(workType string, holder Block) ([]WorkElement, error) {
	out := make([]WorkElement, 0, len(holder.Elements))
	for _, element := range holder.Elements {
		content, err := element.ContentJSON()
		if err != nil {
			return nil, err
		}
		facts := element.Facts()
		if facts == nil {
			facts = []string{}
		}
		served := WorkElement{
			Id:       element.ID,
			Type:     ElementType(element.Type),
			Slot:     string(element.Slot),
			Label:    element.Label(),
			Pinned:   holder.Pinned(element.Role, workType),
			FromFile: holder.FromFile(element.Role, workType),
			IsEmpty:  element.Content == nil || element.Content.Empty(),
			Facts:    facts,
			Content:  content,
		}
		if element.Role != "" {
			role := string(element.Role)
			served.Role = &role
		}
		if element.Options.Display != "" {
			display := WorkElementDisplay(element.Options.Display)
			served.Display = &display
		}
		if element.Options.ItemSize != "" {
			size := WorkElementItemSize(element.Options.ItemSize)
			served.ItemSize = &size
		}
		out = append(out, served)
	}
	return out, nil
}
