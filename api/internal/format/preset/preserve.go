package preset

import (
	"encoding/json"
	"maps"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

type itemLeftover struct {
	namespace string
	fields    map[string]json.RawMessage
}

type preservation struct {
	body       string
	extensions string
	reserved   []string
}

func (p preservation) remainder(
	source map[string]json.RawMessage,
	items ...map[uuid.UUID]itemLeftover,
) []format.Remainder {
	extensions := make(map[string]json.RawMessage)
	if raw, held := source[p.extensions]; held {
		if json.Unmarshal(raw, &extensions) == nil {
			delete(source, p.extensions)
		} else {
			extensions = make(map[string]json.RawMessage)
		}
	}
	clashes := make(map[string]json.RawMessage)
	for _, name := range p.reserved {
		if collision, clash := extensions[name]; clash {
			clashes[name] = collision
			delete(extensions, name)
		}
	}
	if len(clashes) > 0 {
		source[p.extensions], _ = json.Marshal(clashes)
	}

	rows := make([]format.Remainder, 0, len(extensions)+1)
	if len(source) > 0 {
		payload, _ := json.Marshal(source)
		rows = append(rows, format.Remainder{
			Owner: format.OwnerWork, Namespace: p.body, Payload: payload,
		})
	}
	for _, namespace := range slices.Sorted(maps.Keys(extensions)) {
		rows = append(rows, format.Remainder{
			Owner: format.OwnerWork, Namespace: namespace, Payload: extensions[namespace],
		})
	}
	for _, group := range items {
		for _, id := range slices.SortedFunc(maps.Keys(group), compareIDs) {
			payload, _ := json.Marshal(group[id].fields)
			rows = append(rows, format.Remainder{
				Owner: format.OwnerItem, OwnerID: id,
				Namespace: group[id].namespace, Payload: payload,
			})
		}
	}
	return rows
}

func (p preservation) restoreExtensions(body map[string]json.RawMessage, held kept) {
	extensions := keys.Object(body[p.extensions])
	for namespace, payload := range held.work {
		if namespace == p.body {
			continue
		}
		if _, written := extensions[namespace]; !written {
			extensions[namespace] = payload
		}
	}
	if len(extensions) > 0 {
		body[p.extensions] = keys.Must(extensions)
	}
}

func compareIDs(first, second uuid.UUID) int {
	return slices.Compare(first[:], second[:])
}

func scriptLeftovers(
	scripts []block.Script,
	namespace string,
	fields map[uuid.UUID]map[string]json.RawMessage,
) map[uuid.UUID]itemLeftover {
	leftovers := make(map[uuid.UUID]itemLeftover, len(fields))
	for _, script := range scripts {
		if held, kept := fields[script.ID]; kept {
			leftovers[script.ID] = itemLeftover{namespace: namespace, fields: held}
		}
	}
	return leftovers
}

type kept struct {
	work  map[string]json.RawMessage
	items map[string]map[uuid.UUID]map[string]json.RawMessage
}

func preservedBy(rows []format.Remainder) kept {
	held := kept{
		work:  make(map[string]json.RawMessage),
		items: make(map[string]map[uuid.UUID]map[string]json.RawMessage),
	}
	for _, row := range rows {
		switch row.Owner {
		case format.OwnerWork:
			held.work[row.Namespace] = row.Payload
		case format.OwnerItem:
			if held.items[row.Namespace] == nil {
				held.items[row.Namespace] = make(map[uuid.UUID]map[string]json.RawMessage)
			}
			held.items[row.Namespace][row.OwnerID] = keys.Object(row.Payload)
		}
	}
	return held
}

func (k kept) item(namespace string, id uuid.UUID) map[string]json.RawMessage {
	return k.items[namespace][id]
}

func (k kept) object(namespace string) map[string]json.RawMessage {
	return keys.Object(k.work[namespace])
}

func itemName(held kept, namespace string, id uuid.UUID, key string) string {
	var name string
	if json.Unmarshal(held.item(namespace, id)[key], &name) == nil && name != "" {
		return name
	}
	return id.String()
}
