package asset

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrReplacementDecision = errors.New("a replacement decision is required")

type stagedReplacement struct {
	Prepared preparedIngest
	Preview  ReplacementPreview
}

func (s *Service) stageReplacement(ctx context.Context, job ingestJob, prepared preparedIngest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replacement preview: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := lockBlobDigest(ctx, tx, job.BlobID); err != nil {
		return err
	}
	if _, err := (&Candidate{Version: job.Target.Version}).Lock(ctx, tx, job.OwnerID, job.Target.AssetID); err != nil {
		return err
	}
	preview, err := s.replacementPreview(ctx, tx, job.Target.AssetID, prepared)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(stagedReplacement{Prepared: prepared, Preview: preview})
	if err != nil {
		return fmt.Errorf("encode replacement preview: %w", err)
	}
	if err := s.ensureAccountStorage(ctx, tx, job.OwnerID, replacementBlobIDs(job, prepared)); err != nil {
		return err
	}
	result, err := tx.Exec(ctx, `
		update ingest_operations
		   set status = 'preview', replacement_preview = $3,
		       lease_token = null, lease_expires_at = null, updated_at = $4
		 where id = $1 and lease_token = $2 and status = 'processing'
	`, job.ID, job.LeaseToken, stored, s.now())
	if err != nil {
		return fmt.Errorf("store replacement preview: %w", err)
	}
	if result.RowsAffected() == 0 {
		return errIngestLeaseLost
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit replacement preview: %w", err)
	}
	return nil
}

func replacementBlobIDs(job ingestJob, prepared preparedIngest) []uuid.UUID {
	ids := make([]uuid.UUID, 1, len(prepared.Media)+1)
	ids[0] = job.BlobID
	for _, media := range prepared.Media {
		ids = append(ids, media.BlobID)
	}
	return ids
}

func (s *Service) replacementPreview(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, prepared preparedIngest) (ReplacementPreview, error) {
	working, err := readBlocks(ctx, tx, assetID)
	if err != nil {
		return ReplacementPreview{}, err
	}
	public, err := readPublishedBlocks(ctx, tx, assetID)
	if err != nil {
		return ReplacementPreview{}, err
	}
	incoming := blocksWithSuppliedRoles(prepared.Blocks, prepared.SuppliedRoles)
	changes := compareReplacementRoles(working, public, incoming)
	currentRemainder, err := readRemainder(ctx, tx, "asset_preserved_data", assetID)
	if err != nil {
		return ReplacementPreview{}, err
	}
	publicRemainder, err := readRemainder(ctx, tx, "asset_public.asset_preserved_data", assetID)
	if err != nil {
		return ReplacementPreview{}, err
	}
	changes = append(changes, replacementOpaqueChanges(currentRemainder, publicRemainder, prepared.Remainder)...)
	currentImages, publicImages, err := replacementImages(ctx, tx, assetID)
	if err != nil {
		return ReplacementPreview{}, err
	}
	incomingImages := make([]uuid.UUID, len(prepared.Media))
	for index, media := range prepared.Media {
		incomingImages[index] = media.BlobID
	}
	changes = append(changes, replacementImageChanges(currentImages, publicImages, incomingImages)...)
	slices.SortFunc(changes, func(a, b ReplacementChange) int {
		if a.Subject == b.Subject {
			if a.Kind < b.Kind {
				return -1
			}
			if a.Kind > b.Kind {
				return 1
			}
			return 0
		}
		if a.Subject < b.Subject {
			return -1
		}
		return 1
	})
	current := roleContent(working)
	incomingRoles := roleContent(incoming)
	unsupported := make([]string, 0)
	if declaration, known := s.reg.Declaration(prepared.Format); known {
		for role := range current {
			support, declared := declaration.Roles[role]
			if _, supplied := incomingRoles[role]; supplied || (declared && support.Write.Grade != format.SupportNone) {
				continue
			}
			unsupported = append(unsupported, string(role))
		}
	}
	slices.Sort(unsupported)
	return ReplacementPreview{Format: prepared.Format, Changes: changes, Unrepresentable: unsupported}, nil
}

func replacementImages(ctx context.Context, tx pgx.Tx, assetID uuid.UUID) (current, public []uuid.UUID, err error) {
	for _, source := range []struct {
		table string
		into  *[]uuid.UUID
	}{
		{table: "asset_media", into: &current},
		{table: "asset_public.asset_media", into: &public},
	} {
		rows, queryErr := tx.Query(ctx, fmt.Sprintf(`
			select blob_id from %s
			 where asset_id = $1 and is_extracted and blob_id is not null%s
		`, source.table, map[bool]string{true: " and is_current", false: ""}[source.table == "asset_media"]), assetID)
		if queryErr != nil {
			return nil, nil, queryErr
		}
		for rows.Next() {
			var id uuid.UUID
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return nil, nil, err
			}
			*source.into = append(*source.into, id)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, nil, err
		}
		rows.Close()
	}
	return current, public, nil
}

