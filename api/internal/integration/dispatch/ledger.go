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

// Tables names the attempt table and the table of tries under it
type Tables struct {
	Attempts string
	Tries    string
}

// Ledger leases, defers and settles attempt rows in one pair of tables
type Ledger struct {
	pool   *pgxpool.Pool
	tables Tables
}

func NewLedger(pool *pgxpool.Pool, tables Tables) Ledger {
	if !identifier.MatchString(tables.Attempts) || !identifier.MatchString(tables.Tries) {
		panic("attempt tables are plain lower-case identifiers")
	}
	return Ledger{pool: pool, tables: tables}
}

type Work struct {
	ID             uuid.UUID
	AnnouncementID uuid.UUID
	IntegrationID  *uuid.UUID
	Token          uuid.UUID
	Run            int
	Tries          int
	Made           int
	Reclaimed      bool
}

type Try struct {
	Run       int
	Number    int
	Outcome   string
	Status    *int
	Detail    string
	Took      time.Duration
	Attempted time.Time
}

// Lease takes one due attempt, counting the try it is about to make
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
		   set state = $5, tries = tries + 1,
		       lease_token = $2, lease_expires_at = $3, updated_at = $1
		  from candidate
		 where work.id = candidate.id
		returning work.id, work.announcement_id, work.integration_id, work.run, work.tries,
		          candidate.reclaimed, (select count(*) from %[2]s made
		            where made.attempt_id = work.id and made.run = work.run)
	`, l.tables.Attempts, l.tables.Tries),
		now, held.Token, now.Add(Lease), Pending, Sending,
	).Scan(&held.ID, &held.AnnouncementID, &held.IntegrationID, &held.Run, &held.Tries, &held.Reclaimed, &held.Made)
	if errors.Is(err, pgx.ErrNoRows) {
		return Work{}, false, nil
	}
	if err != nil {
		return Work{}, false, fmt.Errorf("lease a due attempt: %w", err)
	}
	return held, true, nil
}

// Record writes the try and either leaves the attempt due again or settles it
func (l Ledger) Record(ctx context.Context, held Work, said Verdict, now time.Time) error {
	made := Try{
		Run: held.Run, Number: held.Tries, Outcome: said.Outcome,
		Status: said.Status, Detail: said.Detail, Took: said.Took, Attempted: now.UTC(),
	}
	delay, again := Delay(held.Made+1, Spread())
	if said.Retry && again {
		return l.deferTry(ctx, held, made, now.Add(max(delay, said.After)))
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

func (l Ledger) deferTry(ctx context.Context, held Work, made Try, due time.Time) error {
	return l.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			update %s
			   set state = $2, due_at = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $3
			 where id = $1
		`, l.tables.Attempts), held.ID, Pending, due.UTC())
		if err != nil {
			return fmt.Errorf("leave the attempt due again: %w", err)
		}
		return nil
	})
}

func (l Ledger) settle(ctx context.Context, held Work, state, reason string, made Try) error {
	return l.close(ctx, held, made, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`
			update %s
			   set state = $2, settled_at = $4, settled_reason = $3,
			       lease_token = null, lease_expires_at = null, updated_at = $4
			 where id = $1
		`, l.tables.Attempts), held.ID, state, reason, made.Attempted)
		if err != nil {
			return fmt.Errorf("settle the attempt: %w", err)
		}
		return nil
	})
}

func (l Ledger) close(ctx context.Context, held Work, made Try, change func(pgx.Tx) error) error {
	tx, err := l.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin the attempt record: %w", err)
	}
	defer tx.Rollback(ctx)
	still, err := l.heldBy(ctx, tx, held)
	if err != nil || !still {
		return err
	}
	if err := change(tx); err != nil {
		return err
	}
	if err := l.recordTry(ctx, tx, held.ID, made); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit the attempt record: %w", err)
	}
	return nil
}

func (l Ledger) heldBy(ctx context.Context, tx pgx.Tx, held Work) (bool, error) {
	var found bool
	err := tx.QueryRow(ctx, fmt.Sprintf(`
		select true from %s
		 where id = $1 and lease_token = $2 and state = $3
		   for update
	`, l.tables.Attempts), held.ID, held.Token, Sending).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("check the attempt lease: %w", err)
	}
	return found, nil
}

func (l Ledger) recordTry(ctx context.Context, tx pgx.Tx, attemptID uuid.UUID, made Try) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`
		insert into %s
		       (id, attempt_id, run, number, outcome, status, detail, took_ms, attempted_at)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		on conflict (attempt_id, number) do nothing
	`, l.tables.Tries), uuid.New(), attemptID, made.Run, made.Number, made.Outcome,
		made.Status, made.Detail, made.Took.Milliseconds(), made.Attempted)
	if err != nil {
		return fmt.Errorf("record the try: %w", err)
	}
	return nil
}

// KeepMessage stores the Discord message an attempt made
func (l Ledger) KeepMessage(ctx context.Context, attemptID uuid.UUID, message string) error {
	_, err := l.pool.Exec(ctx, fmt.Sprintf(`
		update %s set message_id = $2 where id = $1
	`, l.tables.Attempts), attemptID, message)
	if err != nil {
		return fmt.Errorf("keep the attempt's message id: %w", err)
	}
	return nil
}

// StopTo settles every unfinished attempt to one integration with the given verdict
func (l Ledger) StopTo(ctx context.Context, tx pgx.Tx, integrationID uuid.UUID, said Verdict) error {
	now := time.Now().UTC()
	rows, err := tx.Query(ctx, fmt.Sprintf(`
		update %s
		   set state = $2, settled_at = $6, settled_reason = $5, tries = tries + 1,
		       lease_token = null, lease_expires_at = null, updated_at = $6
		 where integration_id = $1 and state in ($3, $4)
		returning id, run, tries
	`, l.tables.Attempts), integrationID, Failed, Pending, Sending, said.Reason, now)
	if err != nil {
		return fmt.Errorf("stop the attempts waiting on an integration: %w", err)
	}
	halted := make([]Try, 0, 4)
	held := make([]uuid.UUID, 0, 4)
	for rows.Next() {
		var attemptID uuid.UUID
		var run, number int
		if err := rows.Scan(&attemptID, &run, &number); err != nil {
			rows.Close()
			return fmt.Errorf("stop one attempt waiting on an integration: %w", err)
		}
		held = append(held, attemptID)
		halted = append(halted, Try{
			Run: run, Number: number, Outcome: said.Outcome, Detail: said.Detail, Attempted: now,
		})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("stop the attempts waiting on an integration: %w", err)
	}
	for index, attemptID := range held {
		if err := l.recordTry(ctx, tx, attemptID, halted[index]); err != nil {
			return err
		}
	}
	return nil
}

// Requeue opens a fresh run of tries for one settled attempt
func (l Ledger) Requeue(ctx context.Context, tx pgx.Tx, id uuid.UUID, now time.Time) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`
		update %s
		   set state = $2, run = run + 1, due_at = $3,
		       settled_at = null, settled_reason = null,
		       lease_token = null, lease_expires_at = null, updated_at = $3
		 where id = $1
	`, l.tables.Attempts), id, Pending, now.UTC())
	if err != nil {
		return fmt.Errorf("put the attempt back in the queue: %w", err)
	}
	return nil
}
