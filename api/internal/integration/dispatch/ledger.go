package dispatch

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const Poll = 5 * time.Second

const Lease = time.Minute

var identifier = regexp.MustCompile(`^[a-z_]+$`)

// Tables names the announcement and attempt tables
type Tables struct {
	Deliveries string
	Attempts   string
}

// Ledger leases, defers and settles delivery rows in one pair of tables.
type Ledger struct {
	pool   *pgxpool.Pool
	tables Tables
}

func NewLedger(pool *pgxpool.Pool, tables Tables) Ledger {
	if !identifier.MatchString(tables.Deliveries) || !identifier.MatchString(tables.Attempts) {
		panic("outbox tables are plain lower-case identifiers")
	}
	return Ledger{pool: pool, tables: tables}
}

type Work struct {
	ID            uuid.UUID
	EventID       uuid.UUID
	DestinationID *uuid.UUID
	Token         uuid.UUID
	Run           int
	Attempts      int
	Made          int
	Reclaimed     bool
}

type Attempt struct {
	Run       int
	Number    int
	Outcome   string
	Status    *int
	Detail    string
	Took      time.Duration
	Attempted time.Time
}

// Lease takes one due delivery, counting the attempt it is about to make.
func (l Ledger) Lease(ctx context.Context, now time.Time) (Work, bool, error) {
	var held Work
	held.Token = uuid.New()
	err := l.pool.QueryRow(ctx, fmt.Sprintf(`
		with candidate as (
			select id, state = $5 as reclaimed
			  from %[1]s
			 where due_at <= $1
			   and (state = $4 or (state = $5 and lease_expires_at <= $1))
			 order by due_at
			 for update skip locked
			 limit 1
		)
		update %[1]s work
		   set state = $5, attempts = attempts + 1,
		       lease_token = $2, lease_expires_at = $3, updated_at = $1
		  from candidate
		 where work.id = candidate.id
		returning work.id, work.event_id, work.destination_id, work.run, work.attempts,
		          candidate.reclaimed, (select count(*) from %[2]s made
		            where made.delivery_id = work.id and made.run = work.run)
	`, l.tables.Deliveries, l.tables.Attempts),
		now, held.Token, now.Add(Lease), Pending, Sending,
	).Scan(&held.ID, &held.EventID, &held.DestinationID, &held.Run, &held.Attempts, &held.Reclaimed, &held.Made)
	if errors.Is(err, pgx.ErrNoRows) {
		return Work{}, false, nil
	}
	if err != nil {
		return Work{}, false, fmt.Errorf("lease a due delivery: %w", err)
	}
	return held, true, nil
}

// Record writes the attempt and either leaves the delivery due again or settles it.
func (l Ledger) Record(ctx context.Context, held Work, said Verdict, now time.Time) error {
	made := Attempt{
		Run: held.Run, Number: held.Attempts, Outcome: said.Outcome,
		Status: said.Status, Detail: said.Detail, Took: said.Took, Attempted: now.UTC(),
	}
	delay, again := Delay(held.Made+1, Spread())
	if said.Retry && again {
		return l.deferAttempt(ctx, held, made, now.Add(max(delay, said.After)))
	}
	reason := said.Reason
	state := Failed
	switch {
	case said.Outcome == OutcomeDelivered:
		state = Delivered
	case said.Outcome == OutcomeUnconfirmed:
		state = Unconfirmed
	case reason == "":
		reason = Exhausted
	}
	return l.settle(ctx, held, state, reason, made)
}

func (l Ledger) deferAttempt(ctx context.Context, held Work, made Attempt, due time.Time) error {
	return l.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			update %s
			   set state = $2, due_at = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $3
			 where id = $1
		`, l.tables.Deliveries), held.ID, Pending, due.UTC())
		if err != nil {
			return fmt.Errorf("leave the delivery due again: %w", err)
		}
		return nil
	})
}

func (l Ledger) settle(ctx context.Context, held Work, state, reason string, made Attempt) error {
	return l.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			update %s
			   set state = $2, settled_at = $4, settled_reason = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $4
			 where id = $1
		`, l.tables.Deliveries), held.ID, state, reason, made.Attempted)
		if err != nil {
			return fmt.Errorf("settle the delivery: %w", err)
		}
		return nil
	})
}

