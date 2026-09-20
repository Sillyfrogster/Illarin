package lorebook

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/book"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

const (
	SillyTavernID             = "lorebook_sillytavern"
	sillyTavernEntryNamespace = "lorebook_sillytavern_entry"
)

const (
	stKeys          = "key"
	stSecondaryKeys = "keysecondary"
	stName          = "comment"
	stText          = "content"
	stDisable       = "disable"
	stOrder         = "order"
	stPosition      = "position"
	stConstant      = "constant"
	stSelective     = "selective"
	stCaseSensitive = "caseSensitive"
	stExclude       = "excludeRecursion"
	stPrevent       = "preventRecursion"
	stDelayUntil    = "delayUntilRecursion"
)

const (
	stBeforeCharacter = 0
	stAfterCharacter  = 1
)

type SillyTavernModule struct{}

func (SillyTavernModule) ID() string { return SillyTavernID }

func (SillyTavernModule) Declaration() format.Declaration {
	return format.Declaration{
		ID: SillyTavernID, Label: "SillyTavern lorebook", Type: Type,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Type:       format.RecognitionShape,
			Containers: []format.Container{format.JSON},
			Required:   map[string]format.ValueType{entriesKey: format.ValueObject},
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
		Header: nil,
		Slots:  nil,
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes,
		},
		ConsumedKeys:          []string{entriesKey},
		Boilerplate:           nil,
		Preservation:          format.PreservationDeclaration{Body: bookNamespace},
		TestedOriginalFormats: []string{SillyTavernID, format.OriginalFormatIllarin},
	}
}

func (m SillyTavernModule) Match(file format.Inspection) (format.Match, bool) {
	return format.MatchByDeclaration(file, m.Declaration())
}

func (m SillyTavernModule) Parse(
	_ context.Context,
	file format.Inspection,
	match format.Match,
) (format.Parsed, error) {
	payload, ok := match.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the matched payload is missing", SillyTavernID)
	}
	source := maps.Clone(payload.Root)

	var keyed map[string]json.RawMessage
	if raw, present := source[entriesKey]; !present || json.Unmarshal(raw, &keyed) != nil {
		return format.Parsed{}, format.MalformedInput(
			fmt.Errorf("%s entries: a World Info file's entries have to be an object", SillyTavernID),
		)
	}
	delete(source, entriesKey)

	entries := make([]block.Entry, 0, len(keyed))
	leftovers := make(map[uuid.UUID]json.RawMessage, len(keyed))
	for _, key := range entryOrder(keyed) {
		entry, kept := readSillyTavernEntry(keyed[key])
		entries = append(entries, entry)
		if len(kept) > 0 {
			leftovers[entry.ID] = kept
		}
	}

	element := block.Element{
		ID: uuid.New(), Type: block.TypeEntryTable, Role: block.RoleLorebookEntries,
		Content: block.EntryTable{Entries: entries},
	}
	return format.Parsed{
		Type: Type, Format: SillyTavernID,
		Elements:  []block.Element{element},
		Remainder: sillyTavernRemainder(source, entries, leftovers),
	}, nil
}

func entryOrder(keyed map[string]json.RawMessage) []string {
	keys := slices.Collect(maps.Keys(keyed))
	slices.SortFunc(keys, func(a, b string) int {
		first, firstNumeric := strconv.Atoi(a)
		second, secondNumeric := strconv.Atoi(b)
		if firstNumeric == nil && secondNumeric == nil {
			return first - second
		}
		if firstNumeric == nil {
			return -1
		}
		if secondNumeric == nil {
			return 1
		}
		return strings.Compare(a, b)
	})
	return keys
}

func readSillyTavernEntry(payload json.RawMessage) (block.Entry, json.RawMessage) {
	entry := block.Entry{ID: block.NewItemID(), Enabled: true}
	var fields map[string]json.RawMessage
	if json.Unmarshal(payload, &fields) != nil || fields == nil {
		return entry, payload
	}

	keys.Take(fields, stName, &entry.Name)
	keys.Take(fields, stKeys, &entry.Keys)
	keys.Take(fields, stSecondaryKeys, &entry.SecondaryKeys)
	keys.Take(fields, stSelective, &entry.Selective)
	keys.Take(fields, stCaseSensitive, &entry.CaseSensitive)
	keys.Take(fields, stConstant, &entry.Constant)
	keys.Take(fields, stOrder, &entry.Order)
	keys.Take(fields, stText, &entry.Text)
	keys.Take(fields, stExclude, &entry.Recursion.Exclude)
	keys.Take(fields, stPrevent, &entry.Recursion.Prevent)
	keys.Take(fields, stDelayUntil, &entry.Recursion.DelayUntil)

	var disabled bool
	if keys.Take(fields, stDisable, &disabled) {
		entry.Enabled = !disabled
	}

	var placement int
	if keys.Take(fields, stPosition, &placement) {
		switch placement {
		case stBeforeCharacter:
			entry.Position = block.BeforeCharacter
		case stAfterCharacter:
			entry.Position = block.AfterCharacter
		default:
			fields[stPosition], _ = json.Marshal(placement)
		}
	}
	if len(fields) == 0 {
		return entry, nil
	}
	kept, _ := json.Marshal(fields)
	return entry, kept
}

