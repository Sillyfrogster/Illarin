package block

func ToBlocks(kind string, blocks []Block) ([]AssetBlock, error) {
	out := make([]AssetBlock, 0, len(blocks))
	for _, b := range blocks {
		definition, _ := b.Definition.Definition(kind)
		title, isDefault := definition.Title, true
		if b.Title != nil {
			title, isDefault = *b.Title, false
		}
		elements, err := toAPIElements(kind, b)
		if err != nil {
			return nil, err
		}
		out = append(out, AssetBlock{
			Id:             b.ID,
			Definition:     string(b.Definition),
			Title:          title,
			TitleIsDefault: isDefault,
			Position:       b.Position,
			Hidden:         b.Hidden,
			Layout:         AssetBlockLayout(b.Layout),
			Width:          AssetBlockWidth(b.Width),
			AllowedLayouts: apiLayouts(definition.Layouts),
			Required:       definition.Required,
			Hideable:       !definition.Required || definition.Hideable,
			IsEmpty:        b.Empty(),
			Elements:       elements,
		})
	}
	return out, nil
}

func apiLayouts(layouts []Layout) []AssetBlockAllowedLayouts {
	out := make([]AssetBlockAllowedLayouts, len(layouts))
	for i, layout := range layouts {
		out[i] = AssetBlockAllowedLayouts(layout)
	}
	return out
}

func toAPIElements(kind string, holder Block) ([]AssetElement, error) {
	out := make([]AssetElement, 0, len(holder.Elements))
	for _, element := range holder.Elements {
		content, err := element.ContentJSON()
		if err != nil {
			return nil, err
		}
		facts := element.Facts()
		if facts == nil {
			facts = []string{}
		}
		served := AssetElement{
			Id:      element.ID,
			Type:    ElementType(element.Type),
			Slot:    string(element.Slot),
			Label:   element.Label(),
			Pinned:  holder.Pinned(element.Role, kind),
			Locked:  holder.Locked(element.Role, kind),
			IsEmpty: element.Content == nil || element.Content.Empty(),
			Facts:   facts,
			Content: content,
		}
		if element.Role != "" {
			role := string(element.Role)
			served.Role = &role
		}
		if element.Options.Display != "" {
			display := AssetElementDisplay(element.Options.Display)
			served.Display = &display
		}
		if element.Options.ItemSize != "" {
			size := AssetElementItemSize(element.Options.ItemSize)
			served.ItemSize = &size
		}
		out = append(out, served)
	}
	return out, nil
}
