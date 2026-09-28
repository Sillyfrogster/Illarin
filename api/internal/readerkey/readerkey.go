// Package readerkey tells one reader from another for a day without keeping who they are
package readerkey

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Reader is the address and user-agent a request came with, hashed and never stored
type Reader struct {
	Address string
	Agent   string
}

// Today is the reader's key for one work on now's UTC day, and the day's first call deletes earlier days
func Today(ctx context.Context, conn db.DBTX, now time.Time, reader Reader, work uuid.UUID) (day time.Time, key []byte, err error) {
	day = now.UTC().Truncate(24 * time.Hour)
	fresh := make([]byte, sha256.Size)
	rand.Read(fresh)
	made, err := conn.Exec(ctx, `insert into reader_secrets (day, secret) values ($1, $2) on conflict (day) do nothing`, day, fresh)
	if err != nil {
		return time.Time{}, nil, fmt.Errorf("make the day's reader secret: %w", err)
	}
	if made.RowsAffected() == 1 {
		if err := Forget(ctx, conn, day); err != nil {
			return time.Time{}, nil, err
		}
	}
	var secret []byte
	if err := conn.QueryRow(ctx, `select secret from reader_secrets where day = $1`, day).Scan(&secret); err != nil {
		return time.Time{}, nil, fmt.Errorf("read the day's reader secret: %w", err)
	}
	mac := hmac.New(sha256.New, secret)
	fmt.Fprintf(mac, "%s\n%s\n%s", reader.Address, reader.Agent, work)
	return day, mac.Sum(nil), nil
}

// Forget deletes every secret from before day and every key they produced
func Forget(ctx context.Context, conn db.DBTX, day time.Time) error {
	if _, err := conn.Exec(ctx, `delete from reader_secrets where day < $1`, day); err != nil {
		return fmt.Errorf("delete earlier reader secrets: %w", err)
	}
	return nil
}

// SecretGone is a key made just before midnight whose day was deleted before the key was saved
func SecretGone(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23503"
}
