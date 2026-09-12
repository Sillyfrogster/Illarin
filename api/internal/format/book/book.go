package book

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

func Read(payloads []json.RawMessage) ([]block.Entry, map[uuid.UUID]json.RawMessage) {
	entries := make([]block.Entry, 0, len(payloads))
	leftovers := make(map[uuid.UUID]json.RawMessage)
	for _, payload := range payloads {
		item := block.Entry{ID: block.NewItemID(), Enabled: true}
		var fields map[string]json.RawMessage
		if json.Unmarshal(payload, &fields) != nil || fields == nil {
			leftovers[item.ID] = payload
			entries = append(entries, item)
			continue
		}
		ReadEntry(fields, &item)
		entries = append(entries, item)
		if len(fields) > 0 {
			leftovers[item.ID], _ = json.Marshal(fields)
		}
	}
	return entries, leftovers
}

func ReadEntry(fields map[string]json.RawMessage, item *block.Entry) {
	keys.Take(fields, "name", &item.Name)
	keys.Take(fields, "keys", &item.Keys)
	keys.Take(fields, "secondary_keys", &item.SecondaryKeys)
	keys.Take(fields, "selective", &item.Selective)
	keys.Take(fields, "case_sensitive", &item.CaseSensitive)
	keys.Take(fields, "constant", &item.Constant)
	keys.Take(fields, "enabled", &item.Enabled)
	keys.Take(fields, "insertion_order", &item.Order)
	keys.Take(fields, "content", &item.Text)
	var position string
	if !keys.Take(fields, "position", &position) {
		return
	}
	switch position {
	case "", "before_char", "before_character":
		if position != "" {
			item.Position = block.BeforeCharacter
		}
	case "after_char", "after_character":
		item.Position = block.AfterCharacter
	default:
		fields["position"], _ = json.Marshal(position)
	}
}

func Write(entries []block.Entry) []map[string]json.RawMessage {
	written := make([]map[string]json.RawMessage, 0, len(entries))
	for _, entry := range entries {
		fields := map[string]json.RawMessage{
			"keys":            keys.Must(OrEmptyStrings(entry.Keys)),
			"content":         keys.Must(entry.Text),
			"enabled":         keys.Must(entry.Enabled),
			"insertion_order": keys.Must(entry.Order),
		}
		keys.WriteIfSet(fields, "name", entry.Name != "", entry.Name)
		keys.WriteIfSet(fields, "secondary_keys", len(entry.SecondaryKeys) > 0, entry.SecondaryKeys)
		keys.WriteIfSet(fields, "selective", entry.Selective, entry.Selective)
		keys.WriteIfSet(fields, "case_sensitive", entry.CaseSensitive, entry.CaseSensitive)
		keys.WriteIfSet(fields, "constant", entry.Constant, entry.Constant)
		keys.WriteIfSet(fields, "position", entry.Position != "", writtenPosition(entry.Position))
		written = append(written, fields)
	}
	return written
}

func writtenPosition(position block.EntryPosition) string {
	if position == block.AfterCharacter {
		return "after_char"
	}
	return "before_char"
}

func OrEmptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
