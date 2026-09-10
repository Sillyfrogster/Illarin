package publication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

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

const CredentialSession = "session"

const CredentialToken = "token"

const CredentialSystem = "system"

type change struct {
	Actor      uuid.UUID
	Credential string
	Action     string
	AppID      *uuid.UUID
	CategoryID *uuid.UUID
	GrantID    *uuid.UUID
	TokenID    *uuid.UUID
	PostID     *uuid.UUID
	RevisionID *uuid.UUID
	ScheduleID *uuid.UUID
	SubjectID  *uuid.UUID

	DestinationID *uuid.UUID
	DeliveryID    *uuid.UUID
	Before        string
	After         string
}

func recordPublicationAudit(ctx context.Context, tx pgx.Tx, made change) error {
	if made.Credential == "" {
		made.Credential = CredentialSession
	}
	var actor *uuid.UUID
	if made.Actor != uuid.Nil {
		actor = &made.Actor
	}
	_, err := tx.Exec(ctx, `
		insert into publication_audits
		       (id, actor_id, credential, action, app_id, category_id, grant_id,
		        token_id, post_id, revision_id, schedule_id, subject_id,
		        destination_id, delivery_id, before_state, after_state)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`, uuid.New(), actor, made.Credential, made.Action, made.AppID, made.CategoryID,
		made.GrantID, made.TokenID, made.PostID, made.RevisionID, made.ScheduleID,
		made.SubjectID, made.DestinationID, made.DeliveryID,
		nullable(made.Before), nullable(made.After))
	if err != nil {
		return fmt.Errorf("record publication audit: %w", err)
	}
	return nil
}
