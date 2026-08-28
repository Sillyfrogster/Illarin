package publication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// recordAudit keeps who changed what, as identifiers and never as profile text
func recordAudit(
	ctx context.Context,
	tx pgx.Tx,
	actor uuid.UUID,
	action string,
	distinctionID, assignmentID, subjectID *uuid.UUID,
) error {
	_, err := tx.Exec(ctx, `
		insert into profile_distinction_audits
		       (id, actor_id, action, distinction_id, assignment_id, subject_id)
		values ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), actor, action, distinctionID, assignmentID, subjectID)
	if err != nil {
		return fmt.Errorf("record distinction audit: %w", err)
	}
	return nil
}

// change is one entry in the private record of who changed the publication.
type change struct {
	Actor      uuid.UUID
	Action     string
	AppID      *uuid.UUID
	CategoryID *uuid.UUID
	GrantID    *uuid.UUID
	TokenID    *uuid.UUID
	SubjectID  *uuid.UUID
}

// recordPublicationAudit keeps who changed what, as identifiers and never as a
// token value, a profile field or anything else a reader could spend.
func recordPublicationAudit(ctx context.Context, tx pgx.Tx, made change) error {
	_, err := tx.Exec(ctx, `
		insert into publication_audits
		       (id, actor_id, action, app_id, category_id, grant_id, token_id, subject_id)
		values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, uuid.New(), made.Actor, made.Action, made.AppID, made.CategoryID,
		made.GrantID, made.TokenID, made.SubjectID)
	if err != nil {
		return fmt.Errorf("record publication audit: %w", err)
	}
	return nil
}
