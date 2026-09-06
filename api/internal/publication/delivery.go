package publication

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// The states a delivery passes through. Neither settled state changes whether
// the post is public.
const (
	DeliveryPending   = "pending"
	DeliverySending   = "sending"
	DeliveryDelivered = "delivered"
	DeliveryFailed    = "failed"
)

// What one attempt found at the far end.
const (
	AttemptDelivered   = "delivered"
	AttemptRefused     = "refused"
	AttemptUnreachable = "unreachable"
)

// DeliveryPoll is how often the worker looks for delivery work.
const DeliveryPoll = 5 * time.Second

// DeliveryLease is how long one attempt holds a delivery before another
// attempt may take it over.
const DeliveryLease = time.Minute

// The reasons Illarin settles a delivery without reaching anything.
const (
	stoppedByRemoval   = "The destination was removed."
	stoppedByDisabling = "The destination was disabled."
	stoppedByMoving    = "The destination moved to another address."
)

// Sender is how a publication destination is checked and reached, under
// Illarin's outbound address policy.
type Sender interface {
	Check(address string) (string, error)
	Post(
		ctx context.Context,
		address string,
		headers map[string]string,
		body []byte,
	) (outbound.Answer, error)
}

// Delivery is what a contributor and an admin are told about one event on its
// way to one destination.
type Delivery struct {
	ID          uuid.UUID
	EventID     uuid.UUID
	EventType   string
	PostID      uuid.UUID
	RevisionID  uuid.UUID
	Destination string
	State       string
	Attempts    int
	OccurredAt  time.Time
	SettledAt   *time.Time
	Last        *DeliveryAttempt
}

// DeliveryAttempt is the safe record of one request. It holds no body, in
// either direction, and no header Illarin signed it with.
type DeliveryAttempt struct {
	Number    int
	Outcome   string
	Status    *int
	Detail    string
	Took      time.Duration
	Attempted time.Time
}

