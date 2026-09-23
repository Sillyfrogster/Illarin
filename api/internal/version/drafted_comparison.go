package version

import (
	"context"
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) CompareDraftedChanges(ctx context.Context, ownerID, workID uuid.UUID, candidate *work.Candidate) ([]ChangeGroup, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := candidate.Lock(ctx, tx, ownerID, workID); err != nil {
		return nil, err
	}
	later, err := readComparisonVersion(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `set local search_path = work_public, public`); err != nil {
		return nil, err
	}
	earlier, err := readComparisonVersion(ctx, tx, workID)
	if err != nil {
		return nil, err
	}
	groups := compareVersions(earlier, later)
	if !sameMedia(earlier.OriginalFileID, later.OriginalFileID) {
		groups = AddGroup(groups, "original_file", "Uploaded file", []Change{{Type: ChangeEdited, Name: "Original file"}})
	}
	for _, group := range groups {
		for i := range group.Changes {
			change := &group.Changes[i]
			if change.BeforeMedia != nil {
				change.BeforeImage = s.works.ImageAddress(*change.BeforeMedia, "thumb", false, true)
			}
			if change.AfterMedia != nil {
				change.AfterImage = s.works.ImageAddress(*change.AfterMedia, "thumb", false, true)
			}
		}
	}
	return groups, nil
}

func readComparisonVersion(ctx context.Context, tx pgx.Tx, workID uuid.UUID) (work.FullVersion, error) {
	var version work.FullVersion
	var metadata, preserved []byte
	err := tx.QueryRow(ctx, `select type, coalesce(original_format, ''), original_file_id, to_jsonb(w),
		coalesce((select jsonb_agg(jsonb_build_object('id', p.id, 'owner_type', p.owner_type,
		'owner_id', p.owner_id, 'namespace', p.namespace, 'payload', p.payload::text))
		from work_preserved_data p where p.work_id = w.id), '[]'::jsonb)
		from works w where id = $1`, workID).Scan(&version.Type, &version.OriginalFormat, &version.OriginalFileID, &metadata, &preserved)
	if err != nil {
		return version, err
	}
	if err := json.Unmarshal(metadata, &version.Metadata); err != nil {
		return version, err
	}
	if err := json.Unmarshal(preserved, &version.Preserved); err != nil {
		return version, err
	}
	version.Blocks, err = block.Read(ctx, tx, workID)
	if err != nil {
		return version, err
	}
	return version, private.RestorePromptFragments(ctx, tx, workID, version.Blocks)
}