func sameImageSet(first, second []uuid.UUID) bool {
	if len(first) != len(second) {
		return false
	}
	first = slices.Clone(first)
	second = slices.Clone(second)
	slices.SortFunc(first, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	slices.SortFunc(second, func(a, b uuid.UUID) int { return bytes.Compare(a[:], b[:]) })
	return slices.Equal(first, second)
}

func replacementImageChanges(current, public, incoming []uuid.UUID) []ReplacementChange {
	changes := make([]ReplacementChange, 0, 2)
	switch {
	case len(current) == 0 && len(incoming) > 0:
		changes = append(changes, ReplacementChange{Kind: "addition", Subject: "images"})
	case len(current) > 0 && len(incoming) == 0:
		changes = append(changes, ReplacementChange{Kind: "removal", Subject: "images"})
	case !sameImageSet(current, incoming):
		changes = append(changes, ReplacementChange{Kind: "change", Subject: "images"})
	}
	if !sameImageSet(public, current) && !sameImageSet(current, incoming) {
		changes = append(changes, ReplacementChange{Kind: "conflict", Subject: "images"})
	}
	return changes
}

func replacementOpaqueChanges(current, public, incoming []format.Remainder) []ReplacementChange {
	if reflect.DeepEqual(current, incoming) {
		return nil
	}
	changes := []ReplacementChange{{Kind: "change", Subject: "opaque_data"}}
	if !reflect.DeepEqual(public, current) {
		changes = append(changes, ReplacementChange{Kind: "conflict", Subject: "opaque_data"})
	}
	return changes
}

func compareReplacementRoles(working, public, incoming []block.Block) []ReplacementChange {
	current := roleContent(working)
	baseline := roleContent(public)
	replacement := roleContent(incoming)
	roles := make(map[block.Role]struct{}, len(current)+len(replacement))
	for role := range current {
		roles[role] = struct{}{}
	}
	for role := range replacement {
		roles[role] = struct{}{}
	}
	changes := make([]ReplacementChange, 0, len(roles))
	for role := range roles {
		currentValue, currentOK := current[role]
		replacementValue, replacementOK := replacement[role]
		subject := string(role)
		switch {
		case !currentOK && replacementOK:
			changes = append(changes, ReplacementChange{Kind: "addition", Subject: subject})
		case currentOK && !replacementOK:
			changes = append(changes, ReplacementChange{Kind: "removal", Subject: subject})
		case !bytes.Equal(currentValue.Content, replacementValue.Content):
			if len(currentValue.Items) > 0 || len(replacementValue.Items) > 0 {
				removed, added, shared := itemDifferences(currentValue.Items, replacementValue.Items)
				for range removed {
					changes = append(changes, ReplacementChange{Kind: "removal", Subject: subject})
				}
				for range added {
					changes = append(changes, ReplacementChange{Kind: "addition", Subject: subject})
				}
				if shared {
					changes = append(changes, ReplacementChange{Kind: "change", Subject: subject})
				}
			} else {
				changes = append(changes, ReplacementChange{Kind: "change", Subject: subject})
			}
		}
		if baselineValue, baselineOK := baseline[role]; baselineOK &&
			!sameRoleValue(baselineValue, true, currentValue, currentOK) &&
			!sameRoleValue(currentValue, currentOK, replacementValue, replacementOK) {
			changes = append(changes, ReplacementChange{Kind: "conflict", Subject: subject})
		}
	}
	return changes
}

func sameRoleValue(first roleValue, firstOK bool, second roleValue, secondOK bool) bool {
	return firstOK == secondOK && (!firstOK || bytes.Equal(first.Content, second.Content))
}

type roleValue struct {
	Content []byte
	Items   []uuid.UUID
}

func roleContent(blocks []block.Block) map[block.Role]roleValue {
	found := make(map[block.Role]roleValue)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			if element.Role == "" || element.Content == nil || element.Content.Empty() {
				continue
			}
			content, err := json.Marshal(element.Content)
			if err == nil {
				found[element.Role] = roleValue{Content: content, Items: block.ItemIDs(element.Content)}
			}
		}
	}
	return found
}

