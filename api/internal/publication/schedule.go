package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The states a schedule passes through. A post has at most one schedule
// pending or publishing at a time.
const (
	SchedulePending    = "pending"
	SchedulePublishing = "publishing"
	SchedulePublished  = "published"
	ScheduleCancelled  = "cancelled"
	ScheduleStopped    = "stopped"
)

// SchedulePoll is how often the worker looks for an edition that is due.
const SchedulePoll = 15 * time.Second

// ScheduleLease is how long one attempt holds a due schedule before another
// attempt may take it over.
const ScheduleLease = 2 * time.Minute

// ScheduleAttempts is how many times publication is tried before the schedule
// is stopped and left for a person.
const ScheduleAttempts = 5

var (
	ErrScheduleNotFound   = errors.New("no such schedule for that post")
	ErrAlreadyScheduled   = errors.New("the post already has an edition waiting to publish")
	ErrSchedulePublishing = errors.New("the schedule is already publishing")
)

// The reasons Illarin stops a schedule rather than publishing what it names.
const (
	stoppedByRevocation = "The approval behind this post was revoked."
	stoppedByAddress    = "Another post took this address before the edition went live."
	stoppedByFailure    = "Illarin could not publish this edition."
)

// Schedule is a post's most recent plan to publish one exact edition at one
// instant. It never holds the edition, only names it.
type Schedule struct {
	ID             uuid.UUID
	RevisionID     uuid.UUID
	RevisionNumber int
	At             time.Time
	State          string
	StoppedBecause string
	CreatedBy      string
	CreatedAt      time.Time
}

// Waiting says the schedule still means to publish something.
func (s Schedule) Waiting() bool {
	return s.State == SchedulePending || s.State == SchedulePublishing
}

// SchedulePost captures the named working copy and sets that exact edition to
// go live at one instant. Editing the working copy afterwards cannot reach it.
func (s *Service) SchedulePost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
	at time.Time,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := s.checkInstant(at); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin scheduling: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Document, err = readyToPublish(locked); err != nil {
		return Post{}, err
	}
	revisionID, err := captureRevision(ctx, tx, editor, locked, RevisionPublication)
	if err != nil {
		return Post{}, err
	}
	if err := carryUsesForward(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	scheduleID := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into post_schedules (id, post_id, revision_id, due_at, created_by)
		values ($1, $2, $3, $4, $5)
	`, scheduleID, id, revisionID, at.UTC(), editor.ID)
	if isUniqueViolation(err) {
		return Post{}, ErrAlreadyScheduled
	}
	if err != nil {
		return Post{}, fmt.Errorf("keep the schedule: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.scheduled",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, RevisionID: &revisionID, ScheduleID: &scheduleID,
		Before: locked.Status, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit scheduling: %w", err)
	}
	return s.post(ctx, id)
}

// ReplaceSchedule points a waiting schedule at another edition the post has
// already kept. The edition it leaves stays exactly as it was.
func (s *Service) ReplaceSchedule(
	ctx context.Context,
	editor Editor,
	id, revisionID uuid.UUID,
	at time.Time,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := s.checkInstant(at); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin schedule replacement: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	waiting, err := lockWaitingSchedule(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if _, err := lockedRevision(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update post_schedules
		   set revision_id = $2, due_at = $3, attempts = 0,
		       lease_token = null, lease_expires_at = null, updated_at = now()
		 where id = $1
	`, waiting, revisionID, at.UTC())
	if err != nil {
		return Post{}, fmt.Errorf("replace the scheduled edition: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.schedule.replaced",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, RevisionID: &revisionID, ScheduleID: &waiting,
		Before: locked.Status, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit schedule replacement: %w", err)
	}
	return s.post(ctx, id)
}

// CancelSchedule stops a waiting schedule. A post with nothing waiting is
// already in the state the caller asked for, so asking again changes nothing.
func (s *Service) CancelSchedule(ctx context.Context, editor Editor, id uuid.UUID) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin schedule cancellation: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	waiting, err := lockWaitingSchedule(ctx, tx, id)
	if errors.Is(err, ErrScheduleNotFound) {
		return current, nil
	}
	if err != nil {
		return Post{}, err
	}
	if err := settleSchedule(ctx, tx, waiting, ScheduleCancelled, ""); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.schedule.cancelled",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, ScheduleID: &waiting,
		Before: locked.Status, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit schedule cancellation: %w", err)
	}
	return s.post(ctx, id)
}

