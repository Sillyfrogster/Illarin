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

// recordPublicationAudit keeps who changed which app, category or grant.
func recordPublicationAudit(
	ctx context.Context,
	tx pgx.Tx,
	actor uuid.UUID,
	action string,
	appID, categoryID, grantID, subjectID *uuid.UUID,
) error {
	_, err := tx.Exec(ctx, `
		insert into publication_audits
		       (id, actor_id, action, app_id, category_id, grant_id, subject_id)
		values ($1, $2, $3, $4, $5, $6, $7)
	`, uuid.New(), actor, action, appID, categoryID, grantID, subjectID)
	if err != nil {
		return fmt.Errorf("record publication audit: %w", err)
	}
	return nil
}
