package lorebook

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/book"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

func (Module) Write(_ context.Context, asset format.ExportAsset) (format.Artifact, error) {
	entries := bookEntries(asset)
	body := map[string]json.RawMessage{
		"name":     keys.Must(asset.Header.Name),
		entriesKey: keys.Must(book.Write(entries)),
	}
	if err := restorePreserved(body, entries, asset.Preserved); err != nil {
		return format.Artifact{}, err
	}
	document, err := json.Marshal(body)
	if err != nil {
		return format.Artifact{}, fmt.Errorf("write the lorebook: %w", err)
	}
	return format.Artifact{
		Body: document, MediaType: "application/json", Extension: ".json",
	}, nil
}

func bookEntries(asset format.ExportAsset) []block.Entry {
	content, ok := asset.Content(block.RoleLorebookEntries)
	if !ok {
		return nil
	}
	table, isTable := content.(block.EntryTable)
	if !isTable {
		return nil
	}
	return table.Entries
}

func restorePreserved(
	body map[string]json.RawMessage,
	entries []block.Entry,
	rows []format.Remainder,
) error {
	extensions := keys.Object(body[extensionsKey])
	written, err := writtenEntries(body)
	if err != nil {
		return err
	}
	position := make(map[uuid.UUID]int, len(entries))
	for index, entry := range entries {
		position[entry.ID] = index
	}

	for _, row := range rows {
		var fields map[string]json.RawMessage
		if json.Unmarshal(row.Payload, &fields) != nil {
			fields = nil
		}
		switch {
		case row.Owner == format.OwnerAsset && row.Namespace == bookNamespace:
			keys.MergeAbsent(body, fields)
		case row.Owner == format.OwnerAsset:
			if _, held := extensions[row.Namespace]; !held {
				extensions[row.Namespace] = row.Payload
			}
		case row.Owner == format.OwnerItem && row.Namespace == entryNamespace:
			index, kept := position[row.OwnerID]
			if !kept || index >= len(written) {
				continue
			}
			keys.MergeAbsent(written[index], fields)
		}
	}

	if len(extensions) > 0 {
		body[extensionsKey], _ = json.Marshal(extensions)
	}
	body[entriesKey], _ = json.Marshal(written)
	return nil
}

func writtenEntries(body map[string]json.RawMessage) ([]map[string]json.RawMessage, error) {
	var entries []map[string]json.RawMessage
	if err := json.Unmarshal(body[entriesKey], &entries); err != nil {
		return nil, fmt.Errorf("read the written entries: %w", err)
	}
	return entries, nil
}