// PostDeliveries answers what one post's public transitions have produced, to
// the contributor who owns it and to an admin.
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
	rows, err := s.pool.Query(ctx, `
		select work.id, event.id, event.type, event.post_id, event.revision_id,
		       work.destination_name, work.state, work.attempts, event.occurred_at,
		       work.settled_at, last.number, last.outcome, last.status, last.detail,
		       last.took_ms, last.attempted_at
		  from publication_deliveries work
		  join publication_events event on event.id = work.event_id
		  left join lateral (
			select number, outcome, status, detail, took_ms, attempted_at
			  from publication_delivery_attempts
			 where delivery_id = work.id
			 order by number desc
			 limit 1
		  ) last on true
		 where event.post_id = $1
		 order by event.occurred_at desc, work.destination_name
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("read what a post sent: %w", err)
	}
	defer rows.Close()
	found := make([]Delivery, 0, 4)
	for rows.Next() {
		var one Delivery
		var number, status *int
		var outcome, detail *string
		var took *int
		var attempted *time.Time
		err := rows.Scan(
			&one.ID, &one.EventID, &one.EventType, &one.PostID, &one.RevisionID,
			&one.Destination, &one.State, &one.Attempts, &one.OccurredAt,
			&one.SettledAt, &number, &outcome, &status, &detail, &took, &attempted,
		)
		if err != nil {
			return nil, fmt.Errorf("read one thing a post sent: %w", err)
		}
		if number != nil {
			one.Last = &DeliveryAttempt{
				Number: *number, Outcome: *outcome, Status: status, Detail: *detail,
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

// queueDeliveries turns one transition's captured choice into durable work. It
// runs inside the transaction that makes the post public and makes no request.
func queueDeliveries(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
	chosen []Choice,
) error {
	for _, one := range chosen {
		_, err := tx.Exec(ctx, `
			insert into publication_deliveries (id, event_id, destination_id, destination_name)
			values ($1, $2, $3, $4)
			on conflict do nothing
		`, uuid.New(), eventID, one.ID, one.Name)
		if err != nil {
			return fmt.Errorf("keep the delivery work: %w", err)
		}
	}
	return nil
}

// stopDeliveriesTo settles the work waiting on a destination that is no longer
// somewhere Illarin will send, and says so in one safe attempt record.
func stopDeliveriesTo(ctx context.Context, tx pgx.Tx, id uuid.UUID, because string) error {
	rows, err := tx.Query(ctx, `
		update publication_deliveries
		   set state = $2, settled_at = now(), attempts = attempts + 1,
		       lease_token = null, lease_expires_at = null, updated_at = now()
		 where destination_id = $1 and state in ($3, $4)
		returning id, attempts
	`, id, DeliveryFailed, DeliveryPending, DeliverySending)
	if err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	stopped := make(map[uuid.UUID]int)
	for rows.Next() {
		var deliveryID uuid.UUID
		var number int
		if err := rows.Scan(&deliveryID, &number); err != nil {
			rows.Close()
			return fmt.Errorf("stop one piece of work waiting on a destination: %w", err)
		}
		stopped[deliveryID] = number
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	for deliveryID, number := range stopped {
		err := recordAttempt(ctx, tx, deliveryID, DeliveryAttempt{
			Number: number, Outcome: AttemptRefused, Detail: because,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// RunDeliveries sends queued publication events until the context is done.
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

// SendDueDeliveries sends every delivery due at the given instant and answers
// how many it settled.
func (s *Service) SendDueDeliveries(ctx context.Context, now time.Time) (int, error) {
	settled := 0
	for {
		held, taken, err := s.leaseDueDelivery(ctx, now)
		if err != nil || !taken {
			return settled, err
		}
		if err := s.sendLeased(ctx, held, now); err != nil {
			return settled, err
		}
		settled++
	}
}

// waiting is one due delivery this worker holds and what it has to send.
type waiting struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	DestinationID *uuid.UUID
	Token         uuid.UUID
	Attempts      int
}

// leaseDueDelivery takes one due delivery, including one an earlier attempt
// stopped holding, and answers false when nothing is due.
func (s *Service) leaseDueDelivery(ctx context.Context, now time.Time) (waiting, bool, error) {
	var held waiting
	held.Token = uuid.New()
	err := s.pool.QueryRow(ctx, `
		with candidate as (
			select id
			  from publication_deliveries
			 where due_at <= $1
			   and (state = $4 or (state = $5 and lease_expires_at <= $1))
			 order by due_at
			 for update skip locked
			 limit 1
		)
		update publication_deliveries work
		   set state = $5, attempts = attempts + 1,
		       lease_token = $2, lease_expires_at = $3, updated_at = $1
		  from candidate
		 where work.id = candidate.id
		returning work.id, work.event_id, work.destination_id, work.attempts
	`, now, held.Token, now.Add(DeliveryLease), DeliveryPending, DeliverySending).Scan(
		&held.ID, &held.EventID, &held.DestinationID, &held.Attempts,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return waiting{}, false, nil
	}
	if err != nil {
		return waiting{}, false, fmt.Errorf("lease a due delivery: %w", err)
	}
	return held, true, nil
}

// sendLeased makes the one request a leased delivery names and settles it.
func (s *Service) sendLeased(ctx context.Context, held waiting, now time.Time) error {
	if held.DestinationID == nil {
		return s.settle(ctx, held, DeliveryFailed, DeliveryAttempt{
			Number: held.Attempts, Outcome: AttemptRefused, Detail: stoppedByRemoval,
		})
	}
	state, err := s.destinationState(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	if state != DestinationActive {
		return s.settle(ctx, held, DeliveryFailed, DeliveryAttempt{
			Number: held.Attempts, Outcome: AttemptRefused, Detail: stoppedByDisabling,
		})
	}
	body, err := s.eventBody(ctx, held.EventID)
	if err != nil {
		return err
	}
	address, secret, err := s.endpointOf(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	headers, err := webhook.Headers(secret, held.ID.String(), now.UTC(), body)
	if err != nil {
		return err
	}
	answer, err := s.sender.Post(ctx, address, headers, body)
	if err != nil {
		return s.settle(ctx, held, DeliveryFailed, DeliveryAttempt{
			Number: held.Attempts, Outcome: AttemptUnreachable, Detail: "Illarin could not reach it.",
		})
	}
	status := answer.Status
	attempt := DeliveryAttempt{
		Number: held.Attempts, Status: &status, Took: answer.Took,
		Outcome: AttemptRefused, Detail: fmt.Sprintf("It answered %d.", status),
	}
	state = DeliveryFailed
	if status >= http.StatusOK && status < http.StatusMultipleChoices {
		attempt.Outcome, attempt.Detail, state = AttemptDelivered, "", DeliveryDelivered
	}
	return s.settle(ctx, held, state, attempt)
}

// settle closes one delivery and keeps the safe record of the attempt that
// closed it, whether or not that attempt reached anything.
func (s *Service) settle(
	ctx context.Context,
	held waiting,
	state string,
	attempt DeliveryAttempt,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delivery settlement: %w", err)
	}
	defer tx.Rollback(ctx)
	still, err := deliveryHeldByThisAttempt(ctx, tx, held)
	if err != nil || !still {
		return err
	}
	_, err = tx.Exec(ctx, `
		update publication_deliveries
		   set state = $2, settled_at = now(),
		       lease_token = null, lease_expires_at = null, updated_at = now()
		 where id = $1
	`, held.ID, state)
	if err != nil {
		return fmt.Errorf("settle the delivery: %w", err)
	}
	if err := recordAttempt(ctx, tx, held.ID, attempt); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delivery settlement: %w", err)
	}
	return nil
}

func recordAttempt(ctx context.Context, tx pgx.Tx, deliveryID uuid.UUID, made DeliveryAttempt) error {
	_, err := tx.Exec(ctx, `
		insert into publication_delivery_attempts
		       (id, delivery_id, number, outcome, status, detail, took_ms)
		values ($1, $2, $3, $4, $5, $6, $7)
		on conflict (delivery_id, number) do nothing
	`, uuid.New(), deliveryID, made.Number, made.Outcome, made.Status, made.Detail,
		made.Took.Milliseconds())
	if err != nil {
		return fmt.Errorf("record the delivery attempt: %w", err)
	}
	return nil
}

// deliveryHeldByThisAttempt answers whether the lease this attempt took is
// still the one on the row, so work someone else took over is left alone.
func deliveryHeldByThisAttempt(ctx context.Context, tx pgx.Tx, held waiting) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, `
		select true from publication_deliveries
		 where id = $1 and lease_token = $2 and state = $3
		   for update
	`, held.ID, held.Token, DeliverySending).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the delivery lease: %w", err)
	}
	return found, nil
}

func (s *Service) destinationState(ctx context.Context, id uuid.UUID) (string, error) {
	var state string
	err := s.pool.QueryRow(ctx, `
		select state from publication_destinations where id = $1
	`, id).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read whether a destination still receives: %w", err)
	}
	return state, nil
}
