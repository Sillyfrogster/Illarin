package publication

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Action is one entry of the private record of what happened to a post.
type Action struct {
	ID         uuid.UUID
	Actor      string
	Credential string
	Action     string
	Revision   *int
	Before     string
	After      string
	At         time.Time
}

// PostHistory answers what has been done to one post, newest first. It names
// actors, editions and times, and never carries a word anyone wrote.
func (s *Service) PostHistory(ctx context.Context, editor Editor, id uuid.UUID) ([]Action, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select audit.id, actor.username, audit.credential, audit.action, revision.number,
		       audit.before_state, audit.after_state, audit.recorded_at
		  from publication_audits audit
		  left join users actor on actor.id = audit.actor_id
		  left join post_revisions revision on revision.id = audit.revision_id
		 where audit.post_id = $1
		 order by audit.recorded_at desc, audit.id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("read what has happened to a post: %w", err)
	}
	defer rows.Close()
	done := make([]Action, 0, 8)
	for rows.Next() {
		var one Action
		var actor, before, after *string
		err := rows.Scan(
			&one.ID, &actor, &one.Credential, &one.Action, &one.Revision,
			&before, &after, &one.At,
		)
		if err != nil {
			return nil, fmt.Errorf("read something that happened to a post: %w", err)
		}
		if actor != nil {
			one.Actor = *actor
		}
		if before != nil {
			one.Before = *before
		}
		if after != nil {
			one.After = *after
		}
		done = append(done, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read what has happened to a post: %w", err)
	}
	return done, nil
}
