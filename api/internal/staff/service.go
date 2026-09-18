// Package staff holds what staff do to accounts and works, restricting profiles and withholding works
package staff

import (
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound       = errors.New("profile does not exist")
	ErrNotRestricted         = errors.New("profile is not restricted")
	ErrInvalidReason         = errors.New("invalid restriction reason")
	ErrWorkNotFound          = errors.New("work not found")
	ErrInvalidWithholdReason = errors.New("invalid withhold reason")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func textToPgtype(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}
