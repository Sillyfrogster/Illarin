package page

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// knownBots stops only honest bots; a bot posing as a browser still counts as a view
var knownBots = regexp.MustCompile(`(?i)bot\b|crawl|spider|slurp|headless|facebookexternalhit|curl|wget|python|go-http`)

// RecordView counts one display of a published work's page by anyone but its creator
func (s *Service) RecordView(ctx context.Context, id uuid.UUID, viewerID *uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `
		insert into events (kind, work_id)
		select 'view', work.id from works work
		 where work.id = $1 and work.lifecycle = 'published'
		   and work.deleted_at is null and work.taken_down_at is null
		   and work.owner_id is distinct from $2
	`, id, viewerID); err != nil {
		return fmt.Errorf("record a work view: %w", err)
	}
	return nil
}

// lifetimeCounts reads a work's views, public downloads, sends and current followers
func lifetimeCounts(ctx context.Context, tx pgx.Tx, id uuid.UUID) (views, downloads, sends, followers int, err error) {
	err = tx.QueryRow(ctx, `
		with counted as (
			select kind, count from public.daily_totals where work_id = $1 and kind in ('view', 'send')
			union all
			select event.kind, 1 from public.events event
			 where event.work_id = $1 and event.kind in ('view', 'send')
			   and not exists (select from public.daily_totals total
			                    where total.day = event.day and total.kind = event.kind and total.work_id = $1)
		)
		select coalesce(sum(count) filter (where kind = 'view'), 0),
		       (select count(*) from public.download_records where work_id = $1 and access = 'public'),
		       coalesce(sum(count) filter (where kind = 'send'), 0),
		       (select count(*) from public.work_followers where work_id = $1)
		  from counted
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
		if err := h.works.RecordView(c.Request.Context(), id, viewerID); err != nil {
			api.Refuse(c, http.StatusInternalServerError, "could not count the view")
			return
		}
	}
	c.Status(http.StatusNoContent)
}
