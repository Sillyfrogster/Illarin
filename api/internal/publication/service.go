package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
	"github.com/Sillyfrogster/Illarin/api/internal/outbox"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/Sillyfrogster/Illarin/api/internal/signing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotAuthority     = errors.New("account does not hold publication authority")
	ErrAccountNotFound  = errors.New("no such account")
	ErrNotFound         = errors.New("no such distinction")
	ErrAlreadyAssigned  = errors.New("account already holds that distinction")
	ErrDistinctionForm  = errors.New("distinction form does not allow that")
	ErrIncompleteOrder  = errors.New("order does not name every member exactly once")
	ErrRetiredAssigning = errors.New("a retired distinction cannot be assigned")

	ErrAppNotFound       = errors.New("no such publication app")
	ErrCategoryNotFound  = errors.New("no such publication category")
	ErrGrantNotFound     = errors.New("no such publication grant")
	ErrSlugTaken         = errors.New("another publication app already uses that slug")
	ErrAccountUnverified = errors.New("the account has not verified its email")
	ErrAlreadyGranted    = errors.New("the account already publishes for that app")
	ErrGrantRevoked      = errors.New("the grant has been revoked")
	ErrTokenNotFound     = errors.New("no such publication token")
	ErrNotTokenOwner     = errors.New("the account neither holds the grant nor publication authority")
	ErrTokenCredential   = errors.New("the value does not identify a live publication token")
	ErrTokenExpired      = errors.New("the publication token has expired")
	ErrTokenRevoked      = errors.New("the publication token has been revoked")
)

var ErrCategoryRefused = errors.New("the grant does not cover that category")

type FieldError struct {
	Field   string
	Message string
	cause   error
}

func (e FieldError) Error() string { return e.Message }

func (e FieldError) Unwrap() error { return e.cause }

type Publishing struct {
	Sealing secrets.Key
	Sender  Sender
	Site    string
	Blog    string
}

func DefaultPublishing(sealing secrets.Key, site, blog string) Publishing {
	return Publishing{
		Sealing: sealing,
		Sender:  outbound.NewCaller(outbound.DefaultLimits()),
		Site:    site,
		Blog:    blog,
	}
}

type Service struct {
	pool    *pgxpool.Pool
	media   *mediaproc.Library
	signer  signing.Key
	sealing secrets.Key
	sender  Sender
	site    string
	blog    string
	rates   Rates
	ledger  outbox.Ledger
	now     func() time.Time
}

func NewService(
	pool *pgxpool.Pool,
	media *mediaproc.Library,
	rates Rates,
	sending Publishing,
) *Service {
	return &Service{
		pool: pool, media: media, signer: signing.NewKey(),
		sealing: sending.Sealing, sender: sending.Sender,
		site: sending.Site, blog: sending.Blog,
		rates: rates, ledger: outbox.NewLedger(pool, deliveryTables), now: time.Now,
	}
}

func (s *Service) HoldsAuthority(ctx context.Context, accountID uuid.UUID) (bool, error) {
	var held bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from publication_authorities where user_id = $1)
	`, accountID).Scan(&held)
	if err != nil {
		return false, fmt.Errorf("read publication authority: %w", err)
	}
	return held, nil
}

func (s *Service) AssignAuthority(ctx context.Context, handle string) (uuid.UUID, error) {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = s.pool.Exec(ctx, `
		insert into publication_authorities (user_id) values ($1)
		on conflict (user_id) do nothing
	`, accountID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("assign publication authority: %w", err)
	}
	return accountID, nil
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