func sillyTavernRemainder(
	source map[string]json.RawMessage,
	entries []block.Entry,
	leftovers map[uuid.UUID]json.RawMessage,
) []format.Remainder {
	rows := make([]format.Remainder, 0, len(leftovers)+1)
	if len(source) > 0 {
		payload, _ := json.Marshal(source)
		rows = append(rows, format.Remainder{
			Owner: format.OwnerWork, Namespace: bookNamespace, Payload: payload,
		})
	}
	for _, entry := range entries {
		fields, held := leftovers[entry.ID]
		if !held {
			continue
		}
		rows = append(rows, format.Remainder{
			Owner: format.OwnerItem, OwnerID: entry.ID,
			Namespace: sillyTavernEntryNamespace, Payload: fields,
		})
	}
	return rows
}

func (SillyTavernModule) Write(
	_ context.Context,
	work format.ExportWork,
) (format.MainFile, error) {
	entries := bookEntries(work)
	written := make([]map[string]json.RawMessage, 0, len(entries))
	for _, entry := range entries {
		written = append(written, writeSillyTavernEntry(entry))
	}

	body := map[string]json.RawMessage{}
	position := make(map[uuid.UUID]int, len(entries))
	for index, entry := range entries {
		position[entry.ID] = index
	}
	for _, row := range work.Preserved {
		var fields map[string]json.RawMessage
		if json.Unmarshal(row.Payload, &fields) != nil {
			continue
		}
		switch {
		case row.Owner == format.OwnerWork && row.Namespace == bookNamespace:
			keys.MergeAbsent(body, fields)
		case row.Owner == format.OwnerItem && row.Namespace == sillyTavernEntryNamespace:
			index, kept := position[row.OwnerID]
			if !kept || index >= len(written) {
				continue
			}
			keys.MergeAbsent(written[index], fields)
		}
	}

	keyed := make(map[string]map[string]json.RawMessage, len(written))
	for index, fields := range written {
		keyed[strconv.Itoa(index)] = fields
	}
	body[entriesKey], _ = json.Marshal(keyed)

	document, err := json.Marshal(body)
	if err != nil {
		return format.MainFile{}, fmt.Errorf("write the world info file: %w", err)
	}
	return format.MainFile{
		Body: document, MediaType: "application/json", Extension: ".json",
	}, nil
}

func writeSillyTavernEntry(entry block.Entry) map[string]json.RawMessage {
	fields := map[string]json.RawMessage{
		stKeys:    keys.Must(book.OrEmptyStrings(entry.Keys)),
		stText:    keys.Must(entry.Text),
		stDisable: keys.Must(!entry.Enabled),
		stOrder:   keys.Must(entry.Order),
	}
	keys.WriteIfSet(fields, stName, entry.Name != "", entry.Name)
	keys.WriteIfSet(fields, stSecondaryKeys, len(entry.SecondaryKeys) > 0, entry.SecondaryKeys)
	keys.WriteIfSet(fields, stSelective, entry.Selective, entry.Selective)
	keys.WriteIfSet(fields, stCaseSensitive, entry.CaseSensitive, entry.CaseSensitive)
	keys.WriteIfSet(fields, stConstant, entry.Constant, entry.Constant)
	keys.WriteIfSet(fields, stExclude, entry.Recursion.Exclude, entry.Recursion.Exclude)
	keys.WriteIfSet(fields, stPrevent, entry.Recursion.Prevent, entry.Recursion.Prevent)
	keys.WriteIfSet(fields, stDelayUntil, entry.Recursion.DelayUntil, entry.Recursion.DelayUntil)
	if entry.Position != "" {
		placement := stBeforeCharacter
		if entry.Position == block.AfterCharacter {
			placement = stAfterCharacter
		}
		fields[stPosition] = keys.Must(placement)
	}
	return fields
}
