package blog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	AttemptPending     = dispatch.Pending
	AttemptSending     = dispatch.Sending
	AttemptDelivered   = dispatch.Delivered
	AttemptFailed      = dispatch.Failed
	AttemptUnconfirmed = dispatch.Unconfirmed
)

const (
	TryDelivered   = dispatch.OutcomeDelivered
	TryRefused     = dispatch.OutcomeRefused
	TryUnreachable = dispatch.OutcomeUnreachable
	TryUnconfirmed = dispatch.OutcomeUnconfirmed
)

const AttemptPoll = dispatch.Poll

var attemptTables = dispatch.Tables{
	Attempts: "blog_announcement_attempts", Tries: "blog_announcement_tries",
}

var ErrAttemptNotFound = errors.New("no such publication attempt")

var ErrAttemptUnsettled = errors.New("the attempt has not finished trying")

var ErrAttemptUnsendable = errors.New("the integration no longer receives")

type Sender interface {
	Check(address string) (string, error)
	Get(ctx context.Context, address string) (dispatch.Answer, error)
	Post(
		ctx context.Context,
		address string,
		headers map[string]string,
		body []byte,
	) (dispatch.Answer, error)
}

type Attempt struct {
	ID               uuid.UUID
	AnnouncementID   uuid.UUID
	AnnouncementType string
	PostID           uuid.UUID
	PostTitle        string
	RevisionID       uuid.UUID
	Integration      string
	Type             string
	Removed          bool
	State            string
	SettledReason    string
	MessageID        string
	Run              int
	Tries            int
	OccurredAt       time.Time
	DueAt            time.Time
	SettledAt        *time.Time
	Last             *Try
}

type Try = dispatch.Try

func (s *Service) Attempts(ctx context.Context, state string, limit int) ([]Attempt, error) {
	rows, err := s.pool.Query(ctx, selectAttempts+`
		 where $1 = '' or work.state = $1
		 order by work.updated_at desc
		 limit $2
	`, state, limit)
	if err != nil {
		return nil, fmt.Errorf("read publication attempts: %w", err)
	}
	return collectAttempts(rows)
}

func (s *Service) Attempt(ctx context.Context, id uuid.UUID) (Attempt, error) {
	rows, err := s.pool.Query(ctx, selectAttempts+` where work.id = $1`, id)
	if err != nil {
		return Attempt{}, fmt.Errorf("read a publication attempt: %w", err)
	}
	found, err := collectAttempts(rows)
	if err != nil {
		return Attempt{}, err
	}
	if len(found) == 0 {
		return Attempt{}, ErrAttemptNotFound
	}
	return found[0], nil
}

