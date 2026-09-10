package publication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/credential"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	maxTokenName    = 48
	minTokenLife    = time.Minute
	maxTokenLife    = 10 * 365 * 24 * time.Hour
	tokensPerGrant  = 20
	lastUseInterval = time.Minute
)

type Token struct {
	ID         uuid.UUID
	GrantID    uuid.UUID
	Name       string
	Prefix     string
	CreatedAt  time.Time
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

type Issued struct {
	Token Token
	Value string
}

type TokenEdit struct {
	Name      string
	ExpiresAt *time.Time
}

type Bearer struct {
	Token Token
	Grant Grant
}

func (b Bearer) Editor() Editor {
	return Editor{ID: b.Grant.Holder.ID, Grant: &b.Grant.ID, Token: &b.Token.ID}
}

func (t Token) Live(now time.Time) bool {
	return t.RevokedAt == nil && (t.ExpiresAt == nil || t.ExpiresAt.After(now))
}

func (s *Service) GrantTokens(
	ctx context.Context,
	reader uuid.UUID,
	grantID uuid.UUID,
) ([]Token, error) {
	if _, err := s.readableGrant(ctx, reader, grantID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, selectTokens+`
		 where token.grant_id = $1
		 order by token.revoked_at is not null, token.created_at desc
	`, grantID)
	if err != nil {
		return nil, fmt.Errorf("read publication tokens: %w", err)
	}
	return collectTokens(rows)
}

func (s *Service) IssueToken(
	ctx context.Context,
	actor uuid.UUID,
	grantID uuid.UUID,
	in TokenEdit,
) (Issued, error) {
	held, err := s.grant(ctx, grantID)
	if err != nil {
		return Issued{}, err
	}
	if held.Holder.ID != actor {
		return Issued{}, ErrNotTokenOwner
	}
	if !held.Active {
		return Issued{}, ErrGrantRevoked
	}
	name, expiry, err := validToken(in)
	if err != nil {
		return Issued{}, err
	}
	minted, err := credential.Mint(credential.Publication)
	if err != nil {
		return Issued{}, fmt.Errorf("mint a publication token: %w", err)
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Issued{}, fmt.Errorf("begin token issue: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := roomForOneMore(ctx, tx, grantID); err != nil {
		return Issued{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into publication_tokens (id, grant_id, name, prefix, token_hash, expires_at)
		values ($1, $2, $3, $4, $5, $6)
	`, id, grantID, name, minted.Prefix, minted.Hash, expiry)
	if err != nil {
		return Issued{}, fmt.Errorf("issue a publication token: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "token.issued",
		AppID: &held.App.ID, GrantID: &grantID, TokenID: &id, SubjectID: &held.Holder.ID,
	})
	if err != nil {
		return Issued{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Issued{}, fmt.Errorf("commit token issue: %w", err)
	}
	made, err := s.token(ctx, id)
	if err != nil {
		return Issued{}, err
	}
	return Issued{Token: made, Value: minted.Value}, nil
}

func (s *Service) RevokeToken(ctx context.Context, actor uuid.UUID, id uuid.UUID) error {
	found, err := s.token(ctx, id)
	if err != nil {
		return err
	}
	held, err := s.readableGrant(ctx, actor, found.GrantID)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin token revocation: %w", err)
	}
	defer tx.Rollback(ctx)
	command, err := tx.Exec(ctx, `
		update publication_tokens set revoked_at = now()
		 where id = $1 and revoked_at is null
	`, id)
	if err != nil {
		return fmt.Errorf("revoke a publication token: %w", err)
	}
	if command.RowsAffected() == 0 {
		return ErrTokenNotFound
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "token.revoked",
		AppID: &held.App.ID, GrantID: &held.ID, TokenID: &id, SubjectID: &held.Holder.ID,
	})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit token revocation: %w", err)
	}
	return nil
}

func (s *Service) Bearing(ctx context.Context, value string) (Bearer, error) {
	supplied, ok := credential.Read(value, credential.Publication)
	if !ok {
		return Bearer{}, ErrTokenCredential
	}
	var found Token
	var stored []byte
	var live bool
	err := s.pool.QueryRow(ctx, `
		select token.id, token.grant_id, token.name, token.prefix, token.created_at,
		       token.expires_at, token.last_used_at, token.revoked_at, token.token_hash,
		       grant_row.active and holder.email_verified_at is not null
		         and app.retired_at is null
		  from publication_tokens token
		  join publication_grants grant_row on grant_row.id = token.grant_id
		  join users holder on holder.id = grant_row.user_id
		  join publication_apps app on app.id = grant_row.app_id
		 where token.prefix = $1
	`, supplied.Prefix).Scan(
		&found.ID, &found.GrantID, &found.Name, &found.Prefix, &found.CreatedAt,
		&found.ExpiresAt, &found.LastUsedAt, &found.RevokedAt, &stored, &live,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Bearer{}, ErrTokenCredential
	}
	if err != nil {
		return Bearer{}, fmt.Errorf("read a publication token: %w", err)
	}
	if !credential.Matches(supplied.Hash, stored) {
		return Bearer{}, ErrTokenCredential
	}
	switch {
	case found.RevokedAt != nil:
		return Bearer{}, ErrTokenRevoked
	case !found.Live(time.Now()):
		return Bearer{}, ErrTokenExpired
	case !live:
		return Bearer{}, ErrGrantRevoked
	}
	held, err := s.grant(ctx, found.GrantID)
	if err != nil {
		return Bearer{}, err
	}
	if err := s.markUsed(ctx, found); err != nil {
		return Bearer{}, err
	}
	return Bearer{Token: found, Grant: held}, nil
}

func (s *Service) markUsed(ctx context.Context, found Token) error {
	if found.LastUsedAt != nil && time.Since(*found.LastUsedAt) < lastUseInterval {
		return nil
	}
	_, err := s.pool.Exec(ctx, `
		update publication_tokens set last_used_at = now() where id = $1
	`, found.ID)
	if err != nil {
		return fmt.Errorf("record publication token use: %w", err)
	}
	return nil
}

func (s *Service) readableGrant(
	ctx context.Context,
	reader uuid.UUID,
	grantID uuid.UUID,
) (Grant, error) {
	held, err := s.grant(ctx, grantID)
	if err != nil {
		return Grant{}, err
	}
	if held.Holder.ID == reader {
		return held, nil
	}
	authority, err := s.HoldsAuthority(ctx, reader)
	if err != nil {
		return Grant{}, err
	}
	if !authority {
		return Grant{}, ErrNotTokenOwner
	}
	return held, nil
}

func (s *Service) token(ctx context.Context, id uuid.UUID) (Token, error) {
	rows, err := s.pool.Query(ctx, selectTokens+` where token.id = $1`, id)
	if err != nil {
		return Token{}, fmt.Errorf("read a publication token: %w", err)
	}
	found, err := collectTokens(rows)
	if err != nil {
		return Token{}, err
	}
	if len(found) == 0 {
		return Token{}, ErrTokenNotFound
	}
	return found[0], nil
}

func validToken(in TokenEdit) (string, *time.Time, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || len([]rune(name)) > maxTokenName {
		return "", nil, FieldError{
			Field:   "name",
			Message: "Name the token in 48 characters or fewer.",
		}
	}
	if in.ExpiresAt == nil {
		return name, nil, nil
	}
	life := time.Until(*in.ExpiresAt)
	if life < minTokenLife || life > maxTokenLife {
		return "", nil, FieldError{
			Field:   "expiresAt",
			Message: "Set the expiry between a minute and ten years from now.",
		}
	}
	return name, in.ExpiresAt, nil
}

func roomForOneMore(ctx context.Context, tx pgx.Tx, grantID uuid.UUID) error {
	var live int
	err := tx.QueryRow(ctx, `
		select count(*) from publication_tokens
		 where grant_id = $1 and revoked_at is null
		   and (expires_at is null or expires_at > now())
	`, grantID).Scan(&live)
	if err != nil {
		return fmt.Errorf("count publication tokens: %w", err)
	}
	if live >= tokensPerGrant {
		return FieldError{
			Field:   "name",
			Message: "Revoke a token before making another. Twenty at a time is the limit.",
		}
	}
	return nil
}

const selectTokens = `
	select token.id, token.grant_id, token.name, token.prefix, token.created_at,
	       token.expires_at, token.last_used_at, token.revoked_at
	  from publication_tokens token
	`

func collectTokens(rows pgx.Rows) ([]Token, error) {
	defer rows.Close()
	found := make([]Token, 0, 8)
	for rows.Next() {
		var one Token
		err := rows.Scan(
			&one.ID, &one.GrantID, &one.Name, &one.Prefix, &one.CreatedAt,
			&one.ExpiresAt, &one.LastUsedAt, &one.RevokedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("read a publication token: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication tokens: %w", err)
	}
	return found, nil
}
