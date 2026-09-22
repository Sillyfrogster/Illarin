package integration

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	poll     = 5 * time.Second
	lease    = time.Minute
	maxTries = 8
)

// RunAnnouncements posts due announcements until the context ends
func (s *Service) RunAnnouncements(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	for {
		if _, err := s.SendDue(ctx); err != nil && ctx.Err() == nil && onError != nil {
			onError(err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// SendDue makes one try at every post that is due and reports how many it tried
func (s *Service) SendDue(ctx context.Context) (int, error) {
	tried := 0
	for {
		held, found, err := s.lease(ctx)
		if err != nil || !found {
			return tried, err
		}
		if err := s.send(ctx, held); err != nil {
			return tried, err
		}
		tried++
	}
}

type leased struct {
	id      uuid.UUID
	address []byte
	body    []byte
	tries   int
}

func (s *Service) lease(ctx context.Context) (leased, bool, error) {
	var held leased
	now := s.now()
	err := s.pool.QueryRow(ctx, `
		update discord_posts post
		   set tries = post.tries + 1, due_at = $2
		  from discord_webhooks hook
		 where hook.id = post.webhook_id
		   and post.id = (
		       select id from discord_posts
		        where sent_at is null and failed_at is null and due_at <= $1
		        order by due_at
		        for update skip locked
		        limit 1)
		returning post.id, hook.address, post.body::text, post.tries
	`, now, now.Add(lease)).Scan(&held.id, &held.address, &held.body, &held.tries)
	if errors.Is(err, pgx.ErrNoRows) {
		return leased{}, false, nil
	}
	if err != nil {
		return leased{}, false, fmt.Errorf("lease a Discord post: %w", err)
	}
	return held, true, nil
}

func (s *Service) send(ctx context.Context, held leased) error {
	address, err := s.sealing.Open(held.address)
	if err != nil {
		return fmt.Errorf("open the Discord address: %w", err)
	}
	status := s.post(ctx, string(address), held.body)
	switch {
	case status >= 200 && status < 300:
		return s.settle(ctx, held.id, "sent_at = now()")
	case status == 0 || status == http.StatusTooManyRequests || status >= 500:
		if held.tries < maxTries {
			return s.settle(ctx, held.id, "due_at = now() + $2::interval", backoff(held.tries).String())
		}
	}
	// ponytail: a failed post is only logged; show it to its owner if creators ask
	log.Printf("Discord post %s failed after %d tries with status %d", held.id, held.tries, status)
	return s.settle(ctx, held.id, "failed_at = now()")
}

// post answers Discord's status, or 0 when Discord could not be reached
func (s *Service) post(ctx context.Context, address string, body []byte) int {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return 0
	}
	request.Header.Set("Content-Type", "application/json")
	answer, err := s.client.Do(request)
	if err != nil {
		return 0
	}
	defer answer.Body.Close()
	io.Copy(io.Discard, io.LimitReader(answer.Body, 64<<10))
	return answer.StatusCode
}

func (s *Service) settle(ctx context.Context, id uuid.UUID, set string, args ...any) error {
	if _, err := s.pool.Exec(ctx, `update discord_posts set `+set+` where id = $1`, append([]any{id}, args...)...); err != nil {
		return fmt.Errorf("settle the Discord post: %w", err)
	}
	return nil
}

func backoff(tries int) time.Duration {
	return time.Duration(1<<tries) * 15 * time.Second
}
