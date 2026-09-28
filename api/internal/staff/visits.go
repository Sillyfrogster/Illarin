package staff

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/readerkey"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RecordVisit counts one visit to the site, once per reader on now's UTC day
func (s *Service) RecordVisit(ctx context.Context, reader readerkey.Reader, now time.Time) error {
	day, key, err := readerkey.Today(ctx, s.pool, now, reader, uuid.Nil)
	if err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `
		with seen as (
			insert into reader_keys (day, kind, key) values ($1, 'visit', $2)
			on conflict do nothing returning day
		)
		insert into events (kind, day) select 'visit', day from seen
	`, day, key); err != nil && !readerkey.SecretGone(err) {
		return fmt.Errorf("record a site visit: %w", err)
	}
	return nil
}

// RecordVisit answers 204 whether or not the visit counted
func (h *Handlers) RecordVisit(c *gin.Context) {
	if agent := c.Request.UserAgent(); api.CountsAsReader(agent) {
		reader := readerkey.Reader{Address: api.RequestSource(c), Agent: agent}
		if err := h.staff.RecordVisit(c.Request.Context(), reader, time.Now()); err != nil {
			api.Refuse(c, http.StatusInternalServerError, "could not count the visit")
			return
		}
	}
	c.Status(http.StatusNoContent)
}
