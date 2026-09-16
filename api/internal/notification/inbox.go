package notification

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 50
)

var ErrPageSize = errors.New("a page holds between 1 and 50 notifications")

// Entry is one notification in an account's inbox.
type Entry struct {
	ID        uuid.UUID
	Type      Type
	Asset     *uuid.UUID
	Words     Words
	Count     int
	CreatedAt time.Time
	ReadAt    *time.Time
}

// Cursor is the last entry on the previous page.
type Cursor struct {
	Before   time.Time
	BeforeID uuid.UUID
}

type Page struct {
	Entries []Entry
	Next    *Cursor
}

// Inbox lists an account's visible entries newest first, starting after the cursor when there is one.
func (s *Service) Inbox(ctx context.Context, account uuid.UUID, after *Cursor, limit int) (Page, error) {
	if limit < 1 || limit > MaxPageSize {
		return Page{}, ErrPageSize
	}
	var before *time.Time
	var beforeID *uuid.UUID
	if after != nil {
		before, beforeID = &after.Before, &after.BeforeID
	}
	rows, err := s.pool.Query(ctx, `
		select id, type, asset_id, words, update_count, created_at, read_at
		  from inbox_entries
		 where account_id = $1
		   and ($2::timestamptz is null or (created_at, id) < ($2, $3::uuid))
		 order by created_at desc, id desc
		 limit $4
	`, account, before, beforeID, limit+1)
	if err != nil {
		return Page{}, fmt.Errorf("list the inbox: %w", err)
	}
	defer rows.Close()
	page := Page{Entries: make([]Entry, 0, limit)}
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(
			&entry.ID, &entry.Type, &entry.Asset, &entry.Words, &entry.Count, &entry.CreatedAt, &entry.ReadAt,
		); err != nil {
			return Page{}, fmt.Errorf("read an inbox entry: %w", err)
		}
		page.Entries = append(page.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("list the inbox: %w", err)
	}
	if len(page.Entries) > limit {
		page.Entries = page.Entries[:limit]
		last := page.Entries[limit-1]
		page.Next = &Cursor{Before: last.CreatedAt, BeforeID: last.ID}
	}
	return page, nil
}

// Unread counts the visible entries the account has not opened.
func (s *Service) Unread(ctx context.Context, account uuid.UUID) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `
		select count(*) from inbox_entries where account_id = $1 and read_at is null
	`, account).Scan(&count); err != nil {
		return 0, fmt.Errorf("count unread notifications: %w", err)
	}
	return count, nil
}

// MarkRead marks one of the account's entries read and reports whether the account has that entry.
func (s *Service) MarkRead(ctx context.Context, account, id uuid.UUID) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
		update inbox_entries set read_at = coalesce(read_at, now())
		 where id = $1 and account_id = $2
	`, id, account)
	if err != nil {
		return false, fmt.Errorf("mark a notification read: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// MarkAllRead marks every entry the account has as read.
func (s *Service) MarkAllRead(ctx context.Context, account uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `
		update inbox_entries set read_at = now() where account_id = $1 and read_at is null
	`, account); err != nil {
		return fmt.Errorf("mark every notification read: %w", err)
	}
	return nil
}

// Remove takes one entry out of the account's inbox and reports whether the account had it.
func (s *Service) Remove(ctx context.Context, account, id uuid.UUID) (bool, error) {
	tag, err := s.pool.Exec(ctx, `delete from inbox_entries where id = $1 and account_id = $2`, id, account)
	if err != nil {
		return false, fmt.Errorf("remove a notification: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}

// Clear takes every entry the account can see out of its inbox.
func (s *Service) Clear(ctx context.Context, account uuid.UUID) error {
	if _, err := s.pool.Exec(ctx, `delete from inbox_entries where account_id = $1`, account); err != nil {
		return fmt.Errorf("clear the inbox: %w", err)
	}
	return nil
}
