package download

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/readerkey"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

// Access says how a download was authorised, never who received the bytes
type Access string

const (
	AccessPublic Access = "public"
	AccessOwner  Access = "owner"
	AccessApp    Access = "app"
)

// Record is what Illarin keeps of one handed-over main file
type Record struct {
	WorkID         uuid.UUID
	OriginalFileID *uuid.UUID
	VersionNumber  *int
	Format         string
	Access         Access
}

func newRecord(
	workID uuid.UUID,
	originalFileID *uuid.UUID,
	versionNumber *int,
	formatID string,
	ownerID *uuid.UUID,
	viewerID *uuid.UUID,
) Record {
	access := AccessPublic
	if viewerID != nil && ownerID != nil && *viewerID == *ownerID {
		access = AccessOwner
	}
	return Record{
		WorkID: workID, OriginalFileID: originalFileID, VersionNumber: versionNumber, Format: formatID,
		Access: access,
	}
}

// Record keeps every handoff and counts a public one once per reader, work and UTC day
func (s *Service) Record(ctx context.Context, record Record, reader readerkey.Reader, now time.Time) error {
	err := s.record(ctx, record, reader, now)
	if readerkey.SecretGone(err) {
		err = s.record(ctx, record, reader, now)
	}
	return err
}

func (s *Service) record(ctx context.Context, record Record, reader readerkey.Reader, now time.Time) error {
	var day time.Time
	var key []byte
	if record.Access == AccessPublic {
		var err error
		if day, key, err = readerkey.Today(ctx, s.pool, now, reader, record.WorkID); err != nil {
			return err
		}
	}
	var recorded int
	err := s.pool.QueryRow(ctx, `
		with recorded as (
			insert into download_records
				(work_id, original_file_id, format, access, visibility, version_number)
			select work.id, $2, $3, $4, work.visibility, $5
			  from works work
			 where work.id = $1
			   and work.lifecycle = 'published'
			   and work.deleted_at is null
			   and (work.taken_down_at is null or $4 = 'owner')
			returning access
		), seen as (
			insert into reader_keys (day, kind, key)
			select $6, 'download', $7 from recorded where access = 'public'
			on conflict do nothing returning day
		), counted as (
			insert into events (kind, work_id, day) select 'download', $1, day from seen
		)
		select count(*) from recorded
	`, record.WorkID, record.OriginalFileID, record.Format, record.Access, record.VersionNumber, day, key).Scan(&recorded)
	if err != nil {
		return fmt.Errorf("record download: %w", err)
	}
	if recorded != 1 {
		return work.ErrNotFound
	}
	return nil
}
