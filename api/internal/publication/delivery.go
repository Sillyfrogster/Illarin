package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
	"github.com/Sillyfrogster/Illarin/api/internal/outbox"
	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	DeliveryPending     = outbox.Pending
	DeliverySending     = outbox.Sending
	DeliveryDelivered   = outbox.Delivered
	DeliveryFailed      = outbox.Failed
	DeliveryUnconfirmed = outbox.Unconfirmed
)

const (
	AttemptDelivered   = outbox.OutcomeDelivered
	AttemptRefused     = outbox.OutcomeRefused
	AttemptUnreachable = outbox.OutcomeUnreachable
	AttemptUnconfirmed = outbox.OutcomeUnconfirmed
)

const DeliveryPoll = outbox.Poll

var deliveryTables = outbox.Tables{
	Deliveries: "publication_deliveries", Attempts: "publication_delivery_attempts",
}

var ErrDeliveryNotFound = errors.New("no such publication delivery")

var ErrDeliveryUnsettled = errors.New("the delivery has not finished trying")

var ErrDeliveryUnsendable = errors.New("the destination no longer receives")

type Sender interface {
	Check(address string) (string, error)
	Get(ctx context.Context, address string) (outbound.Answer, error)
	Post(
		ctx context.Context,
		address string,
		headers map[string]string,
		body []byte,
	) (outbound.Answer, error)
}

type Delivery struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	EventType     string
	PostID        uuid.UUID
	PostTitle     string
	RevisionID    uuid.UUID
	Destination   string
	Kind          string
	Removed       bool
	State         string
	SettledReason string
	MessageID     string
	Run           int
	Attempts      int
	OccurredAt    time.Time
	DueAt         time.Time
	SettledAt     *time.Time
	Last          *DeliveryAttempt
}

type DeliveryAttempt = outbox.Attempt

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
	rows, err := s.pool.Query(ctx, selectDeliveries+`
		 where event.post_id = $1
		 order by event.occurred_at desc, work.destination_name
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("read what a post sent: %w", err)
	}
	return collectDeliveries(rows)
}

func (s *Service) Deliveries(ctx context.Context, state string, limit int) ([]Delivery, error) {
	rows, err := s.pool.Query(ctx, selectDeliveries+`
		 where $1 = '' or work.state = $1
		 order by work.updated_at desc
		 limit $2
	`, state, limit)
	if err != nil {
		return nil, fmt.Errorf("read publication deliveries: %w", err)
	}
	return collectDeliveries(rows)
}

func (s *Service) Delivery(ctx context.Context, id uuid.UUID) (Delivery, error) {
	rows, err := s.pool.Query(ctx, selectDeliveries+` where work.id = $1`, id)
	if err != nil {
		return Delivery{}, fmt.Errorf("read a publication delivery: %w", err)
	}
	found, err := collectDeliveries(rows)
	if err != nil {
		return Delivery{}, err
	}
	if len(found) == 0 {
		return Delivery{}, ErrDeliveryNotFound
	}
	return found[0], nil
}

func (s *Service) DeliveryHistory(ctx context.Context, id uuid.UUID) ([]DeliveryAttempt, error) {
	if _, err := s.Delivery(ctx, id); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select run, number, outcome, status, detail, took_ms, attempted_at
		  from publication_delivery_attempts
		 where delivery_id = $1
		 order by number
	`, id)
	if err != nil {
		return nil, fmt.Errorf("read the attempts a delivery made: %w", err)
	}
	defer rows.Close()
	made := make([]DeliveryAttempt, 0, DeliveryAttempts)
	for rows.Next() {
		var one DeliveryAttempt
		var took int
		err := rows.Scan(
			&one.Run, &one.Number, &one.Outcome, &one.Status, &one.Detail, &took, &one.Attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one attempt a delivery made: %w", err)
		}
		one.Took = time.Duration(took) * time.Millisecond
		made = append(made, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the attempts a delivery made: %w", err)
	}
	return made, nil
}

