package publication

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/blog"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type sent = blog.Event
type sentPost = blog.EventPost
type sentCategory = blog.EventCategory
type sentApp = blog.EventApp
type sentRelease = blog.EventRelease
type sentByline = blog.EventByline

func (s *Service) summary(ctx context.Context, eventID uuid.UUID) (sent, error) {
	return s.summaryWith(ctx, s.pool, eventID)
}

func (s *Service) summaryWith(ctx context.Context, q db.DBTX, eventID uuid.UUID) (sent, error) {
	var held sent
	var post sentPost
	var slug, categorySlug, categoryLabel string
	var socialID *uuid.UUID
	err := q.QueryRow(ctx, `
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
	if post.Release, err = s.sentRelease(ctx, q, post.RevisionID); err != nil {
		return sent{}, err
	}
	if post.Byline, err = s.sentByline(ctx, q, post.ID); err != nil {
		return sent{}, err
	}
	held.Post = post
	return held, nil
}

func (s *Service) sentRelease(ctx context.Context, q db.DBTX, revisionID uuid.UUID) (*sentRelease, error) {
	release, err := s.publishedReleaseWith(ctx, q, revisionID)
	if err != nil || release == nil {
		return nil, err
	}
	return &sentRelease{
		App:     sentApp{Slug: release.App.Slug, Name: release.App.Name, URL: s.appAddress(release.App.Slug)},
		Version: release.Version,
		URL:     release.Address,
	}, nil
}

func (s *Service) sentByline(ctx context.Context, q db.DBTX, postID uuid.UUID) (sentByline, error) {
	byline, err := readByline(ctx, q, postID)
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
