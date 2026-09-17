package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"slices"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) contentFingerprint(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
) (string, error) {
	digest := sha256.New()

	var kind, name, blurb, assetVersion, creditedAuthor, nickname string
	var origin pgtype.Text
	var cover pgtype.UUID
	err := q.QueryRow(ctx, `
		select kind, origin_format, name, blurb, asset_version, credited_author,
		       nickname, cover_media_id
		  from assets
		 where id = $1 and deleted_at is null
	`, assetID).Scan(&kind, &origin, &name, &blurb, &assetVersion, &creditedAuthor,
		&nickname, &cover)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read the asset to fingerprint: %w", err)
	}
	fmt.Fprintf(digest, "asset\x00%s\x00%s\n", kind, origin.String)
	if err := s.fingerprintUpload(ctx, q, assetID, origin.String, digest); err != nil {
		return "", err
	}

	values := map[format.HeaderField]string{
		format.HeaderName:           name,
		format.HeaderBlurb:          blurb,
		format.HeaderAssetVersion:   assetVersion,
		format.HeaderCreditedAuthor: creditedAuthor,
		format.HeaderNickname:       nickname,
	}
	for _, field := range s.reg.ExportedHeaderFields(kind) {
		fmt.Fprintf(digest, "header\x00%s\x00%s\n", field, values[field])
	}

	blocks, err := block.Read(ctx, q, assetID)
	if err != nil {
		return "", err
	}
	if err := private.RestorePromptFragments(ctx, q, assetID, blocks); err != nil {
		return "", err
	}
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		elements = append(elements, holder.Elements...)
	}
	names, err := fingerprintNames(ctx, q, assetID, elements)
	if err != nil {
		return "", err
	}
	slices.SortFunc(elements, func(a, b block.Element) int {
		return strings.Compare(names[a.ID], names[b.ID])
	})
	for _, element := range elements {
		if element.Content == nil || element.Content.Empty() {
			continue
		}
		if prompts, ok := element.Content.(block.PromptList); ok {
			prompts.Fragments = append([]block.PromptFragment(nil), prompts.Fragments...)
			for index := range prompts.Fragments {
				prompts.Fragments[index].Protected = false
			}
			element.Content = prompts
		}
		content, err := element.ContentJSON()
		if err != nil {
			return "", fmt.Errorf("fingerprint the %s element: %w", element.Role, err)
		}
		var value any
		decoder := json.NewDecoder(strings.NewReader(string(content)))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return "", err
		}
		content, err = json.Marshal(steadyIDs(value, names))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(digest, "element\x00%s\x00%s\n", names[element.ID], content)
	}

	if err := fingerprintPreserved(ctx, q, assetID, names, digest); err != nil {
		return "", err
	}
	if err := fingerprintPictures(ctx, q, assetID, elements, uuidOrNil(cover), names, digest); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// fingerprintUpload counts the uploaded bytes as content where the export hands them back unchanged.
func (s *Service) fingerprintUpload(ctx context.Context, q db.DBTX, assetID uuid.UUID, origin string, digest hash.Hash) error {
	declaration, known := s.reg.Declaration(origin)
	if !known || !declaration.KeepsUpload {
		return nil
	}
	var sum []byte
	err := q.QueryRow(ctx, `
		select blob.sha256
		  from assets asset
		  join asset_revisions revision on revision.id = asset.current_revision_id
		  join blobs blob on blob.id = revision.blob_id
		 where asset.id = $1
	`, assetID).Scan(&sum)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the upload to fingerprint: %w", err)
	}
	fmt.Fprintf(digest, "upload\x00%x\n", sum)
	return nil
}

// fingerprintNames gives imported elements, items and pictures stable comparison keys.
func fingerprintNames(ctx context.Context, q db.DBTX, assetID uuid.UUID, elements []block.Element) (map[uuid.UUID]string, error) {
	names := make(map[uuid.UUID]string)
	counts := make(map[string]int)
	for _, element := range elements {
		key := string(element.Type) + "/" + string(element.Role)
		counts[key]++
		key += "/" + strconv.Itoa(counts[key])
		names[element.ID] = key
		for index, id := range block.ItemIDs(element.Content) {
			names[id] = key + "/" + strconv.Itoa(index)
		}
	}
	rows, err := q.Query(ctx, `select id, blob_id from asset_media where asset_id = $1`, assetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, blob uuid.UUID
		if err := rows.Scan(&id, &blob); err != nil {
			return nil, err
		}
		names[id] = "picture/" + blob.String()
	}
	return names, rows.Err()
}