func (s *Service) Tries(ctx context.Context, id uuid.UUID) ([]Try, error) {
	if _, err := s.Attempt(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select run, number, outcome, status, detail, took_ms, attempted_at
		  from blog_announcement_tries
		 where attempt_id = $1
		 order by number
	`, id)
	if err != nil {
		return nil, fmt.Errorf("read the attempts a attempt made: %w", err)
	}
	defer rows.Close()
	made := make([]Try, 0, MaxTries)
	for rows.Next() {
		var one Try
		var took int
		err := rows.Scan(
			&one.Run, &one.Number, &one.Outcome, &one.Status, &one.Detail, &took, &one.Attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one attempt a attempt made: %w", err)
		}
		one.Took = time.Duration(took) * time.Millisecond
		made = append(made, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the attempts a attempt made: %w", err)
	}
	return made, nil
}

func (s *Service) ReplayAttempt(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Attempt, error) {
	held, err := s.Attempt(ctx, id)
	if err != nil {
		return Attempt{}, err
	}
	if held.SettledAt == nil {
		return Attempt{}, ErrAttemptUnsettled
	}
	if held.Removed {
		return Attempt{}, ErrAttemptUnsendable
	}
	if held.Type == TypeDiscord && (held.State == AttemptUnconfirmed || held.MessageID != "") {
		return Attempt{}, FieldError{Field: "attempt", Message: "Check the Discord message and use an explicit repair action."}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Attempt{}, fmt.Errorf("begin attempt replay: %w", err)
	}
	defer tx.Rollback(ctx)
	var state string
	err = tx.QueryRow(ctx, `
		select integration.state
		  from blog_announcement_attempts work
		  join blog_integrations integration on integration.id = work.integration_id
		 where work.id = $1
		   for no key update of integration
	`, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, ErrAttemptUnsendable
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("read whether a replay has anywhere to go: %w", err)
	}
	if state != IntegrationActive {
		return Attempt{}, ErrAttemptUnsendable
	}
	if err := s.ledger.Requeue(ctx, tx, id, s.now()); err != nil {
		return Attempt{}, err
	}
	err = s.Audit(ctx, tx, Change{
		Actor: actor, Action: "attempt.replayed", AttemptID: &id, PostID: &held.PostID,
	})
	if err != nil {
		return Attempt{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Attempt{}, fmt.Errorf("commit attempt replay: %w", err)
	}
	return s.Attempt(ctx, id)
}

func (s *Service) QueueAttempts(
	ctx context.Context,
	tx pgx.Tx,
	postID uuid.UUID,
	eventID uuid.UUID,
	event string,
	chosen []Sending,
) error {
	for _, one := range chosen {
		if one.Type == TypeDiscord && event == PostPublished {
			summary, err := s.Summary(ctx, tx, eventID)
			if err != nil {
				return err
			}
			role := ""
			if one.Ping {
				if err := tx.QueryRow(ctx, `select coalesce(role_id, '') from blog_integrations where id = $1`, one.ID).Scan(&role); err != nil {
					return err
				}
			}
			if _, err := announcementOf(summary, role).Body(); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `
			insert into blog_announcement_attempts
			       (id, announcement_id, integration_id, integration_name, mention_role)
			select $1, $2, integration.id, $4, $6
			  from blog_integrations integration
			 where integration.id = $3 and $5 = any (integration.announcements)
			   and (integration.type <> $7 or not exists (
			         select 1 from blog_announcement_attempts already
			           join blog_announcements sent on sent.id = already.announcement_id
			          where sent.post_id = $8 and already.integration_id = integration.id))
			on conflict do nothing
		`, uuid.New(), eventID, one.ID, one.Name, event, one.Ping, TypeDiscord, postID)
		if err != nil {
			return fmt.Errorf("keep the attempt work: %w", err)
		}
	}
	return nil
}

func (s *Service) stopAttemptsTo(ctx context.Context, tx pgx.Tx, id uuid.UUID, reason string) error {
	return s.ledger.StopTo(ctx, tx, id, dispatch.Stopped(reason))
}

func (s *Service) RunAttempts(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(AttemptPoll)
	defer ticker.Stop()
	for {
		_, err := s.SendDueAttempts(ctx, s.now())
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

func (s *Service) SendDueAttempts(ctx context.Context, now time.Time) (int, error) {
	made := 0
	for {
		held, taken, err := s.ledger.Lease(ctx, now)
		if err != nil || !taken {
			return made, err
		}
		if err := s.sendLeased(ctx, held, now); err != nil {
			return made, err
		}
		made++
	}
}

type carried struct {
	dispatch.Work
	AnnouncementType string
	Ping             bool
}

func (s *Service) sendLeased(ctx context.Context, work dispatch.Work, now time.Time) error {
	held := carried{Work: work}
	err := s.pool.QueryRow(ctx, `
		select event.type, work.mention_role
		  from blog_announcement_attempts work
		  join blog_announcements event on event.id = work.announcement_id
		 where work.id = $1
	`, work.ID).Scan(&held.AnnouncementType, &held.Ping)
	if err != nil {
		return fmt.Errorf("read what a leased attempt carries: %w", err)
	}
	if held.IntegrationID == nil {
		return s.ledger.Record(ctx, held.Work, dispatch.Stopped(SettledRemoved), now)
	}
	integrationType, state, err := s.integrationStanding(ctx, *held.IntegrationID)
	if err != nil {
		return err
	}
	if state != IntegrationActive {
		return s.ledger.Record(ctx, held.Work, dispatch.Stopped(SettledDisabled), now)
	}
	if integrationType == TypeDiscord {
		return s.announceOnDiscord(ctx, held, *held.IntegrationID, now)
	}
	body, err := s.announcementBody(ctx, held.AnnouncementID)
	if err != nil {
		return err
	}
	address, secrets, err := s.endpointOf(ctx, *held.IntegrationID)
	if err != nil {
		return err
	}
	headers, err := dispatch.Headers(secrets, held.ID.String(), now.UTC(), body)
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, address, headers, body)
	if err != nil {
		return s.ledger.Record(ctx, held.Work, dispatch.Unreachable, now)
	}
	said := dispatch.ReadAnswer(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if err := s.ledger.Record(ctx, held.Work, said, now); err != nil {
		return err
	}
	if !said.Gone {
		return nil
	}
	return s.retireIntegration(ctx, *held.IntegrationID)
}

func (s *Service) integrationStanding(ctx context.Context, id uuid.UUID) (string, string, error) {
	var integrationType, state string
	err := s.pool.QueryRow(ctx, `
		select type, state from blog_integrations where id = $1
	`, id).Scan(&integrationType, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("read whether a integration still receives: %w", err)
	}
	return integrationType, state, nil
}

const selectAttempts = `
	select work.id, event.id, event.type, event.post_id, revision.title, event.revision_id,
	       work.integration_name, coalesce(integration.type, ''),
	       work.integration_id is null, work.state,
	       coalesce(work.settled_reason, ''), coalesce(work.message_id, ''),
	       work.run, work.tries,
	       event.occurred_at, work.due_at, work.settled_at,
	       last.run, last.number, last.outcome, last.status, last.detail,
	       last.took_ms, last.attempted_at
	  from blog_announcement_attempts work
	  join blog_announcements event on event.id = work.announcement_id
	  join post_revisions revision on revision.id = event.revision_id
	  left join blog_integrations integration on integration.id = work.integration_id
	  left join lateral (
		select run, number, outcome, status, detail, took_ms, attempted_at
		  from blog_announcement_tries
		 where attempt_id = work.id
		 order by number desc
		 limit 1
	  ) last on true
	`

func collectAttempts(rows pgx.Rows) ([]Attempt, error) {
	defer rows.Close()
	found := make([]Attempt, 0, 4)
	for rows.Next() {
		var one Attempt
		var run, number, status *int
		var outcome, detail *string
		var took *int
		var attempted *time.Time
		err := rows.Scan(
			&one.ID, &one.AnnouncementID, &one.AnnouncementType, &one.PostID, &one.PostTitle, &one.RevisionID,
			&one.Integration, &one.Type, &one.Removed, &one.State,
			&one.SettledReason, &one.MessageID, &one.Run, &one.Tries,
			&one.OccurredAt, &one.DueAt, &one.SettledAt,
			&run, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing a post sent: %w", err)
		}
		if number != nil {
			one.Last = &Try{
				Run: *run, Number: *number, Outcome: *outcome, Status: status, Detail: *detail,
				Took: time.Duration(*took) * time.Millisecond, Attempted: *attempted,
			}
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what a post sent: %w", err)
	}
	return found, nil
}

func (s *Service) PostAttempts(ctx context.Context, postID uuid.UUID) ([]Attempt, error) {
	rows, err := s.pool.Query(ctx, selectAttempts+`
  where event.post_id = $1
  order by event.occurred_at desc, work.integration_name
 `, postID)
	if err != nil {
		return nil, fmt.Errorf("read what a post sent: %w", err)
	}
	return collectAttempts(rows)
}
