package character

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/book"
	"github.com/google/uuid"
)

const (
	cardNamespace = "card"
	extensionsKey = "extensions"
	bookKey       = "character_book"
)

type lorebook struct {
	found            bool
	insideExtensions bool
	table            block.EntryTable
	bookFields       json.RawMessage
	entryFields      map[uuid.UUID]json.RawMessage
}

func (l lorebook) preserved(elements []block.Element) []format.Remainder {
	if !l.found {
		return nil
	}
	var holder uuid.UUID
	for _, element := range elements {
		if element.Role == block.RoleLorebookEntries {
			holder = element.ID
			break
		}
	}
	rows := make([]format.Remainder, 0, len(l.entryFields)+1)
	if len(l.bookFields) > 0 {
		rows = append(rows, format.Remainder{
			Owner: format.OwnerElement, OwnerID: holder,
			Namespace: bookKey, Payload: l.bookFields,
		})
	}
	for _, entry := range l.table.Entries {
		fields, ok := l.entryFields[entry.ID]
		if !ok {
			continue
		}
		rows = append(rows, format.Remainder{
			Owner: format.OwnerItem, OwnerID: entry.ID,
			Namespace: bookKey, Payload: fields,
		})
	}
	return rows
}

func (c card) lorebook() lorebook {
	raw := c.fields[bookKey]
	insideExtensions := false
	if len(raw) == 0 {
		raw = c.extensions()[bookKey]
		insideExtensions = true
	}
	var source map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &source) != nil {
		return lorebook{}
	}
	var entryPayloads []json.RawMessage
	if entriesRaw, present := source["entries"]; !present || json.Unmarshal(entriesRaw, &entryPayloads) != nil {
		return lorebook{}
	}
	delete(source, "entries")

	entries, entryFields := book.Read(entryPayloads)
	read := lorebook{
		found:            true,
		insideExtensions: insideExtensions,
		table:            block.EntryTable{Entries: entries},
		entryFields:      entryFields,
	}
	if len(source) > 0 {
		read.bookFields, _ = json.Marshal(source)
	}
	return read
}
