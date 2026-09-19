// A full version is one published version of a work with its recorded content, read back whole.
package work

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrNoEarlierVersion = errors.New("nothing was recorded before that version")

type Version struct {
	ID                    uuid.UUID
	Number                int
	RecordedAt            time.Time
	Initial               bool
	VersionLabel          string
	Summary               string
	Notes                 string
	NotesEditedAt         *time.Time
	WithdrawnAt           *time.Time
	WithdrawalExplanation string
}

func RedactWithdrawn(version *Version) {
	if version.WithdrawnAt == nil {
		return
	}
	version.ID = uuid.Nil
	version.Initial = false
	version.VersionLabel = ""
	version.Summary = ""
	version.Notes = ""
	version.NotesEditedAt = nil
	version.WithdrawnAt = nil
}

func (v FullVersion) HoldPrompts(
	ctx context.Context,
	tx pgx.Tx,
	workID uuid.UUID,
	asOwner bool,
) (bool, error) {
	if err := private.RestoreRecordedPrompts(v.PrivatePrompts, v.Blocks); err != nil {
		return false, err
	}
	if asOwner {
		return false, nil
	}
	return private.ApplyRecordedPolicy(ctx, tx, workID, &v.ID, v.Blocks)
}

type FullVersion struct {
	Version
	Type           string
	Origin         string
	OriginalFileID *uuid.UUID
	Metadata       VersionMetadata
	Blocks         []block.Block
	Preserved      []VersionPreserved
	PrivatePrompts []byte
}

type VersionMetadata struct {
	Name           string     `json:"name"`
	Blurb          string     `json:"blurb"`
	Tags           []string   `json:"tags"`
	IsNSFW         *bool      `json:"is_nsfw"`
	CreditedAuthor string     `json:"credited_author"`
	Nickname       string     `json:"nickname"`
	WorkVersion    string     `json:"work_version"`
	Cover          *uuid.UUID `json:"cover_media_id"`
}

type VersionPreserved struct {
	ID        uuid.UUID `json:"id"`
	Owner     string    `json:"owner_type"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Namespace string    `json:"namespace"`
	Payload   string    `json:"payload"`
}

type versionPayload struct {
	VersionMetadata
	Type      string             `json:"type"`
	Origin    string             `json:"origin_format"`
	Blocks    []block.Block      `json:"blocks"`
	Preserved []VersionPreserved `json:"preserved_data"`
}

func ReadVersion(ctx context.Context, tx pgx.Tx, workID uuid.UUID, number int) (FullVersion, error) {
	var recorded FullVersion
	var stored []byte
	var sourceRevision pgtype.UUID
	err := tx.QueryRow(ctx, `
		select id, number, recorded_at, initial_recorded, version_label, summary, notes,
		       notes_edited_at, withdrawn_at, coalesce(withdrawal_explanation, ''),
		       original_file_id, payload, private_prompts
		  from public.work_versions where work_id = $1 and number = $2
	`, workID, number).Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
		&recorded.Initial, &recorded.VersionLabel, &recorded.Summary, &recorded.Notes,
		&recorded.NotesEditedAt, &recorded.WithdrawnAt, &recorded.WithdrawalExplanation,
		&sourceRevision, &stored, &recorded.PrivatePrompts)
	if errors.Is(err, pgx.ErrNoRows) {
		return FullVersion{}, ErrNotFound
	}
	if err != nil {
		return FullVersion{}, fmt.Errorf("read version %d: %w", number, err)
	}
	var payload versionPayload
	if err := json.Unmarshal(stored, &payload); err != nil {
		return FullVersion{}, fmt.Errorf("read version %d: %w", number, err)
	}
	recorded.Type = payload.Type
	recorded.Origin = payload.Origin
	recorded.OriginalFileID = uuidOrNil(sourceRevision)
	recorded.Metadata = payload.VersionMetadata
	recorded.Blocks = payload.Blocks
	recorded.Preserved = payload.Preserved
	return recorded, nil
}
