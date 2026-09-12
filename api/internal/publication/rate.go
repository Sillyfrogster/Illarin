package publication

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const OperationRead = "read"

const OperationWrite = "write"

const OperationUpload = "upload"

type Rate struct {
	Attempts int
	Window   time.Duration
}

type Rates struct {
	Read   Rate
	Write  Rate
	Upload Rate
}

func DefaultRates() Rates {
	return Rates{
		Read:   Rate{Attempts: 300, Window: time.Minute},
		Write:  Rate{Attempts: 60, Window: time.Minute},
		Upload: Rate{Attempts: 20, Window: time.Minute},
	}
}

type TooManyRequests struct {
	Operation string
	After     time.Duration
}

func (e TooManyRequests) Error() string { return "the publication token is going too fast" }

func (s *Service) Take(ctx context.Context, tokenID uuid.UUID, operation string) error {
	limit := s.rates.of(operation)
	var attempts int
	var windowStart time.Time
	err := s.pool.QueryRow(ctx, `
		insert into publication_rate_limits as rate
		       (token_id, operation, attempts, window_start)
		values ($1, $2, 1, now())
		on conflict (token_id, operation) do update
		   set attempts = case
		           when rate.window_start > $3 then rate.attempts + 1
		           else 1
		       end,
		       window_start = case
		           when rate.window_start > $3 then rate.window_start
		           else now()
		       end
		returning rate.attempts, rate.window_start
	`, tokenID, operation, time.Now().Add(-limit.Window)).Scan(&attempts, &windowStart)
	if err != nil {
		return fmt.Errorf("pace a publication token: %w", err)
	}
	if attempts <= limit.Attempts {
		return nil
	}
	after := time.Until(windowStart.Add(limit.Window))
	if after < time.Second {
		after = time.Second
	}
	return TooManyRequests{Operation: operation, After: after}
}

func (r Rates) of(operation string) Rate {
	switch operation {
	case OperationUpload:
		return r.Upload
	case OperationWrite:
		return r.Write
	default:
		return r.Read
	}
}
