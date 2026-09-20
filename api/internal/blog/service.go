package blog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode"

	"github.com/Sillyfrogster/Illarin/api/internal/db"
	announcements "github.com/Sillyfrogster/Illarin/api/internal/integration/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	mediaproc "github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/secrets"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotAuthority      = errors.New("account does not hold publication authority")
	ErrAccountNotFound   = errors.New("no such account")
	ErrIncompleteOrder   = errors.New("order does not name every member exactly once")
	ErrCategoryNotFound  = errors.New("no such blog category")
	ErrAccountUnverified = errors.New("the account has not verified its email")
)

type FieldError = announcements.FieldError

type Publishing struct {
	Sealing secrets.Key
	Sender  Sender
	Site    string
	Blog    string
}

func DefaultPublishing(sealing secrets.Key, site, blog string) Publishing {
	return Publishing{
		Sealing: sealing,
		Sender:  dispatch.NewCaller(dispatch.DefaultLimits()),
		Site:    site,
		Blog:    blog,
	}
}

type Service struct {
	*announcements.Service
	pool   *pgxpool.Pool
	media  *mediaproc.Library
	signer dispatch.Key
	site   string
	blog   string
	now    func() time.Time
}

func NewService(
	pool *pgxpool.Pool,
	media *mediaproc.Library,
	sending Publishing,
) *Service {
	s := &Service{
		pool: pool, media: media, signer: dispatch.NewKey(),
		site: sending.Site, blog: sending.Blog,
		now: time.Now,
	}
	s.Service = announcements.NewService(pool, sending.Sealing, sending.Sender, s.summaryWith, func(ctx context.Context, tx pgx.Tx, made announcements.Change) error {
		return recordActivity(ctx, tx, made)
	})
	return s
}

func (s *Service) HoldsAuthority(ctx context.Context, accountID uuid.UUID) (bool, error) {
	var held bool
	err := s.pool.QueryRow(ctx, `
		select exists (select 1 from publication_authorities where user_id = $1)
	`, accountID).Scan(&held)
	if err != nil {
		return false, fmt.Errorf("read publication authority: %w", err)
	}
	return held, nil
}

func (s *Service) AssignAuthority(ctx context.Context, handle string) (uuid.UUID, error) {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = s.pool.Exec(ctx, `
		insert into publication_authorities (user_id) values ($1)
		on conflict (user_id) do nothing
	`, accountID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("assign publication authority: %w", err)
	}
	return accountID, nil
}

func (s *Service) accountByHandle(ctx context.Context, handle string) (uuid.UUID, error) {
	var accountID uuid.UUID
	err := s.pool.QueryRow(ctx, `select id from users where username = $1`, handle).Scan(&accountID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrAccountNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("find account: %w", err)
	}
	return accountID, nil
}

func isUniqueViolation(err error) bool {
	var databaseError *pgconn.PgError
	return errors.As(err, &databaseError) && databaseError.Code == "23505"
}

func plainField(field, raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	for _, char := range value {
		if unicode.IsControl(char) {
			return "", FieldError{
				Field:   field,
				Message: "Use plain text without control characters.",
			}
		}
	}
	if len([]rune(value)) > limit {
		return "", FieldError{
			Field:   field,
			Message: fmt.Sprintf("Keep this to %d characters or fewer.", limit),
		}
	}
	return value, nil
}

func collectIDs(rows pgx.Rows) ([]uuid.UUID, error) {
	defer rows.Close()
	found := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read an identifier: %w", err)
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read identifiers: %w", err)
	}
	return found, nil
}

func sameSet(present, given []uuid.UUID) bool {
	if len(present) != len(given) {
		return false
	}
	remaining := make(map[uuid.UUID]bool, len(present))
	for _, id := range present {
		remaining[id] = true
	}
	for _, id := range given {
		if !remaining[id] {
			return false
		}
		delete(remaining, id)
	}
	return len(remaining) == 0
}

type Integration = announcements.Integration
type Channel = announcements.Channel
type IntegrationEdit = announcements.IntegrationEdit
type IntegrationUpdate = announcements.IntegrationUpdate
type AddedIntegration = announcements.AddedIntegration
type Choice = announcements.Choice

var AnnouncementTypes = announcements.AnnouncementTypes
var ErrAnnouncementTypeUnknown = announcements.ErrAnnouncementTypeUnknown
var ErrNotWebhook = announcements.ErrNotWebhook

const TypeWebhook = announcements.TypeWebhook
const TypeDiscord = announcements.TypeDiscord
const IntegrationUnverified = announcements.IntegrationUnverified
const IntegrationActive = announcements.IntegrationActive
const IntegrationDisabled = announcements.IntegrationDisabled

var ErrIntegrationNotFound = announcements.ErrIntegrationNotFound
var ErrIntegrationRefused = announcements.ErrIntegrationRefused
var ErrIntegrationInactive = announcements.ErrIntegrationInactive
var ErrNotProven = announcements.ErrNotProven

const EventVerification = announcements.EventVerification

type ChannelEdit = announcements.ChannelEdit

var ErrNotDiscord = announcements.ErrNotDiscord

type DiscordRepair = announcements.DiscordRepair
type DiscordRepairResult = announcements.DiscordRepairResult
type RotatedSecret = announcements.RotatedSecret

