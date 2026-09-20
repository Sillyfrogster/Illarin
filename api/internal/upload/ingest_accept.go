package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) AcceptIngest(ctx context.Context, in IngestInput) (Operation, error) {
	stored, err := s.store.Put(ctx, in.File)
	if err != nil {
		return Operation{}, fmt.Errorf("store upload: %w", err)
	}

	id := uuid.New()
	var tags []string
	if in.Tags != nil {
		tags = *in.Tags
	}
	visibility := in.Visibility
	if visibility == "" {
		visibility = work.VisibilityListed
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Operation{}, fmt.Errorf("begin ingest acceptance: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.works.EnsureAccountStorage(ctx, tx, in.OwnerID, []uuid.UUID{stored.ID}); err != nil {
		return Operation{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into upload_operations
			(id, owner_id, blob_id, filename, status, name, blurb, tags, is_nsfw, visibility)
		values ($1, $2, $3, $4, 'pending', $5, $6, $7, $8, $9)
	`, id, in.OwnerID, stored.ID, in.Filename, in.Name, in.Blurb, tags, in.IsNSFW, visibility)
	if err != nil {
		return Operation{}, fmt.Errorf("record ingest: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Operation{}, fmt.Errorf("commit ingest acceptance: %w", err)
	}
	return Operation{ID: id, Status: IngestPending}, nil
}
func (s *Service) GetIngest(ctx context.Context, ownerID, id uuid.UUID) (Operation, error) {
	var status Status
	var workID pgtype.UUID
	var failureReason pgtype.Text
	var failureMessage pgtype.Text
	var replacementPreview []byte
	err := s.pool.QueryRow(ctx, `
		select status, work_id, failure_reason, failure_message, replacement_preview
		  from upload_operations where id = $1 and owner_id = $2
	`, id, ownerID).Scan(&status, &workID, &failureReason, &failureMessage, &replacementPreview)
	if errors.Is(err, pgx.ErrNoRows) {
		return Operation{}, ErrIngestNotFound
	}
	if err != nil {
		return Operation{}, fmt.Errorf("read ingest: %w", err)
	}
	operation := Operation{ID: id, Status: status}
	if status == IngestFailed && failureReason.Valid {
		message := s.ingestFailureMessage(failureReason.String)
		if failureMessage.Valid {
			message = failureMessage.String
		}
		operation.Failure = &Failure{
			Reason:  failureReason.String,
			Message: message,
		}
	}
	if status == IngestPreview {
		var staged stagedReplacement
		if err := json.Unmarshal(replacementPreview, &staged); err != nil {
			return Operation{}, fmt.Errorf("read replacement preview: %w", err)
		}
		operation.Preview = &staged.Preview
	}
	if workID.Valid {
		created, err := work.WorkByID(ctx, s.pool, uuidFromPgtype(workID))
		if err != nil {
			return Operation{}, err
		}
		operation.Work = &created
	}
	return operation, nil
}

func (s *Service) ingestFailureMessage(reason string) string {
	switch reason {
	case "malformed_input":
		return "The file is malformed and could not be read."
	case "unsupported_format":
		return "No supported format recognised this file. Illarin can read " +
			joinReadable(s.reg.ReadableLabels()) +
			". If yours is not one of those, start from nothing and build it here."
	case "unsupported_version":
		return "The file uses a version Illarin cannot read safely."
	case "safety_violation":
		return "The file breaks an archive safety rule."
	case "limit_exceeded":
		return "The file is over a content limit."
	case "wrong_type":
		return "This file is a different type from the work it would update."
	default:
		return "Illarin could not finish this upload. Please try again."
	}
}
func joinReadable(labels []string) string {
	switch len(labels) {
	case 0:
		return "nothing yet"
	case 1:
		return labels[0]
	default:
		return strings.Join(labels[:len(labels)-1], ", ") + " and " + labels[len(labels)-1]
	}
}
