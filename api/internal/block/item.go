package block

import (
	"encoding/json"

	"github.com/google/uuid"
)

func NewItemID() uuid.UUID { return uuid.New() }

func itemID(supplied uuid.UUID) uuid.UUID {
	if supplied != uuid.Nil {
		return supplied
	}
	return NewItemID()
}

func ItemIDs(content Content) []uuid.UUID {
	switch held := content.(type) {
	case TextSet:
		return collectItemIDs(held.Texts, func(item TextItem) uuid.UUID { return item.ID })
	case DialogueSample:
		return collectItemIDs(held.Turns, func(item DialogueTurn) uuid.UUID { return item.ID })
	case ImageSet:
		return collectItemIDs(held.Images, func(item ImageItem) uuid.UUID { return item.ID })
	case FieldList:
		return collectItemIDs(held.Fields, func(item FieldItem) uuid.UUID { return item.ID })
	case LinkList:
		return collectItemIDs(held.Links, func(item LinkItem) uuid.UUID { return item.ID })
	case EntryTable:
		return collectItemIDs(held.Entries, func(item Entry) uuid.UUID { return item.ID })
	case PromptList:
		return append(
			collectItemIDs(held.Groups, func(item PromptGroup) uuid.UUID { return item.ID }),
			collectItemIDs(held.Fragments, func(item PromptFragment) uuid.UUID { return item.ID })...,
		)
	case VariableSchema:
		return collectItemIDs(held.Variables, func(item Variable) uuid.UUID { return item.ID })
	case SettingGroup:
		return collectItemIDs(held.Settings, func(item Setting) uuid.UUID { return item.ID })
	case ScriptList:
		return collectItemIDs(held.Scripts, func(item Script) uuid.UUID { return item.ID })
	case ColorSet:
		var ids []uuid.UUID
		for _, mode := range held.Modes {
			ids = append(ids, collectItemIDs(mode.Colors, func(item Color) uuid.UUID { return item.ID })...)
		}
		return ids
	case StylesheetSet:
		return append(
			collectItemIDs(held.Stylesheets, func(item Stylesheet) uuid.UUID { return item.ID }),
			collectItemIDs(held.Files, func(item StylesheetFile) uuid.UUID { return item.ID })...,
		)
	case RecordList:
		return collectItemIDs(held.Records, func(item LumiaRecord) uuid.UUID { return item.ID })
	default:
		return nil
	}
}

func collectItemIDs[T any](items []T, id func(T) uuid.UUID) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, id(item))
	}
	return ids
}

func mintItemIDs(list string) func(json.RawMessage) (json.RawMessage, error) {
	return func(stored json.RawMessage) (json.RawMessage, error) {
		var content map[string]json.RawMessage
		if err := json.Unmarshal(stored, &content); err != nil {
			return nil, err
		}
		var items []map[string]json.RawMessage
		if err := json.Unmarshal(content[list], &items); err != nil {
			return nil, err
		}
		for _, item := range items {
			var id uuid.UUID
			if json.Unmarshal(item["id"], &id) == nil && id != uuid.Nil {
				continue
			}
			minted, err := json.Marshal(NewItemID())
			if err != nil {
				return nil, err
			}
			item["id"] = minted
		}
		upgraded, err := json.Marshal(items)
		if err != nil {
			return nil, err
		}
		content[list] = upgraded
		return json.Marshal(content)
	}
}