// RunScheduler publishes due editions until the context is done.
func (s *Service) RunScheduler(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(SchedulePoll)
	defer ticker.Stop()
	for {
		_, err := s.PublishDueSchedules(ctx, s.now())
		if err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PublishDueSchedules publishes every edition due at the given instant and
// answers how many it settled.
func (s *Service) PublishDueSchedules(ctx context.Context, now time.Time) (int, error) {
	settled := 0
	for {
		leased, held, err := s.leaseDueSchedule(ctx, now)
		if err != nil || !held {
			return settled, err
		}
		if err := s.publishLeased(ctx, leased); err != nil {
			return settled, err
		}
		settled++
	}
}

// leased is one due schedule this worker holds and the work it names.
type leased struct {
	ID         uuid.UUID
	PostID     uuid.UUID
	RevisionID uuid.UUID
	CreatedBy  *uuid.UUID
	Token      uuid.UUID
	Attempts   int
}

// leaseDueSchedule takes one due schedule, including one an earlier attempt
// stopped holding, and answers false when nothing is due.
func (s *Service) leaseDueSchedule(ctx context.Context, now time.Time) (leased, bool, error) {
	var held leased
	held.Token = uuid.New()
	err := s.pool.QueryRow(ctx, `
		with candidate as (
			select id
			  from post_schedules
			 where due_at <= $1
			   and (state = $4
			        or (state = $5 and lease_expires_at <= $1))
			 order by due_at
			 for update skip locked
			 limit 1
		)
		update post_schedules schedule
		   set state = $5, attempts = attempts + 1,
		       lease_token = $2, lease_expires_at = $3, updated_at = $1
		  from candidate
		 where schedule.id = candidate.id
		returning schedule.id, schedule.post_id, schedule.revision_id,
		          schedule.created_by, schedule.attempts
	`, now, held.Token, now.Add(ScheduleLease), SchedulePending, SchedulePublishing).Scan(
		&held.ID, &held.PostID, &held.RevisionID, &held.CreatedBy, &held.Attempts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return leased{}, false, nil
	}
	if err != nil {
		return leased{}, false, fmt.Errorf("lease a due schedule: %w", err)
	}
	return held, true, nil
}

// publishLeased puts the exact edition a leased schedule names in front of
// readers, or stops the schedule when it may no longer be published.
func (s *Service) publishLeased(ctx context.Context, held leased) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin scheduled publication: %w", err)
	}
	defer tx.Rollback(ctx)
	still, err := heldByThisAttempt(ctx, tx, held)
	if err != nil || !still {
		return err
	}
	locked, err := lockPost(ctx, tx, held.PostID)
	if err != nil {
		return err
	}
	refusal, err := s.refusesLeased(ctx, tx, held, locked)
	if err != nil {
		return err
	}
	if refusal != "" {
		if err := settleSchedule(ctx, tx, held.ID, ScheduleStopped, refusal); err != nil {
			return err
		}
		if err := recordScheduleRun(ctx, tx, held, locked, "post.schedule.stopped"); err != nil {
			return err
		}
		return commitScheduleRun(ctx, tx)
	}
	kept, err := lockedRevision(ctx, tx, held.PostID, held.RevisionID)
	if err != nil {
		return err
	}
	if err := makePublic(ctx, tx, locked, held.RevisionID, actorOf(held, locked), kept.Slug); err != nil {
		return err
	}
	if err := settleSchedule(ctx, tx, held.ID, SchedulePublished, ""); err != nil {
		return err
	}
	if err := recordScheduleRun(ctx, tx, held, locked, "post.published"); err != nil {
		return err
	}
	return commitScheduleRun(ctx, tx)
}

// refusesLeased answers why this edition may no longer go live, or the empty
// string when nothing stands in its way.
func (s *Service) refusesLeased(
	ctx context.Context,
	tx pgx.Tx,
	held leased,
	locked working,
) (string, error) {
	if held.Attempts > ScheduleAttempts {
		return stoppedByFailure, nil
	}
	if locked.GrantID != nil {
		var active bool
		err := tx.QueryRow(ctx, `
			select active from publication_grants where id = $1
		`, *locked.GrantID).Scan(&active)
		if err != nil {
			return "", fmt.Errorf("read the approval behind a scheduled post: %w", err)
		}
		if !active {
			return stoppedByRevocation, nil
		}
	}
	if locked.PublishedAt != nil {
		return "", nil
	}
	var address string
	err := tx.QueryRow(ctx, `
		select slug from post_revisions where id = $1 and post_id = $2
	`, held.RevisionID, held.PostID).Scan(&address)
	if errors.Is(err, pgx.ErrNoRows) {
		return stoppedByFailure, nil
	}
	if err != nil {
		return "", fmt.Errorf("read the address a scheduled edition wants: %w", err)
	}
	taken, err := addressTaken(ctx, tx, held.PostID, address)
	if err != nil {
		return "", err
	}
	if taken {
		return stoppedByAddress, nil
	}
	return "", nil
}

// heldByThisAttempt answers whether the lease this attempt took is still the
// one on the row, so an attempt someone else took over gives up quietly.
func heldByThisAttempt(ctx context.Context, tx pgx.Tx, held leased) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, `
		select true from post_schedules
		 where id = $1 and lease_token = $2 and state = $3
		   for update
	`, held.ID, held.Token, SchedulePublishing).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the schedule lease: %w", err)
	}
	return found, nil
}