func (s *Service) ReplayDelivery(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Delivery, error) {
	held, err := s.Delivery(ctx, id)
	if err != nil {
		return Delivery{}, err
	}
	if held.SettledAt == nil {
		return Delivery{}, ErrDeliveryUnsettled
	}
	if held.Removed {
		return Delivery{}, ErrDeliveryUnsendable
	}
	if held.Kind == KindDiscord && (held.State == DeliveryUnconfirmed || held.MessageID != "") {
		return Delivery{}, FieldError{Field: "delivery", Message: "Check the Discord message and use an explicit repair action."}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Delivery{}, fmt.Errorf("begin delivery replay: %w", err)
	}
	defer tx.Rollback(ctx)
	var state string
	err = tx.QueryRow(ctx, `
		select destination.state
		  from publication_deliveries work
		  join publication_destinations destination on destination.id = work.destination_id
		 where work.id = $1
		   for no key update of destination
	`, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return Delivery{}, ErrDeliveryUnsendable
	}
	if err != nil {
		return Delivery{}, fmt.Errorf("read whether a replay has anywhere to go: %w", err)
	}
	if state != DestinationActive {
		return Delivery{}, ErrDeliveryUnsendable
	}
	if err := s.ledger.Requeue(ctx, tx, id, s.now()); err != nil {
		return Delivery{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "delivery.replayed", DeliveryID: &id, PostID: &held.PostID,
	})
	if err != nil {
		return Delivery{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Delivery{}, fmt.Errorf("commit delivery replay: %w", err)
	}
	return s.Delivery(ctx, id)
}

func (s *Service) queueDeliveries(
	ctx context.Context,
	tx pgx.Tx,
	postID uuid.UUID,
	eventID uuid.UUID,
	event string,
	chosen []sending,
) error {
	for _, one := range chosen {
		if one.Kind == KindDiscord && event == EventPublished {
			summary, err := s.summaryWith(ctx, tx, eventID)
			if err != nil {
				return err
			}
			role := ""
			if one.Ping {
				if err := tx.QueryRow(ctx, `select coalesce(role_id, '') from publication_destinations where id = $1`, one.ID).Scan(&role); err != nil {
					return err
				}
			}
			if _, err := announcementOf(summary, role).Body(); err != nil {
				return err
			}
		}
		_, err := tx.Exec(ctx, `
			insert into publication_deliveries
			       (id, event_id, destination_id, destination_name, mention_role)
			select $1, $2, destination.id, $4, $6
			  from publication_destinations destination
			 where destination.id = $3 and $5 = any (destination.events)
			   and (destination.kind <> $7 or not exists (
			         select 1 from publication_deliveries already
			           join publication_events sent on sent.id = already.event_id
			          where sent.post_id = $8 and already.destination_id = destination.id))
			on conflict do nothing
		`, uuid.New(), eventID, one.ID, one.Name, event, one.Ping, KindDiscord, postID)
		if err != nil {
			return fmt.Errorf("keep the delivery work: %w", err)
		}
	}
	return nil
}

func (s *Service) stopDeliveriesTo(ctx context.Context, tx pgx.Tx, id uuid.UUID, reason string) error {
	return s.ledger.StopTo(ctx, tx, id, outbox.Stopped(reason))
}

func (s *Service) RunDeliveries(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(DeliveryPoll)
	defer ticker.Stop()
	for {
		_, err := s.SendDueDeliveries(ctx, s.now())
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

func (s *Service) SendDueDeliveries(ctx context.Context, now time.Time) (int, error) {
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
	outbox.Work
	EventType string
	Ping      bool
}

func (s *Service) sendLeased(ctx context.Context, work outbox.Work, now time.Time) error {
	held := carried{Work: work}
	err := s.pool.QueryRow(ctx, `
		select event.type, work.mention_role
		  from publication_deliveries work
		  join publication_events event on event.id = work.event_id
		 where work.id = $1
	`, work.ID).Scan(&held.EventType, &held.Ping)
	if err != nil {
		return fmt.Errorf("read what a leased delivery carries: %w", err)
	}
	if held.DestinationID == nil {
		return s.ledger.Record(ctx, held.Work, outbox.Stopped(SettledRemoved), now)
	}
	kind, state, err := s.destinationStanding(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	if state != DestinationActive {
		return s.ledger.Record(ctx, held.Work, outbox.Stopped(SettledDisabled), now)
	}
	if kind == KindDiscord {
		return s.announceOnDiscord(ctx, held, *held.DestinationID, now)
	}
	body, err := s.eventBody(ctx, held.EventID)
	if err != nil {
		return err
	}
	address, secrets, err := s.endpointOf(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	headers, err := webhook.Headers(secrets, held.ID.String(), now.UTC(), body)
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, address, headers, body)
	if err != nil {
		return s.ledger.Record(ctx, held.Work, outbox.Unreachable, now)
	}
	said := outbox.ReadAnswer(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if err := s.ledger.Record(ctx, held.Work, said, now); err != nil {
		return err
	}
	if !said.Gone {
		return nil
	}
	return s.retireDestination(ctx, *held.DestinationID)
}

func (s *Service) destinationStanding(ctx context.Context, id uuid.UUID) (string, string, error) {
	var kind, state string
	err := s.pool.QueryRow(ctx, `
		select kind, state from publication_destinations where id = $1
	`, id).Scan(&kind, &state)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", nil
	}
	if err != nil {
		return "", "", fmt.Errorf("read whether a destination still receives: %w", err)
	}
	return kind, state, nil
}

const selectDeliveries = `
	select work.id, event.id, event.type, event.post_id, revision.title, event.revision_id,
	       work.destination_name, coalesce(destination.kind, ''),
	       work.destination_id is null, work.state,
	       coalesce(work.settled_reason, ''), coalesce(work.message_id, ''),
	       work.run, work.attempts,
	       event.occurred_at, work.due_at, work.settled_at,
	       last.run, last.number, last.outcome, last.status, last.detail,
	       last.took_ms, last.attempted_at
	  from publication_deliveries work
	  join publication_events event on event.id = work.event_id
	  join post_revisions revision on revision.id = event.revision_id
	  left join publication_destinations destination on destination.id = work.destination_id
	  left join lateral (
		select run, number, outcome, status, detail, took_ms, attempted_at
		  from publication_delivery_attempts
		 where delivery_id = work.id
		 order by number desc
		 limit 1
	  ) last on true
	`

func collectDeliveries(rows pgx.Rows) ([]Delivery, error) {
	defer rows.Close()
	found := make([]Delivery, 0, 4)
	for rows.Next() {
		var one Delivery
		var run, number, status *int
		var outcome, detail *string
		var took *int
		var attempted *time.Time
		err := rows.Scan(
			&one.ID, &one.EventID, &one.EventType, &one.PostID, &one.PostTitle, &one.RevisionID,
			&one.Destination, &one.Kind, &one.Removed, &one.State,
			&one.SettledReason, &one.MessageID, &one.Run, &one.Attempts,
			&one.OccurredAt, &one.DueAt, &one.SettledAt,
			&run, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing a post sent: %w", err)
		}
		if number != nil {
			one.Last = &DeliveryAttempt{
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
