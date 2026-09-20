package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID        `json:"id"`
	Type       string           `json:"type"`
	OccurredAt time.Time        `json:"occurredAt"`
	Post       AnnouncementPost `json:"post"`
	Note       string           `json:"note,omitempty"`
}

type AnnouncementPost struct {
	ID            uuid.UUID            `json:"id"`
	RevisionID    uuid.UUID            `json:"revisionId"`
	Title         string               `json:"title"`
	Summary       string               `json:"summary"`
	Category      AnnouncementCategory `json:"category"`
	URL           string               `json:"url"`
	LinkCardImage string               `json:"linkCardImageUrl,omitempty"`
	PublishedAt   time.Time            `json:"publishedAt"`
	UpdatedAt     *time.Time           `json:"updatedAt"`
	Byline        AnnouncementByline   `json:"byline"`
}

type AnnouncementCategory struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type AnnouncementByline struct {
	Handle string `json:"handle"`
	Name   string `json:"name"`
	URL    string `json:"url"`
}

func (s *Service) announcementBody(ctx context.Context, eventID uuid.UUID) ([]byte, error) {
	held, err := s.Summary(ctx, s.pool, eventID)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(held)
	if err != nil {
		return nil, fmt.Errorf("write the announcement: %w", err)
	}
	return body, nil
}