// lockWaitingSchedule holds the one schedule a post is waiting on, and refuses
// to hand over one an attempt is already publishing.
func lockWaitingSchedule(ctx context.Context, tx pgx.Tx, postID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	var state string
	err := tx.QueryRow(ctx, `
		select id, state from post_schedules
		 where post_id = $1 and state in ($2, $3)
		   for update
	`, postID, SchedulePending, SchedulePublishing).Scan(&id, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrScheduleNotFound
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("hold the waiting schedule: %w", err)
	}
	if state == SchedulePublishing {
		return uuid.Nil, ErrSchedulePublishing
	}
	return id, nil
}

func settleSchedule(ctx context.Context, tx pgx.Tx, id uuid.UUID, state, because string) error {
	_, err := tx.Exec(ctx, `
		update post_schedules
		   set state = $2, stopped_because = $3, settled_at = now(),
		       lease_token = null, lease_expires_at = null, updated_at = now()
		 where id = $1
	`, id, state, nullable(because))
	if err != nil {
		return fmt.Errorf("settle the schedule: %w", err)
	}
	return nil
}

// recordScheduleRun keeps what the worker did under the account that asked for
// it, so an unattended action still names a person.
func recordScheduleRun(
	ctx context.Context,
	tx pgx.Tx,
	held leased,
	locked working,
	action string,
) error {
	after := locked.Status
	if action == "post.published" {
		after = StatusPublished
	}
	return recordPublicationAudit(ctx, tx, change{
		Actor: actorOf(held, locked), Credential: CredentialSystem, Action: action,
		GrantID: locked.GrantID, PostID: &held.PostID, RevisionID: &held.RevisionID,
		ScheduleID: &held.ID, Before: locked.Status, After: after,
	})
}

func commitScheduleRun(ctx context.Context, tx pgx.Tx) error {
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit scheduled publication: %w", err)
	}
	return nil
}

// actorOf names the account a scheduled run acts for, falling back to the post's
// author when the account that made the schedule is gone.
func actorOf(held leased, locked working) uuid.UUID {
	if held.CreatedBy != nil {
		return *held.CreatedBy
	}
	return locked.AuthorID
}

// checkInstant keeps a schedule pointed at a moment that has not passed.
func (s *Service) checkInstant(at time.Time) error {
	if at.IsZero() {
		return FieldError{Field: "at", Message: "Say when the post goes live."}
	}
	if !at.After(s.now()) {
		return FieldError{Field: "at", Message: "Choose a time that has not passed yet."}
	}
	return nil
}

// stopSchedulesUnder takes the waiting schedules of every post under one grant
// out of the queue, so revoked approval cannot publish later.
func stopSchedulesUnder(ctx context.Context, tx pgx.Tx, grantID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		update post_schedules schedule
		   set state = $2, stopped_because = $3, settled_at = now(),
		       lease_token = null, lease_expires_at = null, updated_at = now()
		  from posts post
		 where post.id = schedule.post_id and post.grant_id = $1
		   and schedule.state in ($4, $5)
	`, grantID, ScheduleStopped, stoppedByRevocation, SchedulePending, SchedulePublishing)
	if err != nil {
		return fmt.Errorf("stop the schedules a revoked approval left: %w", err)
	}
	return nil
}

// attachSchedules gives each post the last schedule it was given, if any.
func (s *Service) attachSchedules(ctx context.Context, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(posts))
	for index := range posts {
		ids = append(ids, posts[index].ID)
	}
	rows, err := s.pool.Query(ctx, `
		select distinct on (schedule.post_id)
		       schedule.post_id, schedule.id, schedule.revision_id, revision.number,
		       schedule.due_at, schedule.state, schedule.stopped_because,
		       maker.username, schedule.created_at
		  from post_schedules schedule
		  join post_revisions revision on revision.id = schedule.revision_id
		  left join users maker on maker.id = schedule.created_by
		 where schedule.post_id = any($1)
		 order by schedule.post_id, schedule.created_at desc
	`, ids)
	if err != nil {
		return fmt.Errorf("read what posts are waiting to publish: %w", err)
	}
	defer rows.Close()
	latest := make(map[uuid.UUID]Schedule, len(posts))
	for rows.Next() {
		var postID uuid.UUID
		var one Schedule
		var because, maker *string
		err := rows.Scan(
			&postID, &one.ID, &one.RevisionID, &one.RevisionNumber,
			&one.At, &one.State, &because, &maker, &one.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("read what a post is waiting to publish: %w", err)
		}
		if because != nil {
			one.StoppedBecause = *because
		}
		if maker != nil {
			one.CreatedBy = *maker
		}
		latest[postID] = one
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read what posts are waiting to publish: %w", err)
	}
	for index := range posts {
		if found, held := latest[posts[index].ID]; held {
			schedule := found
			posts[index].Schedule = &schedule
		}
	}
	return nil
}
