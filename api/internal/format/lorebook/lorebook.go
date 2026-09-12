package lorebook

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"unicode"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/book"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/google/uuid"
)

const (
	ID   = "lorebook"
	Kind = "lorebook"
)

const (
	bookNamespace  = "lorebook"
	entryNamespace = "lorebook_entry"
	extensionsKey  = "extensions"
	entriesKey     = "entries"
)

type Module struct{}

func (Module) ID() string { return ID }

func (Module) Declaration() format.Declaration {
	return format.Declaration{
		ID: ID, Label: "Lorebook", Kind: Kind,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Kind:       format.RecognitionSignature,
			Containers: []probe.Container{probe.JSON},
			Required:   map[string]format.ValueType{entriesKey: format.ValueArray},
		}},
		Roles: map[block.Role]format.DirectionalRoleSupport{
			block.RoleLorebookEntries: {
				Read:  format.RoleSupport{Grade: format.SupportFull},
				Write: format.RoleSupport{Grade: format.SupportFull},
			},
			block.RoleGallery: {
				Read:  format.RoleSupport{Grade: format.SupportNone},
				Write: format.RoleSupport{Grade: format.SupportNone},
			},
			block.RoleCreatorNotes: {
				Read:  format.RoleSupport{Grade: format.SupportNone},
				Write: format.RoleSupport{Grade: format.SupportNone},
			},
		},
		Header: []format.HeaderField{format.HeaderName},
		Slots:  nil,
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes,
		},
		ConsumedKeys: []string{entriesKey, "name"},
		Boilerplate:  nil,
		Preservation: format.PreservationDeclaration{
			Body: bookNamespace, Container: []string{extensionsKey},
		},
		TestedOrigins:    []string{ID, format.OriginIllarin, format.OriginV1},
		PreservesOrigins: []string{format.OriginV1},
	}
}

func (m Module) Claim(file probe.Inspection) (format.Claim, bool) {
	return format.ClaimByDeclaration(file, m.Declaration())
}

func (m Module) Parse(
	_ context.Context,
	file probe.Inspection,
	claim format.Claim,
) (format.Parsed, error) {
	payload, ok := claim.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the claimed payload is missing", ID)
	}
	source := maps.Clone(payload.Root)

	var entryPayloads []json.RawMessage
	if raw, present := source[entriesKey]; !present || json.Unmarshal(raw, &entryPayloads) != nil {
		return format.Parsed{}, format.MalformedInput(
			fmt.Errorf("%s entries: a lorebook's entries have to be a list", ID),
		)
	}
	delete(source, entriesKey)

	entries, entryFields := book.Read(entryPayloads)
	element := block.Element{
		ID: uuid.New(), Type: block.TypeEntryTable, Role: block.RoleLorebookEntries,
		Content: block.EntryTable{Entries: entries},
	}

	name, named := text(source, "name")
	if named {
		delete(source, "name")
	}
	seeded := blurb(source)
	return format.Parsed{
		Kind: Kind, Format: ID,
		Header:    format.Header{Name: strings.TrimSpace(name), Blurb: seeded},
		Elements:  []block.Element{element},
		Remainder: remainder(source, entries, entryFields),
	}, nil
}

func remainder(
	source map[string]json.RawMessage,
	entries []block.Entry,
	entryFields map[uuid.UUID]json.RawMessage,
) []format.Remainder {
	extensions := make(map[string]json.RawMessage)
	if raw, held := source[extensionsKey]; held {
		if json.Unmarshal(raw, &extensions) == nil {
			delete(source, extensionsKey)
		} else {
			extensions = make(map[string]json.RawMessage)
		}
	}
	if collision, clash := extensions[bookNamespace]; clash {
		source[extensionsKey], _ = json.Marshal(
			map[string]json.RawMessage{bookNamespace: collision},
		)
		delete(extensions, bookNamespace)
	}

	rows := make([]format.Remainder, 0, len(extensions)+len(entryFields)+1)
	if len(source) > 0 {
		payload, _ := json.Marshal(source)
		rows = append(rows, format.Remainder{
			Owner: format.OwnerAsset, Namespace: bookNamespace, Payload: payload,
		})
	}
	for _, namespace := range slices.Sorted(maps.Keys(extensions)) {
		rows = append(rows, format.Remainder{
			Owner: format.OwnerAsset, Namespace: namespace, Payload: extensions[namespace],
		})
	}
	for _, entry := range entries {
		fields, held := entryFields[entry.ID]
		if !held {
			continue
		}
		rows = append(rows, format.Remainder{
			Owner: format.OwnerItem, OwnerID: entry.ID,
			Namespace: entryNamespace, Payload: fields,
		})
	}
	return rows
}

func blurb(source map[string]json.RawMessage) string {
	description, _ := text(source, "description")
	return truncate(strings.TrimSpace(description), format.MaxBlurbRunes)
}

func text(source map[string]json.RawMessage, name string) (string, bool) {
	raw, present := source[name]
	if !present {
		return "", false
	}
	var value string
	if json.Unmarshal(raw, &value) != nil {
		return "", false
	}
	return value, true
}

func truncate(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	cut := string(runes[:limit])
	if space := strings.LastIndexFunc(cut, unicode.IsSpace); space > 0 {
		cut = cut[:space]
	}
	return strings.TrimSpace(cut)
}

func Modules() []format.Reader { return []format.Reader{Module{}, SillyTavernModule{}} }
