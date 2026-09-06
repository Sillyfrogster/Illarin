package publication

import (
	"context"
	"errors"
	"fmt"
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

// ErrDeliveryNotFound says no such delivery is on record.
var ErrDeliveryNotFound = errors.New("no such publication delivery")

// ErrDeliveryUnsettled says a delivery still has attempts of its own to make.
var ErrDeliveryUnsettled = errors.New("the delivery has not finished trying")

// ErrDeliveryUnsendable says there is no longer anywhere to send a delivery.
var ErrDeliveryUnsendable = errors.New("the destination no longer receives")

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
	ID            uuid.UUID
	EventID       uuid.UUID
	EventType     string
	PostID        uuid.UUID
	PostTitle     string
	RevisionID    uuid.UUID
	Destination   string
	Removed       bool
	State         string
	SettledReason string
	Run           int
	Attempts      int
	OccurredAt    time.Time
	DueAt         time.Time
	SettledAt     *time.Time
	Last          *DeliveryAttempt
}

// DeliveryAttempt is the safe record of one request. It holds no body, in
// either direction, and no header Illarin signed it with.
type DeliveryAttempt struct {
	Run       int
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
	rows, err := s.pool.Query(ctx, selectDeliveries+`
		 where event.post_id = $1
		 order by event.occurred_at desc, work.destination_name
	`, postID)
	if err != nil {
		return nil, fmt.Errorf("read what a post sent: %w", err)
	}
	return collectDeliveries(rows)
}

// Deliveries answers the delivery work the authority is diagnosing, newest
// first, narrowed to one state where it asked for one.
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

// Delivery answers one piece of delivery work.
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

// DeliveryHistory answers every attempt one delivery has made, oldest first,
// so a replay never hides what the runs before it found.
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

// ReplayDelivery puts settled work back in the queue under a new run. The
// Publication event it carries and every attempt already made stay as they are.
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
	_, err = tx.Exec(ctx, `
		update publication_deliveries
		   set state = $2, run = run + 1, due_at = $3,
		       settled_at = null, settled_reason = null,
		       lease_token = null, lease_expires_at = null, updated_at = $3
		 where id = $1
	`, id, DeliveryPending, s.now().UTC())
	if err != nil {
		return Delivery{}, fmt.Errorf("put the delivery back in the queue: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "delivery.replayed", DeliveryID: &id, PostID: &held.PostID,
		Before: held.State, After: DeliveryPending,
	})
	if err != nil {
		return Delivery{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Delivery{}, fmt.Errorf("commit delivery replay: %w", err)
	}
	return s.Delivery(ctx, id)
}

// queueDeliveries turns one transition's captured choice into durable work,
// skipping a destination that did not subscribe to this event. It runs inside
// the transaction that changes the post and makes no request.
func queueDeliveries(
	ctx context.Context,
	tx pgx.Tx,
	eventID uuid.UUID,
	event string,
	chosen []Choice,
) error {
	for _, one := range chosen {
		_, err := tx.Exec(ctx, `
			insert into publication_deliveries
			       (id, event_id, destination_id, destination_name)
			select $1, $2, destination.id, $4
			  from publication_destinations destination
			 where destination.id = $3 and $5 = any (destination.events)
			on conflict do nothing
		`, uuid.New(), eventID, one.ID, one.Name, event)
		if err != nil {
			return fmt.Errorf("keep the delivery work: %w", err)
		}
	}
	return nil
}

