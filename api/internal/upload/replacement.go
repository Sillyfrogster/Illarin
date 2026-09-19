package upload

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
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrReplacementDecision = errors.New("a replacement decision is required")

type stagedReplacement struct {
	Prepared preparedIngest
	Preview  Preview
}

func (s *Service) stageReplacement(ctx context.Context, job ingestJob, prepared preparedIngest) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin replacement preview: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := storage.LockBlobDigest(ctx, tx, job.BlobID); err != nil {
		return err
	}
	if _, err := (&work.Candidate{Version: job.Target.Version}).Lock(ctx, tx, job.OwnerID, job.Target.WorkID); err != nil {
		return err
	}
	preview, err := s.replacementPreview(ctx, tx, job.Target.WorkID, prepared)
	if err != nil {
		return err
	}
	stored, err := json.Marshal(stagedReplacement{Prepared: prepared, Preview: preview})
	if err != nil {
		return fmt.Errorf("encode replacement preview: %w", err)
	}
	if err := s.works.EnsureAccountStorage(ctx, tx, job.OwnerID, replacementBlobIDs(job, prepared)); err != nil {
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

func (s *Service) replacementPreview(ctx context.Context, tx pgx.Tx, workID uuid.UUID, prepared preparedIngest) (Preview, error) {
	working, err := block.Read(ctx, tx, workID)
	if err != nil {
		return Preview{}, err
	}
	if err := private.RestorePromptFragments(ctx, tx, workID, working); err != nil {
		return Preview{}, err
	}
	public, err := work.ReadPublishedBlocks(ctx, tx, workID)
	if err != nil {
		return Preview{}, err
	}
	incoming := blocksWithSuppliedRoles(prepared.Blocks, prepared.SuppliedRoles)
	carried, err := carriedPromptText(ctx, tx, workID, working, prepared.Remainder)
	if err != nil {
		return Preview{}, err
	}
	unfillable, err := private.UnfillablePrompts(ctx, tx, workID, carried, prepared.Protected)
	if err != nil {
		return Preview{}, err
	}
	missingWording := promptNames(incoming, unfillable)
	keepMissingPromptsEmpty(prepared.Protected, unfillable)
	arriving := mergeReplacementBlocks(working, prepared.Blocks, prepared.SuppliedRoles, nil)
	fillSealedPrompts(arriving, carried, prepared.Protected)

	currentRemainder, err := readRemainder(ctx, tx, "work_preserved_data", workID)
	if err != nil {
		return Preview{}, err
	}
	publicRemainder, err := readRemainder(ctx, tx, "work_public.work_preserved_data", workID)
	if err != nil {
		return Preview{}, err
	}
	currentImages, publicImages, err := replacementImages(ctx, tx, workID)
	if err != nil {
		return Preview{}, err
	}
	incomingImages := make([]uuid.UUID, len(prepared.Media))
	for index, media := range prepared.Media {
		incomingImages[index] = media.BlobID
	}

	names, err := stableItemNames(ctx, tx, workID, prepared.Remainder)
	if err != nil {
		return Preview{}, err
	}
	groups := version.CompareContentKeyed(working, arriving, names)
	groups = version.AddGroup(groups, version.PresentationSubject, "Page",
		version.ComparePresentation(prepared.Type, working, arriving))
	groups = version.AddGroup(groups, version.PreservedSubject, "Preserved data",
		version.ComparePreserved(asVersionPreserved(currentRemainder), asVersionPreserved(prepared.Remainder)))
	groups = version.AddGroup(groups, version.PicturesSubject, "Pictures",
		comparePictureSets(currentImages, incomingImages))
	if err := version.AddressPictures(ctx, tx, s.works, version.ComparisonRequest{
		WorkID: workID, NSFWPreference: work.NSFWShown,
	}, groups); err != nil {
		return Preview{}, err
	}

	conflicts := replacementConflicts(working, public, arriving,
		currentRemainder, publicRemainder, prepared.Remainder,
		currentImages, publicImages, incomingImages)

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
	return Preview{
		Format: prepared.Format, Groups: groups, Conflicts: conflicts,
		Unrepresentable: unsupported, MissingWording: missingWording,
		Seals: len(prepared.Protected),
	}, nil
}

func promptNames(incoming []block.Block, fragments []uuid.UUID) []string {
	wanted := make(map[uuid.UUID]bool, len(fragments))
	for _, id := range fragments {
		wanted[id] = true
	}
	names := make([]string, 0, len(fragments))
	for _, holder := range incoming {
		for _, element := range holder.Elements {
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			for _, fragment := range list.Fragments {
				if wanted[fragment.ID] {
					names = append(names, private.PromptName(fragment))
				}
			}
		}
	}
	slices.Sort(names)
	return names
}

func keepMissingPromptsEmpty(imports []format.ProtectedPrompt, fragments []uuid.UUID) {
	missing := make(map[uuid.UUID]bool, len(fragments))
	for _, id := range fragments {
		missing[id] = true
	}
	for index := range imports {
		if missing[imports[index].FragmentID] {
			imports[index].ReuseExisting = false
			imports[index].Text = ""
		}
	}
}

func fillSealedPrompts(blocks []block.Block, carried map[uuid.UUID]string, imports []format.ProtectedPrompt) {
	if len(imports) == 0 {
		return
	}
	text := make(map[uuid.UUID]string, len(imports))
	for _, imported := range imports {
		if imported.ReuseExisting {
			text[imported.FragmentID] = carried[imported.FragmentID]
			continue
		}
		text[imported.FragmentID] = imported.Text
	}
	for blockIndex := range blocks {
		for elementIndex := range blocks[blockIndex].Elements {
			element := &blocks[blockIndex].Elements[elementIndex]
			list, ok := element.Content.(block.PromptList)
			if !ok {
				continue
			}
			fragments := slices.Clone(list.Fragments)
			for itemIndex := range fragments {
				if held, sealed := text[fragments[itemIndex].ID]; sealed {
					fragments[itemIndex].Text = held
				}
			}
			list.Fragments = fragments
			element.Content = list
		}
	}
}

func asVersionPreserved(records []format.Remainder) []work.VersionPreserved {
	held := make([]work.VersionPreserved, len(records))
	for index, record := range records {
		held[index] = work.VersionPreserved{
			Owner: string(record.Owner), OwnerID: record.OwnerID,
			Namespace: record.Namespace, Payload: string(record.Payload),
		}
	}
	return held
}

func comparePictureSets(current, incoming []uuid.UUID) []version.Change {
	held := make(map[uuid.UUID]bool, len(current))
	for _, id := range current {
		held[id] = true
	}
	arriving := make(map[uuid.UUID]bool, len(incoming))
	for _, id := range incoming {
		arriving[id] = true
	}
	changes := make([]version.Change, 0)
	for _, id := range incoming {
		if !held[id] {
			changes = append(changes, version.Change{Type: version.ChangeAdded, Name: "Picture", AfterMedia: &id})
		}
	}
	for _, id := range current {
		if !arriving[id] {
			changes = append(changes, version.Change{Type: version.ChangeRemoved, Name: "Picture", BeforeMedia: &id})
		}
	}
	return changes
}

// replacementConflicts names the subjects where the file would overwrite an unpublished edit
func replacementConflicts(
	working, public, arriving []block.Block,
	currentRemainder, publicRemainder, incomingRemainder []format.Remainder,
	currentImages, publicImages, incomingImages []uuid.UUID,
) []string {
	current := roleContent(working)
	baseline := roleContent(public)
	replacement := roleContent(arriving)
	roles := make(map[block.Role]struct{}, len(current)+len(replacement))
	for role := range current {
		roles[role] = struct{}{}
	}
	for role := range replacement {
		roles[role] = struct{}{}
	}
	conflicts := make([]string, 0)
	for role := range roles {
		baselineValue, published := baseline[role]
		if !published {
			continue
		}
		currentValue, edited := current[role]
		replacementValue, replaced := replacement[role]
		if sameRoleValue(baselineValue, true, currentValue, edited) ||
			sameRoleValue(currentValue, edited, replacementValue, replaced) {
			continue
		}
		conflicts = append(conflicts, string(role))
	}
	if !reflect.DeepEqual(currentRemainder, incomingRemainder) &&
		!reflect.DeepEqual(publicRemainder, currentRemainder) {
		conflicts = append(conflicts, version.PreservedSubject)
	}
	if !sameImageSet(publicImages, currentImages) && !sameImageSet(currentImages, incomingImages) {
		conflicts = append(conflicts, version.PicturesSubject)
	}
	slices.Sort(conflicts)
	return conflicts
}

func replacementImages(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (current, public []uuid.UUID, err error) {
	for _, source := range []struct {
		table string
		into  *[]uuid.UUID
	}{
		{table: "work_media", into: &current},
		{table: "work_public.work_media", into: &public},
	} {
		rows, queryErr := tx.Query(ctx, fmt.Sprintf(`
			select blob_id from %s
			 where work_id = $1 and is_extracted and blob_id is not null%s
		`, source.table, map[bool]string{true: " and is_current", false: ""}[source.table == "work_media"]), workID)
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
	workID uuid.UUID,
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
		select owner_type, owner_id, namespace, payload
		  from work_preserved_data where work_id = $1
		 order by owner_type, owner_id, namespace
	`, workID)
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

func readRemainder(ctx context.Context, tx pgx.Tx, table string, workID uuid.UUID) ([]format.Remainder, error) {
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		select owner_type, owner_id, namespace, payload
		  from %s where work_id = $1
		 order by owner_type, owner_id, namespace
	`, table), workID)
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

func (s *Service) AcceptReplacement(ctx context.Context, ownerID, workID, operationID uuid.UUID, candidate *work.Candidate, decisions map[string]string, exposeProtected bool) (Operation, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Operation{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return Operation{}, err
	}
	var status Status
	var targetID, blobID pgtype.UUID
	var filename string
	var stored []byte
	err = tx.QueryRow(ctx, `
		select status, target_work_id, blob_id, filename, replacement_preview
		  from ingest_operations
		 where id = $1 and owner_id = $2 for update
	`, operationID, ownerID).Scan(&status, &targetID, &blobID, &filename, &stored)
	if err != nil {
		return Operation{}, fmt.Errorf("read replacement preview: %w", err)
	}
	if status != IngestPreview || !targetID.Valid || uuidFromPgtype(targetID) != workID || !blobID.Valid {
		return Operation{}, ErrIngestNotFound
	}
	var staged stagedReplacement
	if err := json.Unmarshal(stored, &staged); err != nil {
		return Operation{}, fmt.Errorf("read replacement preview: %w", err)
	}
	prepared := staged.Prepared
	if err := replacementDecisions(staged.Preview.Unrepresentable, decisions); err != nil {
		return Operation{}, err
	}
	job := ingestJob{
		ID: operationID, OwnerID: ownerID, BlobID: uuidFromPgtype(blobID), Filename: filename,
		Target: &originalFileTarget{WorkID: workID, Type: prepared.Type, Version: candidate.Version},
	}
	if _, err := s.writeIngestResultWithDecisions(ctx, tx, job, prepared, decisions, exposeProtected); err != nil {
		return Operation{}, err
	}
	if _, err := tx.Exec(ctx, `
		update ingest_operations
		   set status = 'success', work_id = $2, blob_id = null, replacement_preview = null,
		       updated_at = $3
		 where id = $1
	`, operationID, workID, s.now()); err != nil {
		return Operation{}, fmt.Errorf("accept replacement preview: %w", err)
	}
	if err := candidate.Commit(ctx, tx, workID); err != nil {
		return Operation{}, err
	}
	accepted, err := s.GetIngest(ctx, ownerID, operationID)
	if err != nil {
		return Operation{}, err
	}
	return accepted, nil
}

func (s *Service) ReviewedReplacement(ctx context.Context, ownerID, workID uuid.UUID) (Operation, error) {
	var operationID uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select id from ingest_operations
		 where target_work_id = $1 and owner_id = $2
		   and status in ('pending', 'processing', 'preview')
		 order by created_at desc limit 1
	`, workID, ownerID).Scan(&operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, ErrIngestNotFound
	}
	if err != nil {
		return Operation{}, fmt.Errorf("read the replacement waiting on this work: %w", err)
	}
	return s.GetIngest(ctx, ownerID, operationID)
}

func (s *Service) CancelReplacement(ctx context.Context, ownerID, workID, operationID uuid.UUID) error {
	result, err := s.pool.Exec(ctx, `
		update ingest_operations
		   set status = 'cancelled', blob_id = null, replacement_preview = null, updated_at = $4
		 where id = $1 and owner_id = $2 and target_work_id = $3 and status = 'preview'
	`, operationID, ownerID, workID, s.now())
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
