package pack

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

const (
	ID   = "pack_lumiverse"
	Type = "pack"

	packNamespace = ID
	itemNamespace = ID + "_item"
)

type Module struct{}

func (Module) ID() string { return ID }

func (Module) Declaration() format.Declaration {
	return format.Declaration{
		ID: ID, Label: "Lumiverse pack", Type: Type,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Type: format.RecognitionShape, Containers: []format.Container{format.JSON},
			Required: map[string]format.ValueType{
				"packName": format.ValueString, "lumiaItems": format.ValueArray,
				"loomItems": format.ValueArray,
			},
		}},
		Roles: map[block.Role]format.DirectionalRoleSupport{
			block.RolePackItems: {
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
		Header: []format.HeaderField{
			format.HeaderName, format.HeaderWorkVersion, format.HeaderCreditedAuthor,
		},
		Slots: []format.SlotDeclaration{
			{Name: "lumiaName", Type: format.ValueString},
			{Name: "lumiaDefinition", Type: format.ValueString},
			{Name: "lumiaPersonality", Type: format.ValueString},
			{Name: "lumiaBehavior", Type: format.ValueString},
			{Name: "avatarUrl", Type: format.ValueString},
			{Name: "genderIdentity", Type: format.ValueNumber},
			{Name: "authorName", Type: format.ValueString},
			{Name: "version", Type: format.ValueNumber},
		},
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes,
		},
		ConsumedKeys:             []string{"packName", "packAuthor", "version", "lumiaItems"},
		Preservation:             format.PreservationDeclaration{Body: packNamespace},
		TestedOriginalFormats:    []string{ID, format.OriginalFormatIllarin, format.OriginalFormatV1},
		PreservesOriginalFormats: []string{format.OriginalFormatV1},
	}
}

func (module Module) Match(file format.Inspection) (format.Match, bool) {
	return format.MatchByDeclaration(file, module.Declaration())
}

func (Module) Parse(
	_ context.Context,
	file format.Inspection,
	match format.Match,
) (format.Parsed, error) {
	payload, ok := match.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the matched payload is missing", ID)
	}
	source := maps.Clone(payload.Root)
	var name string
	if !keys.Take(source, "packName", &name) || strings.TrimSpace(name) == "" {
		return format.Parsed{}, format.MalformedInput(fmt.Errorf("%s name: packName is required", ID))
	}
	header := format.Header{Name: strings.TrimSpace(name)}
	keys.Take(source, "packAuthor", &header.CreditedAuthor)
	var version json.Number
	if keys.Take(source, "version", &version) {
		header.WorkVersion = version.String()
	}

	var rawItems []json.RawMessage
	if raw, present := source["lumiaItems"]; !present || json.Unmarshal(raw, &rawItems) != nil {
		return format.Parsed{}, format.MalformedInput(fmt.Errorf("%s items: lumiaItems must be a list", ID))
	}
	delete(source, "lumiaItems")
	records, leftovers, unread := readItems(rawItems)
	if len(records) == 0 {
		return format.Parsed{}, format.MalformedInput(fmt.Errorf("%s items: at least one Lumia item is required", ID))
	}
	if len(unread) > 0 {
		source["lumiaItems"] = keys.Must(unread)
	}
	element := block.Element{
		ID: uuid.New(), Type: block.TypeRecordList, Role: block.RolePackItems,
		Content: block.RecordList{Schema: block.LumiaRecordSchema, Records: records},
	}
	return format.Parsed{
		Type: Type, Format: ID, Header: header, Elements: []block.Element{element},
		Remainder: remainder(source, leftovers),
	}, nil
}

func readItems(rawItems []json.RawMessage) (
	[]block.LumiaRecord,
	map[uuid.UUID]map[string]json.RawMessage,
	[]json.RawMessage,
) {
	records := make([]block.LumiaRecord, 0, len(rawItems))
	leftovers := make(map[uuid.UUID]map[string]json.RawMessage)
	unread := make([]json.RawMessage, 0)
	for _, rawItem := range rawItems {
		fields := make(map[string]json.RawMessage)
		if json.Unmarshal(rawItem, &fields) != nil {
			unread = append(unread, rawItem)
			continue
		}
		var name string
		if !keys.Take(fields, "lumiaName", &name) || strings.TrimSpace(name) == "" {
			unread = append(unread, rawItem)
			continue
		}
		record := block.LumiaRecord{
			ID: block.NewItemID(), LumiaName: name, GenderIdentity: 2, Version: 1,
		}
		keys.Take(fields, "lumiaDefinition", &record.LumiaDefinition)
		keys.Take(fields, "lumiaPersonality", &record.LumiaPersonality)
		keys.Take(fields, "lumiaBehavior", &record.LumiaBehavior)
		keys.Take(fields, "authorName", &record.AuthorName)
		var gender int
		if keys.Take(fields, "genderIdentity", &gender) {
			if gender >= 0 && gender <= 2 {
				record.GenderIdentity = gender
			} else {
				fields["genderIdentity"] = keys.Must(gender)
			}
		}
		var version int
		if keys.Take(fields, "version", &version) {
			if version > 0 {
				record.Version = version
			} else {
				fields["version"] = keys.Must(version)
			}
		}
		if len(fields) > 0 {
			leftovers[record.ID] = fields
		}
		records = append(records, record)
	}
	return records, leftovers, unread
}

