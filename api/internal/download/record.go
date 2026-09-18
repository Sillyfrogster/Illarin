package download

import (
	"context"
	"fmt"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type AuthorizationClass string

const (
	AuthorizationAnonymous      AuthorizationClass = "anonymous"
	AuthorizationSignedIn       AuthorizationClass = "signed_in"
	AuthorizationOwner          AuthorizationClass = "owner"
	AuthorizationLinkedInstance AuthorizationClass = "linked_instance"
)

type Event struct {
	WorkID             uuid.UUID
	RevisionID         *uuid.UUID
	ExportTarget       string
	AuthorizationClass AuthorizationClass
}

func newEvent(
	workID uuid.UUID,
	revisionID *uuid.UUID,
	target string,
	ownerID *uuid.UUID,
	viewerID *uuid.UUID,
) Event {
	authorization := AuthorizationAnonymous
	if viewerID != nil {
		authorization = AuthorizationSignedIn
		if ownerID != nil && *viewerID == *ownerID {
			authorization = AuthorizationOwner
		}
	}
	return Event{
		WorkID: workID, RevisionID: revisionID, ExportTarget: target,
		AuthorizationClass: authorization,
	}
}

func (s *Service) Record(ctx context.Context, event Event) error {
	recorded, err := s.pool.Exec(ctx, `
		insert into download_events
			(work_id, revision_id, export_target, authorization_class, visibility)
		select work.id, $2, $3, $4, work.visibility
		  from works work
		 where work.id = $1
		   and work.lifecycle = 'published'
		   and work.deleted_at is null
		   and (work.withheld_at is null or $4 = 'owner')
	`, event.WorkID, event.RevisionID, event.ExportTarget, event.AuthorizationClass)
	if err != nil {
		return fmt.Errorf("record download: %w", err)
	}
	if recorded.RowsAffected() != 1 {
		return work.ErrNotFound
	}
	return nil
}