func fingerprintPreserved(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
	names map[uuid.UUID]string,
	digest hash.Hash,
) error {
	rows, err := q.Query(ctx, `
		select owner_kind, owner_id, namespace, payload::text
		  from asset_preserved_data
		 where asset_id = $1
		 order by owner_kind, owner_id, namespace
	`, assetID)
	if err != nil {
		return fmt.Errorf("read preserved data to fingerprint: %w", err)
	}
	defer rows.Close()
	entries := []string{}
	for rows.Next() {
		var ownerKind, namespace, payload string
		var ownerID uuid.UUID
		if err := rows.Scan(&ownerKind, &ownerID, &namespace, &payload); err != nil {
			return fmt.Errorf("read a preserved row to fingerprint: %w", err)
		}
		var value any
		decoder := json.NewDecoder(strings.NewReader(payload))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		canonical, err := json.Marshal(steadyIDs(value, names))
		if err != nil {
			return err
		}
		owner := ownerID.String()
		if stable, ok := names[ownerID]; ok {
			owner = stable
		}
		entries = append(entries, fmt.Sprintf("preserved\x00%s\x00%s\x00%s\x00%s\n",
			ownerKind, owner, namespace, canonical))
	}
	slices.Sort(entries)
	for _, entry := range entries {
		fmt.Fprint(digest, entry)
	}
	return rows.Err()
}

func fingerprintPictures(
	ctx context.Context,
	q db.DBTX,
	assetID uuid.UUID,
	elements []block.Element,
	cover *uuid.UUID,
	names map[uuid.UUID]string,
	digest hash.Hash,
) error {
	wanted := make([]uuid.UUID, 0)
	if cover != nil {
		wanted = append(wanted, *cover)
	}
	for _, element := range elements {
		set, isSet := element.Content.(block.ImageSet)
		if !isSet {
			continue
		}
		for _, image := range set.Images {
			wanted = append(wanted, image.MediaID)
		}
	}
	if len(wanted) == 0 {
		return nil
	}
	rows, err := q.Query(ctx, `
		select id, blob_id, is_current
		  from asset_media
		 where asset_id = $1 and id = any($2)
		 order by id
	`, assetID, wanted)
	if err != nil {
		return fmt.Errorf("read pictures to fingerprint: %w", err)
	}
	defer rows.Close()
	entries := []string{}
	for rows.Next() {
		var mediaID, blobID uuid.UUID
		var current bool
		if err := rows.Scan(&mediaID, &blobID, &current); err != nil {
			return fmt.Errorf("read a picture to fingerprint: %w", err)
		}
		entries = append(entries, fmt.Sprintf("picture\x00%s\x00%s\x00%t\n", names[mediaID], blobID, current))
	}
	slices.Sort(entries)
	for _, entry := range entries {
		fmt.Fprint(digest, entry)
	}
	return rows.Err()
}

func (s *Service) moveContentGeneration(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	before string,
) error {
	after, err := s.contentFingerprint(ctx, tx, assetID)
	if err != nil {
		return err
	}
	if after == before {
		return nil
	}
	if _, err := tx.Exec(ctx, `
		update assets set content_generation = content_generation + 1
		 where id = $1 and published_snapshot_id is null
	`, assetID); err != nil {
		return fmt.Errorf("move the content generation: %w", err)
	}
	return nil
}

// ChangeContent runs a change to a work's drafted content and moves its content generation when the content changed
func (s *Service) ChangeContent(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, change func() error) error {
	fingerprint, err := s.contentFingerprint(ctx, tx, assetID)
	if err != nil {
		return err
	}
	if err := change(); err != nil {
		return err
	}
	return s.moveContentGeneration(ctx, tx, assetID, fingerprint)
}