func blocksWithSuppliedRoles(blocks []block.Block, supplied []block.Role) []block.Block {
	known := make(map[block.Role]struct{}, len(supplied))
	for _, role := range supplied {
		known[role] = struct{}{}
	}
	filtered := make([]block.Block, len(blocks))
	for index, holder := range blocks {
		filtered[index] = holder
		filtered[index].Elements = make([]block.Element, 0, len(holder.Elements))
		for _, element := range holder.Elements {
			if element.Role == "" {
				filtered[index].Elements = append(filtered[index].Elements, element)
				continue
			}
			if _, ok := known[element.Role]; ok {
				filtered[index].Elements = append(filtered[index].Elements, element)
			}
		}
	}
	return filtered
}

func itemDifferences(first, second []uuid.UUID) (removed, added []uuid.UUID, shared bool) {
	known := make(map[uuid.UUID]struct{}, len(first))
	for _, id := range first {
		known[id] = struct{}{}
	}
	for _, id := range second {
		if _, ok := known[id]; ok {
			shared = true
			delete(known, id)
			continue
		}
		added = append(added, id)
	}
	for id := range known {
		removed = append(removed, id)
	}
	return removed, added, shared
}

func mergeReplacementBlocks(existing, incoming []block.Block, supplied []block.Role, decisions map[string]string) []block.Block {
	incoming = blocksWithSuppliedRoles(incoming, supplied)
	byDefinition := make(map[block.DefinitionID]block.Block, len(incoming))
	for _, holder := range incoming {
		byDefinition[holder.Definition] = holder
	}
	merged := make([]block.Block, 0, len(existing)+len(incoming))
	for _, holder := range existing {
		replacement, supplied := byDefinition[holder.Definition]
		if !supplied {
			if hasSemanticContent(holder) && !keepUnrepresentable(holder, decisions) {
				continue
			}
			merged = append(merged, holder)
			continue
		}
		delete(byDefinition, holder.Definition)
		replacement.ID = holder.ID
		replacement.Title = holder.Title
		replacement.Hidden = holder.Hidden
		replacement.Position = holder.Position
		replacement.Width = holder.Width
		if len(holder.Layout.Slots()) == len(replacement.Elements) {
			replacement.Layout = holder.Layout
			for index := range replacement.Elements {
				replacement.Elements[index].Slot = holder.Layout.Slots()[index]
			}
		}
		for index, element := range replacement.Elements {
			if decisions[string(element.Role)] != "keep" {
				continue
			}
			for _, prior := range holder.Elements {
				if prior.Role == element.Role {
					replacement.Elements[index] = prior
					break
				}
			}
		}
		for _, prior := range holder.Elements {
			if decisions[string(prior.Role)] != "keep" || containsRole(replacement.Elements, prior.Role) {
				continue
			}
			replacement.Elements = append(replacement.Elements, prior)
			replacement.Layout = holder.Layout
			for index := range replacement.Elements {
				replacement.Elements[index].Slot = holder.Layout.Slots()[index]
			}
		}
		merged = append(merged, replacement)
	}
	for _, holder := range incoming {
		if _, left := byDefinition[holder.Definition]; left {
			merged = append(merged, holder)
			delete(byDefinition, holder.Definition)
		}
	}
	for index := range merged {
		merged[index].Position = index
	}
	return merged
}

func containsRole(elements []block.Element, role block.Role) bool {
	for _, element := range elements {
		if element.Role == role {
			return true
		}
	}
	return false
}

func keepUnrepresentable(holder block.Block, decisions map[string]string) bool {
	for _, element := range holder.Elements {
		if decisions[string(element.Role)] == "keep" {
			return true
		}
	}
	return false
}

