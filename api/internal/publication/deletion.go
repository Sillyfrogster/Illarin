package publication

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const StatusDeleted = "deleted"

const RecoveryDays = 30

const RecoveryPoll = time.Hour

var (
	ErrPostDeleted      = errors.New("the post has been deleted")
	ErrPostNotDeleted   = errors.New("the post has not been deleted")
	ErrPostInPublicView = errors.New("the post is still in public view")
	ErrDeletePublished  = errors.New("only an admin may delete a post that has published")
	ErrRecoveryExpired  = errors.New("the recovery window has closed")
)

type Deletion struct {
	At    time.Time
	Until time.Time
	By    string
}

func (s *Service) DeletePost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := mayRemove(editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin deletion: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status == StatusPublished {
		return Post{}, ErrPostInPublicView
	}
	if err := overtakeSchedule(ctx, tx, editor, locked, StatusDeleted); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set deleted_at = now(), recoverable_until = now() + make_interval(days => $2),
		       deleted_by = $3, updated_at = now()
		 where id = $1
	`, id, RecoveryDays, editor.ID)
	if err != nil {
		return Post{}, fmt.Errorf("delete the post: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.deleted",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, Before: locked.Status, After: StatusDeleted,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit deletion: %w", err)
	}
	return s.post(ctx, id)
}

func (s *Service) RecoverPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	if err := mayRemove(editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin recovery: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockRemovedPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.DeletedAt == nil {
		return Post{}, ErrPostNotDeleted
	}
	if !locked.RecoverableUntil.After(s.now()) {
		return Post{}, ErrRecoveryExpired
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set deleted_at = null, recoverable_until = null, deleted_by = null,
		       updated_at = now()
		 where id = $1
	`, id)
	if err != nil {
		return Post{}, fmt.Errorf("recover the post: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.recovered",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, Before: StatusDeleted, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit recovery: %w", err)
	}
	return s.post(ctx, id)
}

func mayRemove(editor Editor, found Post) error {
	if editor.Admin {
		return nil
	}
	if found.PublishedAt != nil {
		return ErrDeletePublished
	}
	return nil
}

func (s *Service) RunRecovery(ctx context.Context, onError func(error)) {
	ticker := time.NewTicker(RecoveryPoll)
	defer ticker.Stop()
	for {
		_, err := s.RemoveExpiredPosts(ctx, s.now())
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

func (s *Service) RemoveExpiredPosts(ctx context.Context, now time.Time) (int, error) {
	expired, err := s.expiredPosts(ctx, now)
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, id := range expired {
		gone, err := s.removePost(ctx, id, now)
		if err != nil {
			return removed, err
		}
		if gone {
			removed++
		}
	}
	return removed, nil
}

func (s *Service) expiredPosts(ctx context.Context, now time.Time) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		select id from posts
		 where deleted_at is not null and recoverable_until <= $1
		 order by recoverable_until, id
	`, now)
	if err != nil {
		return nil, fmt.Errorf("read the posts past recovery: %w", err)
	}
	defer rows.Close()
	expired := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read a post past recovery: %w", err)
		}
		expired = append(expired, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the posts past recovery: %w", err)
	}
	return expired, nil
}

func (s *Service) removePost(ctx context.Context, id uuid.UUID, now time.Time) (bool, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin removal: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockRemovedPost(ctx, tx, id)
	if errors.Is(err, ErrPostNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if locked.DeletedAt == nil || locked.RecoverableUntil.After(now) {
		return false, nil
	}
	if locked.PublishedAt != nil {
		if err := retireAddresses(ctx, tx, locked); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(ctx, `delete from posts where id = $1`, id); err != nil {
		return false, fmt.Errorf("remove the post: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit removal: %w", err)
	}
	return true, nil
}

func retireAddresses(ctx context.Context, tx pgx.Tx, locked working) error {
	var explanation string
	err := tx.QueryRow(ctx, `
		select coalesce((
			select explanation from post_withdrawals
			 where post_id = $1 order by withdrawn_at desc limit 1
		), '')
	`, locked.ID).Scan(&explanation)
	if err != nil {
		return fmt.Errorf("read what a retired address tells readers: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into post_addresses (slug, current_slug, explanation)
		select reservation.slug, $2, $3 from post_slugs reservation
		 where reservation.post_id = $1
		on conflict (slug) do nothing
	`, locked.ID, locked.Slug, explanation)
	if err != nil {
		return fmt.Errorf("keep the addresses a post published under: %w", err)
	}
	return nil
}

func (s *Service) retiredAddress(ctx context.Context, slug string) (Tombstone, error) {
	var found Tombstone
	err := s.pool.QueryRow(ctx, `
		select current_slug, explanation from post_addresses where slug = $1
	`, slug).Scan(&found.Slug, &found.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return Tombstone{}, ErrPostNotFound
	}
	if err != nil {
		return Tombstone{}, fmt.Errorf("read a retired address: %w", err)
	}
	return found, nil
}
