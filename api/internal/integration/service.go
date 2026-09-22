// Package integration keeps each creator's Discord channel and the blog's, and posts announcements to them.
package integration

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotDiscord refuses an address that is not a Discord channel webhook
var ErrNotDiscord = discord.ErrNotACapability

type Service struct {
	pool    *pgxpool.Pool
	sealing secrets.Key
	client  *http.Client
	site    string
	now     func() time.Time
}

// NewService posts through the given client; a nil owner anywhere below means the blog
func NewService(pool *pgxpool.Pool, sealing secrets.Key, client *http.Client, site string) *Service {
	return &Service{
		pool: pool, sealing: sealing, client: client,
		site: strings.TrimRight(site, "/"), now: time.Now,
	}
}

// Connected reports whether the owner has a Discord channel
func (s *Service) Connected(ctx context.Context, owner *uuid.UUID) (bool, error) {
	var found bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from discord_webhooks where owner_id is not distinct from $1)
	`, owner).Scan(&found)
	if err != nil {
		return false, fmt.Errorf("read the Discord channel: %w", err)
	}
	return found, nil
}

// Connect saves the owner's Discord webhook address, replacing any earlier one
func (s *Service) Connect(ctx context.Context, owner *uuid.UUID, raw string) error {
	capability, err := discord.ReadCapability(raw)
	if err != nil {
		return ErrNotDiscord
	}
	sealed, err := s.sealing.Seal([]byte(capability.URL))
	if err != nil {
		return fmt.Errorf("seal the Discord address: %w", err)
	}
	changed, err := s.pool.Exec(ctx, `
		update discord_webhooks set address = $2, updated_at = now()
		 where owner_id is not distinct from $1
	`, owner, sealed)
	if err != nil {
		return fmt.Errorf("replace the Discord address: %w", err)
	}
	if changed.RowsAffected() > 0 {
		return nil
	}
	if _, err := s.pool.Exec(ctx, `
		insert into discord_webhooks (owner_id, address) values ($1, $2)
	`, owner, sealed); err != nil {
		return fmt.Errorf("save the Discord address: %w", err)
	}
	return nil
}

// Disconnect forgets the owner's Discord channel and anything still waiting to go to it
func (s *Service) Disconnect(ctx context.Context, owner *uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `
		delete from discord_webhooks where owner_id is not distinct from $1
	`, owner); err != nil {
		return fmt.Errorf("remove the Discord address: %w", err)
	}
	return nil
}

// Queue posts one announcement to the owner's channel after the transaction commits, if there is a channel
func (s *Service) Queue(ctx context.Context, tx pgx.Tx, owner *uuid.UUID, said discord.Announcement) error {
	body, err := said.Body()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		insert into discord_posts (webhook_id, body)
		select id, $2 from discord_webhooks where owner_id is not distinct from $1
	`, owner, body); err != nil {
		return fmt.Errorf("queue the Discord post: %w", err)
	}
	return nil
}

// Announce posts a public work's new version to its creator's channel when the creator asked for it
func (s *Service) Announce(ctx context.Context, tx pgx.Tx, published version.Version, choice version.Announcement) error {
	if !choice.Discord {
		return nil
	}
	var owner uuid.UUID
	var name, visibility string
	err := tx.QueryRow(ctx, `
		select owner_id, name, visibility from works where id = $1
	`, published.WorkID).Scan(&owner, &name, &visibility)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the work being announced: %w", err)
	}
	if work.Visibility(visibility) != work.VisibilityListed {
		return nil
	}
	update := "#" + strconv.Itoa(published.Number)
	if published.VersionLabel != "" {
		update = published.VersionLabel
	}
	return s.Queue(ctx, tx, &owner, discord.Announcement{
		Title:   name,
		Summary: published.Summary,
		URL:     s.site + "/a/" + published.WorkID.String() + "/history#version-" + strconv.Itoa(published.Number),
		Update:  update,
		At:      s.now(),
		Footer:  discord.Site,
	})
}
