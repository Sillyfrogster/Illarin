// The content fingerprint says whether a change would alter the bytes a download or send hands out.
package work

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
	workID uuid.UUID,
) (string, error) {
	digest := sha256.New()

	var workType, name, blurb, workVersion, creditedAuthor, nickname string
	var origin pgtype.Text
	var cover pgtype.UUID
	err := q.QueryRow(ctx, `
		select type, original_format, name, blurb, work_version, credited_author,
		       nickname, cover_media_id
		  from works
		 where id = $1 and deleted_at is null
	`, workID).Scan(&workType, &origin, &name, &blurb, &workVersion, &creditedAuthor,
		&nickname, &cover)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("read the work to fingerprint: %w", err)
	}
	fmt.Fprintf(digest, "asset\x00%s\x00%s\n", workType, origin.String)
	if err := s.fingerprintUpload(ctx, q, workID, origin.String, digest); err != nil {
		return "", err
	}

	values := map[format.HeaderField]string{
		format.HeaderName:           name,
		format.HeaderBlurb:          blurb,
		format.HeaderWorkVersion:    workVersion,
		format.HeaderCreditedAuthor: creditedAuthor,
		format.HeaderNickname:       nickname,
	}
	for _, field := range s.reg.ExportedHeaderFields(workType) {
		fmt.Fprintf(digest, "header\x00%s\x00%s\n", field, values[field])
	}

	blocks, err := block.Read(ctx, q, workID)
	if err != nil {
		return "", err
	}
	if err := private.RestorePromptFragments(ctx, q, workID, blocks); err != nil {
		return "", err
	}
	elements := make([]block.Element, 0)
	for _, holder := range blocks {
		elements = append(elements, holder.Elements...)
	}
	names, err := fingerprintNames(ctx, q, workID, elements)
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
				prompts.Fragments[index].Private = false
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
		content, err = json.Marshal(SteadyIDs(value, names))
		if err != nil {
			return "", err
		}
		fmt.Fprintf(digest, "element\x00%s\x00%s\n", names[element.ID], content)
	}

	if err := fingerprintPreserved(ctx, q, workID, names, digest); err != nil {
		return "", err
	}
	if err := fingerprintPictures(ctx, q, workID, elements, uuidOrNil(cover), names, digest); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}

// fingerprintUpload counts the uploaded bytes as content where the export hands them back unchanged.
func (s *Service) fingerprintUpload(ctx context.Context, q db.DBTX, workID uuid.UUID, origin string, digest hash.Hash) error {
	declaration, known := s.reg.Declaration(origin)
	if !known || !declaration.KeepsUpload {
		return nil
	}
	var sum []byte
	err := q.QueryRow(ctx, `
		select blob.sha256
		  from works work
		  join work_original_files original on original.id = work.original_file_id
		  join blobs blob on blob.id = original.blob_id
		 where work.id = $1
	`, workID).Scan(&sum)
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
func fingerprintNames(ctx context.Context, q db.DBTX, workID uuid.UUID, elements []block.Element) (map[uuid.UUID]string, error) {
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
	rows, err := q.Query(ctx, `select id, blob_id from work_media where work_id = $1`, workID)
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
	workID uuid.UUID,
	names map[uuid.UUID]string,
	digest hash.Hash,
) error {
	rows, err := q.Query(ctx, `
		select owner_type, owner_id, namespace, payload::text
		  from work_preserved_data
		 where work_id = $1
		 order by owner_type, owner_id, namespace
	`, workID)
	if err != nil {
		return fmt.Errorf("read preserved data to fingerprint: %w", err)
	}
	defer rows.Close()
	entries := []string{}
	for rows.Next() {
		var ownerType, namespace, payload string
		var ownerID uuid.UUID
		if err := rows.Scan(&ownerType, &ownerID, &namespace, &payload); err != nil {
			return fmt.Errorf("read a preserved row to fingerprint: %w", err)
		}
		var value any
		decoder := json.NewDecoder(strings.NewReader(payload))
		decoder.UseNumber()
		if err := decoder.Decode(&value); err != nil {
			return err
		}
		canonical, err := json.Marshal(SteadyIDs(value, names))
		if err != nil {
			return err
		}
		owner := ownerID.String()
		if stable, ok := names[ownerID]; ok {
			owner = stable
		}
		entries = append(entries, fmt.Sprintf("preserved\x00%s\x00%s\x00%s\x00%s\n",
			ownerType, owner, namespace, canonical))
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
	workID uuid.UUID,
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
		  from work_media
		 where work_id = $1 and id = any($2)
		 order by id
	`, workID, wanted)
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

// SteadyIDs swaps ids a file mints afresh for the stable names they stand for
func SteadyIDs(value any, names map[uuid.UUID]string) any {
	if len(names) == 0 {
		return value
	}
	switch held := value.(type) {
	case map[string]any:
		for key, nested := range held {
			held[key] = SteadyIDs(nested, names)
		}
		return held
	case []any:
		for index, nested := range held {
			held[index] = SteadyIDs(nested, names)
		}
		return held
	case string:
		id, err := uuid.Parse(held)
		if err != nil {
			return held
		}
		if stable, named := names[id]; named {
			return stable
		}
		return held
	default:
		return value
	}
}
