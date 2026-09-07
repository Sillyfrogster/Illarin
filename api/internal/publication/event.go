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

// sent is the whole of what a destination receives. It summarizes a post and
// links to it; the blog stays the only place the article itself lives.
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
	Handle    string   `json:"handle"`
	Name      string   `json:"name"`
	URL       string   `json:"url"`
	Positions []string `json:"positions"`
	App       *sentApp `json:"app,omitempty"`
}

// eventBody writes the exact bytes one event is signed and sent as. Every
// attempt on one event produces the same body, so a receiver that saw a retry
// sees the same summary it saw before.
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

// summary reads what one event says about its post, which is what a webhook is
// sent and what a Discord announcement is composed from.
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
		Handle:    byline.Handle,
		Name:      byline.DisplayName,
		URL:       s.profileAddress(byline.Handle),
		Positions: byline.Positions,
	}
	if shown.Name == "" {
		shown.Name = byline.Handle
	}
	if shown.Positions == nil {
		shown.Positions = []string{}
	}
	if byline.App != nil {
		shown.App = &sentApp{
			Slug: byline.App.Slug, Name: byline.App.Name, URL: s.appAddress(byline.App.Slug),
		}
	}
	return shown, nil
}

// postAddress is the permanent address a published post answers at.
func (s *Service) postAddress(slug string) string {
	return s.blogAddress("/blog/" + url.PathEscape(slug))
}

// appAddress is where the publication collects one app's posts.
func (s *Service) appAddress(slug string) string {
	return s.blogAddress("/blog/app/" + url.PathEscape(slug))
}

// profileAddress is where the person behind a byline is found, which is on
// Illarin itself rather than on the publication.
func (s *Service) profileAddress(handle string) string {
	return strings.TrimRight(s.site, "/") + "/@" + url.PathEscape(handle)
}

func (s *Service) blogAddress(path string) string {
	return strings.TrimRight(s.blog, "/") + path
}