func remainder(
	source map[string]json.RawMessage,
	items map[uuid.UUID]map[string]json.RawMessage,
) []format.Remainder {
	rows := make([]format.Remainder, 0, len(items)+1)
	if len(source) > 0 {
		rows = append(rows, format.Remainder{
			Owner: format.OwnerWork, Namespace: packNamespace, Payload: keys.Must(source),
		})
	}
	for id, fields := range items {
		rows = append(rows, format.Remainder{
			Owner: format.OwnerItem, OwnerID: id, Namespace: itemNamespace,
			Payload: keys.Must(fields),
		})
	}
	return rows
}

func (Module) Write(_ context.Context, work format.ExportWork) (format.MainFile, error) {
	body, itemFields := preserved(work.Preserved)
	var unread []json.RawMessage
	_ = json.Unmarshal(body["lumiaItems"], &unread)
	delete(body, "lumiaItems")

	body["packName"] = keys.Must(work.Header.Name)
	if _, unread := body["packAuthor"]; !unread || work.Header.CreditedAuthor != "" {
		body["packAuthor"] = keys.Must(work.Header.CreditedAuthor)
	}
	_, unreadVersion := body["version"]
	if work.Header.WorkVersion == "" && !unreadVersion {
		body["version"] = keys.Must(1)
	} else if work.Header.WorkVersion != "" {
		if _, err := strconv.ParseFloat(work.Header.WorkVersion, 64); err == nil {
			body["version"] = json.RawMessage(work.Header.WorkVersion)
		} else {
			body["version"] = keys.Must(work.Header.WorkVersion)
		}
	}
	if work.Cover != nil && work.Cover.URL != "" {
		body["coverUrl"] = keys.Must(work.Cover.URL)
	}
	if _, present := body["packExtras"]; !present {
		body["packExtras"] = keys.Must([]any{})
	}
	if _, present := body["loomItems"]; !present {
		body["loomItems"] = keys.Must([]any{})
	}

	items := make([]json.RawMessage, 0, len(unread))
	if content, ok := work.Content(block.RolePackItems); ok {
		if list, isList := content.(block.RecordList); isList && list.Schema == block.LumiaRecordSchema {
			for _, record := range list.Records {
				fields := map[string]json.RawMessage{
					"lumiaName": keys.Must(record.LumiaName),
				}
				unread := itemFields[record.ID]
				writeModeledUnlessUnread(fields, unread, "lumiaDefinition", record.LumiaDefinition, "")
				writeModeledUnlessUnread(fields, unread, "lumiaPersonality", record.LumiaPersonality, "")
				writeModeledUnlessUnread(fields, unread, "lumiaBehavior", record.LumiaBehavior, "")
				writeModeledUnlessUnread(fields, unread, "genderIdentity", record.GenderIdentity, 2)
				writeModeledUnlessUnread(fields, unread, "authorName", record.AuthorName, "")
				writeModeledUnlessUnread(fields, unread, "version", record.Version, 1)
				if record.AvatarURL != nil {
					if image, found := work.Images[*record.AvatarURL]; found && image.URL != "" {
						fields["avatarUrl"] = keys.Must(image.URL)
					}
				}
				keys.MergeAbsent(fields, unread)
				items = append(items, keys.Must(fields))
			}
		}
	}
	items = append(items, unread...)
	body["lumiaItems"] = keys.Must(items)

	document, err := json.Marshal(body)
	if err != nil {
		return format.MainFile{}, fmt.Errorf("write the Lumiverse pack: %w", err)
	}
	return format.MainFile{Body: document, MediaType: "application/json", Extension: ".json"}, nil
}

func writeModeledUnlessUnread[T comparable](
	target, unread map[string]json.RawMessage,
	key string,
	value, fallback T,
) {
	if _, held := unread[key]; held && value == fallback {
		return
	}
	target[key] = keys.Must(value)
}

func preserved(rows []format.Remainder) (
	map[string]json.RawMessage,
	map[uuid.UUID]map[string]json.RawMessage,
) {
	body := make(map[string]json.RawMessage)
	items := make(map[uuid.UUID]map[string]json.RawMessage)
	for _, row := range rows {
		switch {
		case row.Owner == format.OwnerWork && row.Namespace == packNamespace:
			body = keys.Object(row.Payload)
		case row.Owner == format.OwnerItem && row.Namespace == itemNamespace:
			items[row.OwnerID] = keys.Object(row.Payload)
		}
	}
	return body, items
}

func Modules() []format.Reader { return []format.Reader{Module{}} }
