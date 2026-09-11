package asset

import (
	"context"
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// sourceIdentity is the name an origin format gave an item, kept in preserved data.
type sourceIdentity struct {
	namespace string
	id        string
}

// stableItemNames names each item by the identity its origin format gave it.
func stableItemNames(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	incoming []format.Remainder,
) (map[uuid.UUID]string, error) {
	held, err := readSourceIdentities(ctx, tx, assetID)
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

// carriedPromptText finds the prompt text this asset already holds for each incoming fragment.
func carriedPromptText(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	existing []block.Block,
	incoming []format.Remainder,
) (map[uuid.UUID]string, error) {
	held, err := readSourceIdentities(ctx, tx, assetID)
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

// sourceIdentities reads the identity each preserved record names for its item.
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

// readSourceIdentities reads the same identities from what the asset already holds.
func readSourceIdentities(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
) (map[uuid.UUID]sourceIdentity, error) {
	rows, err := tx.Query(ctx, `
		select owner_id, namespace, payload
		  from asset_preserved_data
		 where asset_id = $1 and owner_kind = $2
	`, assetID, string(format.OwnerItem))
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

// preservedSourceID reads the identifier a file gave an item, when it gave one.
func preservedSourceID(payload []byte) (string, bool) {
	var fields struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(payload, &fields) != nil || fields.ID == "" {
		return "", false
	}
	return fields.ID, true
}

// invertSourceIdentities keys items by identity, dropping any identity two items share.
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

// promptTextByItem reads the wording each prompt fragment carries.
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
