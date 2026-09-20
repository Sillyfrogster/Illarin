// Package staff holds what staff do to accounts and works, restricting profiles and taking down works
package staff

import (
	"errors"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrProfileNotFound       = errors.New("profile does not exist")
	ErrNotRestricted         = errors.New("profile is not restricted")
	ErrInvalidReason         = errors.New("invalid restricted reason")
	ErrWorkNotFound          = errors.New("work not found")
	ErrInvalidTakedownReason = errors.New("invalid takedown reason")
)

type Service struct {
	pool  *pgxpool.Pool
	works *work.Service
}

func NewService(works *work.Service) *Service {
	return &Service{pool: works.Pool(), works: works}
}

func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func textToPgtype(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}
