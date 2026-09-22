package blog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAccountNotFound   = errors.New("no such account")
	ErrIncompleteOrder   = errors.New("order does not name every member exactly once")
	ErrCategoryNotFound  = errors.New("no such blog category")
	ErrAccountUnverified = errors.New("the account has not verified its email")
)

type FieldError struct {
	Field   string
	Message string
	Cause   error
}

func (e FieldError) Error() string { return e.Message }

func (e FieldError) Unwrap() error { return e.Cause }

type Service struct {
	pool    *pgxpool.Pool
	media   *mediaproc.Library
	discord *integration.Service
	signer  api.URLSigner
	site    string
	now     func() time.Time
}

func NewService(
	pool *pgxpool.Pool,
	media *mediaproc.Library,
	discord *integration.Service,
	site string,
) *Service {
	return &Service{
		pool: pool, media: media, discord: discord, signer: api.NewURLSigner(),
		site: site, now: time.Now,
	}
}

func (s *Service) accountByHandle(ctx context.Context, handle string) (uuid.UUID, error) {
	var accountID uuid.UUID
	err := s.pool.QueryRow(ctx, `select id from users where username = $1`, handle).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAccountNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find account: %w", err)
	}
	return accountID, nil
}

func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func plainField(field, raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	for _, char := range value {
		if unicode.IsControl(char) {
			return "", FieldError{
				Field:   field,
				Message: "Use plain text without control characters.",
			}
		}
	}
	if len([]rune(value)) > limit {
		return "", FieldError{
			Field:   field,
			Message: fmt.Sprintf("Keep this to %d characters or fewer.", limit),
		}
	}
	return value, nil
}

func collectIDs(rows pgx.Rows) ([]uuid.UUID, error) {
	defer rows.Close()
	found := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read an identifier: %w", err)
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read identifiers: %w", err)
	}
	return found, nil
}

func sameSet(present, given []uuid.UUID) bool {
	if len(present) != len(given) {
		return false
	}
	remaining := make(map[uuid.UUID]bool, len(present))
	for _, id := range present {
		remaining[id] = true
	}
	for _, id := range given {
		if !remaining[id] {
			return false
		}
		delete(remaining, id)
	}
	return len(remaining) == 0
}

// Announcement says whether a post's first publication goes to the blog's Discord channel
type Announcement struct {
	Discord bool
}

const CredentialSession = "session"

const CredentialSystem = "system"

type change struct {
	Actor      uuid.UUID
	Credential string
	Action     string
	CategoryID *uuid.UUID
	PostID     *uuid.UUID
	RevisionID *uuid.UUID
	ScheduleID *uuid.UUID
	SubjectID  *uuid.UUID
	Before     string
	After      string
}

func recordActivity(ctx context.Context, tx pgx.Tx, made change) error {
	if made.Credential == "" {
		made.Credential = CredentialSession
	}
	var actor *uuid.UUID
	if made.Actor != uuid.Nil {
		actor = &made.Actor
	}
	_, err := tx.Exec(ctx, `
		insert into blog_activity_log
		       (id, actor_id, credential, action, category_id,
		        post_id, revision_id, schedule_id, subject_id, before_state, after_state)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, uuid.New(), actor, made.Credential, made.Action, made.CategoryID,
		made.PostID, made.RevisionID, made.ScheduleID,
		made.SubjectID, nullable(made.Before), nullable(made.After))
	if err != nil {
		return fmt.Errorf("record blog activity: %w", err)
	}
	return nil
}

func (s *Service) postAddress(slug string) string {
	return strings.TrimRight(s.site, "/") + "/blog/" + url.PathEscape(slug)
}

func (s *Service) profileAddress(handle string) string {
	return strings.TrimRight(s.site, "/") + "/@" + url.PathEscape(handle)
}
