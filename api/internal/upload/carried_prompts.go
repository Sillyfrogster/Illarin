package upload

import (
	"context"
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sourceIdentity struct {
	namespace string
	id        string
}

// stableItemNames names each item by the identity its origin format gave it
func stableItemNames(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	incoming []format.Remainder,
) (map[uuid.UUID]string, error) {
	held, err := readSourceIdentities(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(held)+len(incoming))
	for item, identity := range held {
		names[item] = identity.namespace + "\x00" + identity.id
	}
	for item, identity := range sourceIdentities(incoming) {
		names[item] = identity.namespace + "\x00" + identity.id
	}
	return names, nil
}

// carriedPromptText finds the prompt wording the work already holds for each incoming fragment
func carriedPromptText(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	existing []block.Block,
	incoming []format.Remainder,
) (map[uuid.UUID]string, error) {
	held, err := readSourceIdentities(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	existingByIdentity := invertSourceIdentities(held)
	text := promptTextByItem(existing)
	carried := make(map[uuid.UUID]string)
	for arriving, identity := range sourceIdentities(incoming) {
		settled, known := existingByIdentity[identity]
		if !known {
			continue
		}
		if held, found := text[settled]; found {
			carried[arriving] = held
		}
	}
	return carried, nil
}

func sourceIdentities(records []format.Remainder) map[uuid.UUID]sourceIdentity {
	found := make(map[uuid.UUID]sourceIdentity, len(records))
	for _, record := range records {
		if record.Owner != format.OwnerItem {
			continue
		}
		id, named := preservedSourceID(record.Payload)
		if !named {
			continue
		}
		found[record.OwnerID] = sourceIdentity{namespace: record.Namespace, id: id}
	}
	return found
}

func readSourceIdentities(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
) (map[uuid.UUID]sourceIdentity, error) {
	rows, err := tx.Query(ctx, `
		select owner_id, namespace, payload
		  from work_preserved_data
		 where work_id = $1 and owner_type = $2
	`, workID, string(format.OwnerItem))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	found := make(map[uuid.UUID]sourceIdentity)
	for rows.Next() {
		var ownerID uuid.UUID
		var namespace string
		var payload []byte
		if err := rows.Scan(&ownerID, &namespace, &payload); err != nil {
			return nil, err
		}
		id, named := preservedSourceID(payload)
		if !named {
			continue
		}
		found[ownerID] = sourceIdentity{namespace: namespace, id: id}
	}
	return found, rows.Err()
}

func preservedSourceID(payload []byte) (string, bool) {
	var fields struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(payload, &fields) != nil || fields.ID == "" {
		return "", false
	}
	return fields.ID, true
}

// invertSourceIdentities keys items by identity and drops any identity two items share
func invertSourceIdentities(held map[uuid.UUID]sourceIdentity) map[sourceIdentity]uuid.UUID {
	byIdentity := make(map[sourceIdentity]uuid.UUID, len(held))
	shared := make(map[sourceIdentity]bool, len(held))
	for item, identity := range held {
		if _, taken := byIdentity[identity]; taken {
			shared[identity] = true
			continue
		}
		byIdentity[identity] = item
	}
	for identity := range shared {
		delete(byIdentity, identity)
	}
	return byIdentity
}

func promptTextByItem(blocks []block.Block) map[uuid.UUID]string {
	text := make(map[uuid.UUID]string)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				if fragment.Text != "" {
					text[fragment.ID] = fragment.Text
				}
			}
		}
	}
	return text
}