func retainUnrepresentableRemainder(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	blocks []block.Block,
	incoming []format.Remainder,
	decisions map[string]string,
) ([]format.Remainder, error) {
	kept := make(map[uuid.UUID]struct{})
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			if decisions[string(element.Role)] != "keep" {
				continue
			}
			kept[element.ID] = struct{}{}
			for _, itemID := range block.ItemIDs(element.Content) {
				kept[itemID] = struct{}{}
			}
		}
	}
	if len(kept) == 0 {
		return incoming, nil
	}
	present := make(map[string]struct{}, len(incoming))
	for _, item := range incoming {
		present[remainderKey(item.Owner, item.OwnerID, item.Namespace)] = struct{}{}
	}
	rows, err := tx.Query(ctx, `
		select owner_kind, owner_id, namespace, payload
		  from asset_preserved_data where asset_id = $1
		 order by owner_kind, owner_id, namespace
	`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	retained := slices.Clone(incoming)
	for rows.Next() {
		var item format.Remainder
		if err := rows.Scan(&item.Owner, &item.OwnerID, &item.Namespace, &item.Payload); err != nil {
			return nil, err
		}
		if _, wanted := kept[item.OwnerID]; !wanted {
			continue
		}
		key := remainderKey(item.Owner, item.OwnerID, item.Namespace)
		if _, replaced := present[key]; replaced {
			continue
		}
		retained = append(retained, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return retained, nil
}

func remainderKey(owner format.Owner, ownerID uuid.UUID, namespace string) string {
	return string(owner) + "\x00" + ownerID.String() + "\x00" + namespace
}

func hasSemanticContent(holder block.Block) bool {
	for _, element := range holder.Elements {
		if element.Role != "" {
			return true
		}
	}
	return false
}

func readRemainder(ctx context.Context, tx pgx.Tx, table string, assetID uuid.UUID) ([]format.Remainder, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		select owner_kind, owner_id, namespace, payload
		  from %s where asset_id = $1
		 order by owner_kind, owner_id, namespace
	`, table), assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	current := make([]format.Remainder, 0)
	for rows.Next() {
		var owner format.Owner
		var ownerID uuid.UUID
		var namespace string
		var payload []byte
		if err := rows.Scan(&owner, &ownerID, &namespace, &payload); err != nil {
			return nil, err
		}
		current = append(current, format.Remainder{
			Owner: owner, OwnerID: ownerID, Namespace: namespace, Payload: payload,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return current, nil
}

// AcceptReplacement applies one stored preview only while the reviewed working copy is current.
func (s *Service) AcceptReplacement(ctx context.Context, ownerID, assetID, operationID uuid.UUID, candidate *Candidate, decisions map[string]string) (IngestOperation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return IngestOperation{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := candidate.Lock(ctx, tx, ownerID, assetID); err != nil {
		return IngestOperation{}, err
	}
	var status IngestStatus
	var targetID, blobID pgtype.UUID
	var filename string
	var stored []byte
	err = tx.QueryRow(ctx, `
		select status, target_asset_id, blob_id, filename, replacement_preview
		  from ingest_operations
		 where id = $1 and owner_id = $2 for update
	`, operationID, ownerID).Scan(&status, &targetID, &blobID, &filename, &stored)
	if err != nil {
		return IngestOperation{}, fmt.Errorf("read replacement preview: %w", err)
	}
	if status != IngestPreview || !targetID.Valid || uuidFromPgtype(targetID) != assetID || !blobID.Valid {
		return IngestOperation{}, ErrIngestNotFound
	}
	var staged stagedReplacement
	if err := json.Unmarshal(stored, &staged); err != nil {
		return IngestOperation{}, fmt.Errorf("read replacement preview: %w", err)
	}
	prepared := staged.Prepared
	if err := replacementDecisions(staged.Preview.Unrepresentable, decisions); err != nil {
		return IngestOperation{}, err
	}
	job := ingestJob{
		ID: operationID, OwnerID: ownerID, BlobID: uuidFromPgtype(blobID), Filename: filename,
		Target: &revisionTarget{AssetID: assetID, Kind: prepared.Kind, Version: candidate.Version},
	}
	if _, err := s.writeIngestResultWithDecisions(ctx, tx, job, prepared, decisions); err != nil {
		return IngestOperation{}, err
	}
	if _, err := tx.Exec(ctx, `
		update ingest_operations
		   set status = 'success', asset_id = $2, blob_id = null, replacement_preview = null,
		       updated_at = $3
		 where id = $1
	`, operationID, assetID, s.now()); err != nil {
		return IngestOperation{}, fmt.Errorf("accept replacement preview: %w", err)
	}
	if err := candidate.commit(ctx, tx, assetID); err != nil {
		return IngestOperation{}, err
	}
	accepted, err := s.GetIngest(ctx, ownerID, operationID)
	if err != nil {
		return IngestOperation{}, err
	}
	return accepted, nil
}

func (s *Service) CancelReplacement(ctx context.Context, ownerID, assetID, operationID uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `
		update ingest_operations
		   set status = 'cancelled', blob_id = null, replacement_preview = null, updated_at = $4
		 where id = $1 and owner_id = $2 and target_asset_id = $3 and status = 'preview'
	`, operationID, ownerID, assetID, s.now())
	if err != nil {
		return fmt.Errorf("cancel replacement preview: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrIngestNotFound
	}
	return nil
}

func replacementDecisions(roles []string, decisions map[string]string) error {
	for _, role := range roles {
		decision := decisions[role]
		if decision != "keep" && decision != "remove" {
			return fmt.Errorf("%w: choose whether to keep or remove %s", ErrReplacementDecision, role)
		}
	}
	return nil
}
