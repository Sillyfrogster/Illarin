package work

import (
	"context"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func workToInsertParams(a Work, ownerID uuid.UUID, madeAt *time.Time) db.InsertWorkParams {
	return db.InsertWorkParams{
		Lifecycle:      string(a.Lifecycle),
		WorkVersion:    a.WorkVersion,
		CreditedAuthor: a.CreditedAuthor,
		Nickname:       a.Nickname,
		OriginFormat:   textToNullable(a.OriginFormat),
		ID:             pgtype.UUID{Bytes: a.ID, Valid: true},
		Type:           a.Type,
		OwnerID:        pgtype.UUID{Bytes: ownerID, Valid: true},
		Name:           a.Name,
		Blurb:          a.Blurb,
		Tags:           a.Tags,
		IsNsfw:         boolToNullable(a.IsNSFW),
		Visibility:     string(a.Visibility),
		CreatedAt:      timeToNullable(madeAt),
	}
}

func InsertWork(ctx context.Context, tx pgx.Tx, a Work, ownerID uuid.UUID, madeAt *time.Time) (time.Time, error) {
	queries := db.New(tx)
	params := workToInsertParams(a, ownerID, madeAt)
	made, err := queries.InsertWork(ctx, params)
	if err != nil {
		return time.Time{}, fmt.Errorf("insert work: %w", err)
	}
	return timeFromPgtype(made), nil
}
