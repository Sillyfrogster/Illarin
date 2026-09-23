package download

import (
	"context"
	"fmt"

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
	Format         string
	Access         Access
}

func newRecord(
	workID uuid.UUID,
	originalFileID *uuid.UUID,
	formatID string,
	ownerID *uuid.UUID,
	viewerID *uuid.UUID,
) Record {
	access := AccessPublic
	if viewerID != nil && ownerID != nil && *viewerID == *ownerID {
		access = AccessOwner
	}
	return Record{
		WorkID: workID, OriginalFileID: originalFileID, Format: formatID,
		Access: access,
	}
}

func (s *Service) Record(ctx context.Context, record Record) error {
	recorded, err := s.pool.Exec(ctx, `
		insert into download_records
			(work_id, original_file_id, format, access, visibility)
		select work.id, $2, $3, $4, work.visibility
		  from works work
		 where work.id = $1
		   and work.lifecycle = 'published'
		   and work.deleted_at is null
		   and (work.taken_down_at is null or $4 = 'owner')
	`, record.WorkID, record.OriginalFileID, record.Format, record.Access)
	if err != nil {
		return fmt.Errorf("record download: %w", err)
	}
	if recorded.RowsAffected() != 1 {
		return work.ErrNotFound
	}
	return nil
}
