package page

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool  *pgxpool.Pool
	reg   *format.Registry
	works *work.Service
}

func NewService(pool *pgxpool.Pool, works *work.Service) *Service {
	return &Service{pool: pool, reg: works.Registry(), works: works}
}

func uuidToPgtype(u uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: u, Valid: true}
}

func uuidToNullable(u *uuid.UUID) pgtype.UUID {
	if u == nil {
		return pgtype.UUID{Valid: false}
	}
	return uuidToPgtype(*u)
}

func uuidFromPgtype(p pgtype.UUID) uuid.UUID {
	return p.Bytes
}

func timeFromPgtype(p pgtype.Timestamptz) time.Time {
	return p.Time
}

func timeToNullable(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func boolFromPgtype(b pgtype.Bool) *bool {
	if !b.Valid {
		return nil
	}
	value := b.Bool
	return &value
}