// stopDeliveriesTo settles the work waiting on a destination that is no longer
// somewhere Illarin will send, and says so in one safe attempt record.
func stopDeliveriesTo(ctx context.Context, tx pgx.Tx, id uuid.UUID, reason string) error {
	said := stopped(reason)
	rows, err := tx.Query(ctx, `
		update publication_deliveries
		   set state = $2, settled_at = now(), settled_reason = $5, attempts = attempts + 1,
		       lease_token = null, lease_expires_at = null, updated_at = now()
		 where destination_id = $1 and state in ($3, $4)
		returning id, run, attempts
	`, id, DeliveryFailed, DeliveryPending, DeliverySending, reason)
	if err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	halted := make([]DeliveryAttempt, 0, 4)
	held := make([]uuid.UUID, 0, 4)
	for rows.Next() {
		var deliveryID uuid.UUID
		var run, number int
		if err := rows.Scan(&deliveryID, &run, &number); err != nil {
			rows.Close()
			return fmt.Errorf("stop one piece of work waiting on a destination: %w", err)
		}
		held = append(held, deliveryID)
		halted = append(halted, DeliveryAttempt{
			Run: run, Number: number, Outcome: said.Outcome, Detail: said.Detail,
			Attempted: time.Now().UTC(),
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	for index, deliveryID := range held {
		if err := recordAttempt(ctx, tx, deliveryID, halted[index]); err != nil {
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

// SendDueDeliveries attempts every delivery due at the given instant and
// answers how many attempts it made.
func (s *Service) SendDueDeliveries(ctx context.Context, now time.Time) (int, error) {
	made := 0
	for {
		held, taken, err := s.leaseDueDelivery(ctx, now)
		if err != nil || !taken {
			return made, err
		}
		if err := s.sendLeased(ctx, held, now); err != nil {
			return made, err
		}
		made++
	}
}

// waiting is one due delivery this worker holds and what it has to send.
type waiting struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	EventType     string
	DestinationID *uuid.UUID
	Token         uuid.UUID
	Run           int
	Attempts      int
	Made          int
}

// leaseDueDelivery takes one due delivery, including one an earlier attempt
// stopped holding, and answers false when nothing is due. An attempt that left
// no record behind does not spend a place in the run, which is what makes an
// interrupted process cost a receiver nothing.
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
		  from candidate, publication_events event
		 where work.id = candidate.id and event.id = work.event_id
		returning work.id, work.event_id, event.type, work.destination_id, work.run,
		          work.attempts,
		          (select count(*) from publication_delivery_attempts made
		            where made.delivery_id = work.id and made.run = work.run)
	`, now, held.Token, now.Add(DeliveryLease), DeliveryPending, DeliverySending).Scan(
		&held.ID, &held.EventID, &held.EventType, &held.DestinationID,
		&held.Run, &held.Attempts, &held.Made,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return waiting{}, false, nil
	}
	if err != nil {
		return waiting{}, false, fmt.Errorf("lease a due delivery: %w", err)
	}
	return held, true, nil
}

// sendLeased makes the one request a leased delivery names and records what
// came of it, whether that ends the work or leaves it due again.
func (s *Service) sendLeased(ctx context.Context, held waiting, now time.Time) error {
	if held.DestinationID == nil {
		return s.record(ctx, held, stopped(SettledRemoved), now)
	}
	state, err := s.destinationState(ctx, *held.DestinationID)
	if err != nil {
		return err
	}
	if state != DestinationActive {
		return s.record(ctx, held, stopped(SettledDisabled), now)
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
		return s.record(ctx, held, unreachable, now)
	}
	said := readAnswer(answer)
	said.Status, said.Took = &answer.Status, answer.Took
	if err := s.record(ctx, held, said, now); err != nil {
		return err
	}
	if !said.Gone {
		return nil
	}
	return s.retireDestination(ctx, *held.DestinationID)
}

// record keeps one attempt and leaves the delivery either due again or settled,
// which is the whole of what a bounded at-least-once schedule decides.
func (s *Service) record(
	ctx context.Context,
	held waiting,
	said verdict,
	now time.Time,
) error {
	made := DeliveryAttempt{
		Run: held.Run, Number: held.Attempts, Outcome: said.Outcome,
		Status: said.Status, Detail: said.Detail, Took: said.Took, Attempted: now.UTC(),
	}
	delay, again := deliveryDelay(held.Made+1, spread())
	if said.Retry && again {
		return s.deferAttempt(ctx, held, made, now.Add(max(delay, said.After)))
	}
	reason := said.Reason
	state := DeliveryFailed
	switch {
	case said.Outcome == AttemptDelivered:
		state = DeliveryDelivered
	case reason == "":
		reason = SettledExhausted
	}
	return s.settle(ctx, held, state, reason, made)
}

// deferAttempt leaves a delivery waiting for the next place in its run.
func (s *Service) deferAttempt(
	ctx context.Context,
	held waiting,
	made DeliveryAttempt,
	due time.Time,
) error {
	return s.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			update publication_deliveries
			   set state = $2, due_at = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $3
			 where id = $1
		`, held.ID, DeliveryPending, due.UTC())
		if err != nil {
			return fmt.Errorf("leave the delivery due again: %w", err)
		}
		return nil
	})
}

// settle closes one delivery for good and says why it closed.
func (s *Service) settle(
	ctx context.Context,
	held waiting,
	state, reason string,
	made DeliveryAttempt,
) error {
	return s.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			update publication_deliveries
			   set state = $2, settled_at = $4, settled_reason = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $4
			 where id = $1
		`, held.ID, state, reason, made.Attempted)
		if err != nil {
			return fmt.Errorf("settle the delivery: %w", err)
		}
		return nil
	})
}

// close writes one attempt and whatever it did to the delivery in the same
// transaction, and leaves work another attempt has taken over alone.
func (s *Service) close(
	ctx context.Context,
	held waiting,
	made DeliveryAttempt,
	change func(pgx.Tx) error,
) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the delivery record: %w", err)
	}
	defer tx.Rollback(ctx)
	still, err := deliveryHeldByThisAttempt(ctx, tx, held)
	if err != nil || !still {
		return err
	}
	if err := change(tx); err != nil {
		return err
	}
	if err := recordAttempt(ctx, tx, held.ID, made); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit the delivery record: %w", err)
	}
	return nil
}

func recordAttempt(ctx context.Context, tx pgx.Tx, deliveryID uuid.UUID, made DeliveryAttempt) error {
	_, err := tx.Exec(ctx, `
		insert into publication_delivery_attempts
		       (id, delivery_id, run, number, outcome, status, detail, took_ms, attempted_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		on conflict (delivery_id, number) do nothing
	`, uuid.New(), deliveryID, made.Run, made.Number, made.Outcome, made.Status, made.Detail,
		made.Took.Milliseconds(), made.Attempted)
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

const selectDeliveries = `
	select work.id, event.id, event.type, event.post_id, revision.title, event.revision_id,
	       work.destination_name, work.destination_id is null, work.state,
	       coalesce(work.settled_reason, ''), work.run, work.attempts,
	       event.occurred_at, work.due_at, work.settled_at,
	       last.run, last.number, last.outcome, last.status, last.detail,
	       last.took_ms, last.attempted_at
	  from publication_deliveries work
	  join publication_events event on event.id = work.event_id
	  join post_revisions revision on revision.id = event.revision_id
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
			&one.Destination, &one.Removed, &one.State,
			&one.SettledReason, &one.Run, &one.Attempts,
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
