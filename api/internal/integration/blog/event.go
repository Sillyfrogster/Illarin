package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurredAt"`
	Post       EventPost `json:"post"`
	Note       string    `json:"note,omitempty"`
}

type EventPost struct {
	ID          uuid.UUID     `json:"id"`
	RevisionID  uuid.UUID     `json:"revisionId"`
	Title       string        `json:"title"`
	Summary     string        `json:"summary"`
	Category    EventCategory `json:"category"`
	URL         string        `json:"url"`
	SocialImage string        `json:"socialImageUrl,omitempty"`
	PublishedAt time.Time     `json:"publishedAt"`
	UpdatedAt   *time.Time    `json:"updatedAt"`
	Release     *EventRelease `json:"release,omitempty"`
	Byline      EventByline   `json:"byline"`
}

type EventCategory struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type EventApp struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type EventRelease struct {
	App     EventApp `json:"app"`
	Version string   `json:"version"`
	URL     string   `json:"url,omitempty"`
}

type EventByline struct {
	Handle string    `json:"handle"`
	Name   string    `json:"name"`
	URL    string    `json:"url"`
	App    *EventApp `json:"app,omitempty"`
}

func (s *Service) eventBody(ctx context.Context, eventID uuid.UUID) ([]byte, error) {
	held, err := s.Summary(ctx, s.pool, eventID)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(held)
	if err != nil {
		return nil, fmt.Errorf("write the publication event: %w", err)
	}
	return body, nil
}
