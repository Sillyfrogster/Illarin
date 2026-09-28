package page

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/readerkey"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// knownBots stops only honest bots; a bot posing as a browser still counts as a view
var knownBots = regexp.MustCompile(`(?i)bot\b|crawl|spider|slurp|headless|facebookexternalhit|curl|wget|python|go-http`)

// RecordView counts one display of a published work's page by anyone but its creator, once per reader on now's UTC day
func (s *Service) RecordView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID, reader readerkey.Reader, now time.Time) error {
	day, key, err := readerkey.Today(ctx, s.pool, now, reader, id)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		with work as (
			select work.id from works work
			 where work.id = $1 and work.lifecycle = 'published'
			   and work.deleted_at is null and work.taken_down_at is null
			   and work.owner_id is distinct from $2
		), seen as (
			insert into reader_keys (day, key) select $3, $4 from work
			on conflict do nothing returning day
		)
		insert into events (kind, work_id, day) select 'view', $1, day from seen
	`, id, viewerID, day, key); err != nil && !secretGone(err) {
		return fmt.Errorf("record a work view: %w", err)
	}
	return nil
}

// secretGone is a view keyed just before midnight whose day was deleted before it was saved
func secretGone(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23503"
}

// lifetimeCounts reads a work's views, downloads counting sends, sends alone and current followers
func lifetimeCounts(ctx context.Context, tx pgx.Tx, id uuid.UUID) (views, downloads, sends, followers int, err error) {
	err = tx.QueryRow(ctx, `
		select coalesce(sum(count) filter (where kind = 'view'), 0),
		       coalesce(sum(count) filter (where kind in ('download', 'send')), 0),
		       coalesce(sum(count) filter (where kind = 'send'), 0),
		       (select count(*) from public.work_followers where work_id = $1)
		  from public.work_day_counts where work_id = $1
	`, id).Scan(&views, &downloads, &sends, &followers)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("read the work's counts: %w", err)
	}
	return views, downloads, sends, followers, nil
}

// RecordView answers 204 whether or not the display counted, so it tells a caller nothing about the work
func (h *Handlers) RecordView(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	if agent := c.Request.UserAgent(); agent != "" && !knownBots.MatchString(agent) {
		reader := readerkey.Reader{Address: api.RequestSource(c), Agent: agent}
		if err := h.works.RecordView(c.Request.Context(), id, viewerID, reader, time.Now()); err != nil {
			api.Refuse(c, http.StatusInternalServerError, "could not count the view")
			return
		}
	}
	c.Status(http.StatusNoContent)
}