func (l Ledger) close(ctx context.Context, held Work, made Attempt, change func(pgx.Tx) error) error {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the delivery record: %w", err)
	}
	defer tx.Rollback(ctx)
	still, err := l.heldBy(ctx, tx, held)
	if err != nil || !still {
		return err
	}
	if err := change(tx); err != nil {
		return err
	}
	if err := l.recordAttempt(ctx, tx, held.ID, made); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit the delivery record: %w", err)
	}
	return nil
}

func (l Ledger) heldBy(ctx context.Context, tx pgx.Tx, held Work) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, fmt.Sprintf(`
		select true from %s
		 where id = $1 and lease_token = $2 and state = $3
		   for update
	`, l.tables.Deliveries), held.ID, held.Token, Sending).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the delivery lease: %w", err)
	}
	return found, nil
}

func (l Ledger) recordAttempt(ctx context.Context, tx pgx.Tx, deliveryID uuid.UUID, made Attempt) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`
		insert into %s
		       (id, delivery_id, run, number, outcome, status, detail, took_ms, attempted_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		on conflict (delivery_id, number) do nothing
	`, l.tables.Attempts), uuid.New(), deliveryID, made.Run, made.Number, made.Outcome,
		made.Status, made.Detail, made.Took.Milliseconds(), made.Attempted)
	if err != nil {
		return fmt.Errorf("record the delivery attempt: %w", err)
	}
	return nil
}

// KeepMessage stores the Discord message an announcement made.
func (l Ledger) KeepMessage(ctx context.Context, deliveryID uuid.UUID, message string) error {
	_, err := l.pool.Exec(ctx, fmt.Sprintf(`
		update %s set message_id = $2 where id = $1
	`, l.tables.Deliveries), deliveryID, message)
	if err != nil {
		return fmt.Errorf("keep the announcement's message id: %w", err)
	}
	return nil
}

// StopTo settles every unfinished delivery to one destination with the given verdict.
func (l Ledger) StopTo(ctx context.Context, tx pgx.Tx, destinationID uuid.UUID, said Verdict) error {
	now := time.Now().UTC()
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		update %s
		   set state = $2, settled_at = $6, settled_reason = $5, attempts = attempts + 1,
		       lease_token = null, lease_expires_at = null, updated_at = $6
		 where destination_id = $1 and state in ($3, $4)
		returning id, run, attempts
	`, l.tables.Deliveries), destinationID, Failed, Pending, Sending, said.Reason, now)
	if err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	halted := make([]Attempt, 0, 4)
	held := make([]uuid.UUID, 0, 4)
	for rows.Next() {
		var deliveryID uuid.UUID
		var run, number int
		if err := rows.Scan(&deliveryID, &run, &number); err != nil {
			rows.Close()
			return fmt.Errorf("stop one piece of work waiting on a destination: %w", err)
		}
		held = append(held, deliveryID)
		halted = append(halted, Attempt{
			Run: run, Number: number, Outcome: said.Outcome, Detail: said.Detail, Attempted: now,
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("stop the work waiting on a destination: %w", err)
	}
	for index, deliveryID := range held {
		if err := l.recordAttempt(ctx, tx, deliveryID, halted[index]); err != nil {
			return err
		}
	}
	return nil
}

// Requeue opens a fresh attempt sequence for one settled delivery.
func (l Ledger) Requeue(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`
		update %s
		   set state = $2, run = run + 1, due_at = $3,
		       settled_at = null, settled_reason = null,
		       lease_token = null, lease_expires_at = null, updated_at = $3
		 where id = $1
	`, l.tables.Deliveries), id, Pending, now.UTC())
	if err != nil {
		return fmt.Errorf("put the delivery back in the queue: %w", err)
	}
	return nil
}