const SecretOverlap = announcements.SecretOverlap

var TryDelays = announcements.TryDelays
var MaxTries = announcements.MaxTries

const SettledArrived = announcements.SettledArrived
const SettledExhausted = announcements.SettledExhausted
const SettledRefused = announcements.SettledRefused
const SettledGone = announcements.SettledGone
const SettledRemoved = announcements.SettledRemoved
const SettledDisabled = announcements.SettledDisabled
const SettledMoved = announcements.SettledMoved
const SettledUnconfirmed = announcements.SettledUnconfirmed

type Sender = announcements.Sender
type Attempt = announcements.Attempt
type Try = announcements.Try

var ErrAttemptNotFound = announcements.ErrAttemptNotFound
var ErrAttemptUnsettled = announcements.ErrAttemptUnsettled
var ErrAttemptUnsendable = announcements.ErrAttemptUnsendable

const AttemptPending = announcements.AttemptPending
const AttemptSending = announcements.AttemptSending
const AttemptDelivered = announcements.AttemptDelivered
const AttemptFailed = announcements.AttemptFailed
const AttemptUnconfirmed = announcements.AttemptUnconfirmed
const TryDelivered = announcements.TryDelivered
const TryRefused = announcements.TryRefused
const TryUnreachable = announcements.TryUnreachable
const TryUnconfirmed = announcements.TryUnconfirmed
const AttemptPoll = announcements.AttemptPoll

type Announcement = announcements.Announcement

var ErrRoleRefused = announcements.ErrRoleRefused

func (s *Service) PostIntegrations(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Choice, error) {
	found, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, found); err != nil {
		return nil, err
	}
	allowed, err := s.PostChoices(ctx)
	if err != nil {
		return nil, err
	}
	return announcements.ActiveAmong(allowed), nil
}

func (s *Service) PostAttempts(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Attempt, error) {
	post, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, post); err != nil {
		return nil, err
	}
	return s.Service.PostAttempts(ctx, postID)
}

const CredentialSession = "session"

const CredentialSystem = "system"

type change = announcements.Change

func recordActivity(ctx context.Context, tx pgx.Tx, made change) error {
	if made.Credential == "" {
		made.Credential = CredentialSession
	}
	var actor *uuid.UUID
	if made.Actor != uuid.Nil {
		actor = &made.Actor
	}
	_, err := tx.Exec(ctx, `
		insert into blog_activity_log
		       (id, actor_id, credential, action, category_id,
		        post_id, revision_id, schedule_id, subject_id,
		        integration_id, attempt_id, before_state, after_state)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, uuid.New(), actor, made.Credential, made.Action, made.CategoryID,
		made.PostID, made.RevisionID, made.ScheduleID,
		made.SubjectID, made.IntegrationID, made.AttemptID,
		nullable(made.Before), nullable(made.After))
	if err != nil {
		return fmt.Errorf("record blog activity: %w", err)
	}
	return nil
}

type sent = announcements.Event
type sentPost = announcements.AnnouncementPost
type sentCategory = announcements.AnnouncementCategory
type sentByline = announcements.AnnouncementByline

func (s *Service) summary(ctx context.Context, eventID uuid.UUID) (sent, error) {
	return s.summaryWith(ctx, s.pool, eventID)
}

func (s *Service) summaryWith(ctx context.Context, q db.DBTX, eventID uuid.UUID) (sent, error) {
	var held sent
	var post sentPost
	var slug, categorySlug, categoryLabel string
	var linkCardID *uuid.UUID
	err := q.QueryRow(ctx, `
		select event.id, event.type, event.occurred_at, event.note,
		       post.id, revision.id, revision.title, revision.summary,
		       category.slug, category.label, post.slug,
		       revision.link_card_media_id, post.published_at, post.updated_public_at
		  from blog_announcements event
		  join posts post on post.id = event.post_id
		  join post_revisions revision on revision.id = event.revision_id
		  join blog_categories category on category.id = revision.category_id
		 where event.id = $1
	`, eventID).Scan(
		&held.ID, &held.Type, &held.OccurredAt, &held.Note,
		&post.ID, &post.RevisionID, &post.Title, &post.Summary,
		&categorySlug, &categoryLabel, &slug,
		&linkCardID, &post.PublishedAt, &post.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sent{}, fmt.Errorf("no such announcement")
	}
	if err != nil {
		return sent{}, fmt.Errorf("read the announcement: %w", err)
	}
	post.Category = sentCategory{Slug: categorySlug, Label: categoryLabel}
	post.URL = s.postAddress(slug)
	if linkCardID != nil {
		post.LinkCardImage = s.postAddress(slug) + "/card.png"
	}
	if post.Byline, err = s.sentByline(ctx, q, post.ID); err != nil {
		return sent{}, err
	}
	held.Post = post
	return held, nil
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
	return shown, nil
}

func (s *Service) postAddress(slug string) string {
	return s.blogAddress("/" + url.PathEscape(slug))
}

func (s *Service) profileAddress(handle string) string {
	return strings.TrimRight(s.site, "/") + "/@" + url.PathEscape(handle)
}

func (s *Service) blogAddress(path string) string {
	return strings.TrimRight(s.blog, "/") + path
}
