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
	ErrNotAuthority    = errors.New("account does not hold publication authority")
	ErrAccountNotFound = errors.New("no such account")
	ErrIncompleteOrder = errors.New("order does not name every member exactly once")

	ErrAppNotFound       = errors.New("no such publication app")
	ErrCategoryNotFound  = errors.New("no such publication category")
	ErrGrantNotFound     = errors.New("no such publication grant")
	ErrAccountUnverified = errors.New("the account has not verified its email")
	ErrAlreadyGranted    = errors.New("the account already publishes for that app")
	ErrGrantRevoked      = errors.New("the grant has been revoked")
)

var ErrCategoryRefused = errors.New("the grant does not cover that category")

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
		return recordPublicationAudit(ctx, tx, made)
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

type Destination = announcements.Destination
type Channel = announcements.Channel
type DestinationEdit = announcements.DestinationEdit
type DestinationUpdate = announcements.DestinationUpdate
type AddedDestination = announcements.AddedDestination
type Choice = announcements.Choice

var PostEvents = announcements.PostEvents
var ErrEventUnknown = announcements.ErrEventUnknown
var ErrNotWebhook = announcements.ErrNotWebhook

const KindWebhook = announcements.KindWebhook
const KindDiscord = announcements.KindDiscord
const DestinationUnverified = announcements.DestinationUnverified
const DestinationActive = announcements.DestinationActive
const DestinationDisabled = announcements.DestinationDisabled

var ErrDestinationNotFound = announcements.ErrDestinationNotFound
var ErrDestinationRefused = announcements.ErrDestinationRefused
var ErrDestinationInactive = announcements.ErrDestinationInactive
var ErrNotProven = announcements.ErrNotProven

const EventVerification = announcements.EventVerification

type ChannelEdit = announcements.ChannelEdit

var ErrNotDiscord = announcements.ErrNotDiscord

type DiscordRepair = announcements.DiscordRepair
type DiscordRepairResult = announcements.DiscordRepairResult
type RotatedSecret = announcements.RotatedSecret

const SecretOverlap = announcements.SecretOverlap

var DeliveryDelays = announcements.DeliveryDelays
var DeliveryAttempts = announcements.DeliveryAttempts

const SettledArrived = announcements.SettledArrived
const SettledExhausted = announcements.SettledExhausted
const SettledRefused = announcements.SettledRefused
const SettledGone = announcements.SettledGone
const SettledRemoved = announcements.SettledRemoved
const SettledDisabled = announcements.SettledDisabled
const SettledMoved = announcements.SettledMoved
const SettledUnconfirmed = announcements.SettledUnconfirmed

type Sender = announcements.Sender
type Delivery = announcements.Delivery
type DeliveryAttempt = announcements.DeliveryAttempt

var ErrDeliveryNotFound = announcements.ErrDeliveryNotFound
var ErrDeliveryUnsettled = announcements.ErrDeliveryUnsettled
var ErrDeliveryUnsendable = announcements.ErrDeliveryUnsendable

const DeliveryPending = announcements.DeliveryPending
const DeliverySending = announcements.DeliverySending
const DeliveryDelivered = announcements.DeliveryDelivered
const DeliveryFailed = announcements.DeliveryFailed
const DeliveryUnconfirmed = announcements.DeliveryUnconfirmed
const AttemptDelivered = announcements.AttemptDelivered
const AttemptRefused = announcements.AttemptRefused
const AttemptUnreachable = announcements.AttemptUnreachable
const AttemptUnconfirmed = announcements.AttemptUnconfirmed
const DeliveryPoll = announcements.DeliveryPoll

type Announcement = announcements.Announcement
type DestinationPolicy = announcements.DestinationPolicy

var ErrRoleRefused = announcements.ErrRoleRefused

func (s *Service) PostDestinations(
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
	allowed, err := s.PostChoices(ctx, found.GrantID)
	if err != nil {
		return nil, err
	}
	return announcements.ActiveAmong(allowed), nil
}

func (s *Service) PostDeliveries(
	ctx context.Context,
	editor Editor,
	postID uuid.UUID,
) ([]Delivery, error) {
	post, err := s.post(ctx, postID)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, post); err != nil {
		return nil, err
	}
	return s.Service.PostDeliveries(ctx, postID)
}

func (s *Service) SetGrantDestinations(ctx context.Context, actor, grantID uuid.UUID, in DestinationPolicy) error {
	current, err := s.grant(ctx, grantID)
	if err != nil {
		return err
	}
	return s.Service.SetGrantDestinations(ctx, actor, grantID, current.App.ID, current.Holder.ID, in)
}

const CredentialSession = "session"

const CredentialSystem = "system"

type change = announcements.Change

func recordPublicationAudit(ctx context.Context, tx pgx.Tx, made change) error {
	if made.Credential == "" {
		made.Credential = CredentialSession
	}
	var actor *uuid.UUID
	if made.Actor != uuid.Nil {
		actor = &made.Actor
	}
	_, err := tx.Exec(ctx, `
		insert into publication_audits
		       (id, actor_id, credential, action, app_id, category_id, grant_id,
		        post_id, revision_id, schedule_id, subject_id,
		        destination_id, delivery_id, before_state, after_state)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, uuid.New(), actor, made.Credential, made.Action, made.AppID, made.CategoryID,
		made.GrantID, made.PostID, made.RevisionID, made.ScheduleID,
		made.SubjectID, made.DestinationID, made.DeliveryID,
		nullable(made.Before), nullable(made.After))
	if err != nil {
		return fmt.Errorf("record publication audit: %w", err)
	}
	return nil
}

type sent = announcements.Event
type sentPost = announcements.EventPost
type sentCategory = announcements.EventCategory
type sentApp = announcements.EventApp
type sentRelease = announcements.EventRelease
type sentByline = announcements.EventByline

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
