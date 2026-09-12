package publication

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sent struct {
	ID         uuid.UUID `json:"id"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurredAt"`
	Post       sentPost  `json:"post"`
	Note       string    `json:"note,omitempty"`
}

type sentPost struct {
	ID          uuid.UUID    `json:"id"`
	RevisionID  uuid.UUID    `json:"revisionId"`
	Title       string       `json:"title"`
	Summary     string       `json:"summary"`
	Category    sentCategory `json:"category"`
	URL         string       `json:"url"`
	SocialImage string       `json:"socialImageUrl,omitempty"`
	PublishedAt time.Time    `json:"publishedAt"`
	UpdatedAt   *time.Time   `json:"updatedAt"`
	Release     *sentRelease `json:"release,omitempty"`
	Byline      sentByline   `json:"byline"`
}

type sentCategory struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
}

type sentApp struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type sentRelease struct {
	App     sentApp `json:"app"`
	Version string  `json:"version"`
	URL     string  `json:"url,omitempty"`
}

type sentByline struct {
	Handle string   `json:"handle"`
	Name   string   `json:"name"`
	URL    string   `json:"url"`
	App    *sentApp `json:"app,omitempty"`
}

func (s *Service) eventBody(ctx context.Context, eventID uuid.UUID) ([]byte, error) {
	held, err := s.summary(ctx, eventID)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(held)
	if err != nil {
		return nil, fmt.Errorf("write the publication event: %w", err)
	}
	return body, nil
}

func (s *Service) summary(ctx context.Context, eventID uuid.UUID) (sent, error) {
	var held sent
	var post sentPost
	var slug, categorySlug, categoryLabel string
	var socialID *uuid.UUID
	err := s.pool.QueryRow(ctx, `
		select event.id, event.type, event.occurred_at, event.note,
		       post.id, revision.id, revision.title, revision.summary,
		       category.slug, category.label, post.slug,
		       revision.social_media_id, post.published_at, post.updated_public_at
		  from publication_events event
		  join posts post on post.id = event.post_id
		  join post_revisions revision on revision.id = event.revision_id
		  join publication_categories category on category.id = revision.category_id
		 where event.id = $1
	`, eventID).Scan(
		&held.ID, &held.Type, &held.OccurredAt, &held.Note,
		&post.ID, &post.RevisionID, &post.Title, &post.Summary,
		&categorySlug, &categoryLabel, &slug,
		&socialID, &post.PublishedAt, &post.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sent{}, fmt.Errorf("no such publication event")
	}
	if err != nil {
		return sent{}, fmt.Errorf("read the publication event: %w", err)
	}
	post.Category = sentCategory{Slug: categorySlug, Label: categoryLabel}
	post.URL = s.postAddress(slug)
	if socialID != nil {
		post.SocialImage = s.postAddress(slug) + "/card.png"
	}
	if post.Release, err = s.sentRelease(ctx, post.RevisionID); err != nil {
		return sent{}, err
	}
	if post.Byline, err = s.sentByline(ctx, post.ID); err != nil {
		return sent{}, err
	}
	held.Post = post
	return held, nil
}

func (s *Service) sentRelease(ctx context.Context, revisionID uuid.UUID) (*sentRelease, error) {
	release, err := s.publishedRelease(ctx, revisionID)
	if err != nil || release == nil {
		return nil, err
	}
	return &sentRelease{
		App:     sentApp{Slug: release.App.Slug, Name: release.App.Name, URL: s.appAddress(release.App.Slug)},
		Version: release.Version,
		URL:     release.Address,
	}, nil
}

func (s *Service) sentByline(ctx context.Context, postID uuid.UUID) (sentByline, error) {
	byline, err := readByline(ctx, s.pool, postID)
	if err != nil {
		return sentByline{}, err
	}
	shown := sentByline{
		Handle: byline.Handle,
		Name:   byline.DisplayName,
		URL:    s.profileAddress(byline.Handle),
	}
	if shown.Name == "" {
		shown.Name = byline.Handle
	}
	if byline.App != nil {
		shown.App = &sentApp{
			Slug: byline.App.Slug, Name: byline.App.Name, URL: s.appAddress(byline.App.Slug),
		}
	}
	return shown, nil
}

func (s *Service) postAddress(slug string) string {
	return s.blogAddress("/" + url.PathEscape(slug))
}

func (s *Service) appAddress(slug string) string {
	return s.blogAddress("/app/" + url.PathEscape(slug))
}

func (s *Service) profileAddress(handle string) string {
	return strings.TrimRight(s.site, "/") + "/@" + url.PathEscape(handle)
}

func (s *Service) blogAddress(path string) string {
	return strings.TrimRight(s.blog, "/") + path
}
