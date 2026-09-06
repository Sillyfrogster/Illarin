package publication

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// EventWithdrawn is the immutable record of a post leaving public view.
const EventWithdrawn = "publication.post.withdrawn.v1"

// StatusWithdrawn is a post readers can no longer read and Illarin still holds.
const StatusWithdrawn = "withdrawn"

const withdrawalTextLimit = 500

var (
	ErrPostNotPublic    = errors.New("the post is not in public view")
	ErrPostNotWithdrawn = errors.New("the post is not out of public view")
	ErrPostWithdrawn    = errors.New("the post is out of public view")
)

// Withdrawal is one removal of a post from public view. The reason is private
// to Illarin and the explanation is the only part a reader ever sees.
type Withdrawal struct {
	Reason      string
	Explanation string
	By          string
	At          time.Time
}

// Tombstone is the whole of what a withdrawn address answers with. It carries
// no title, no body and nobody's name.
type Tombstone struct {
	Slug        string
	Explanation string
}

// WithdrawPost takes a published post out of public view and keeps everything
// it was, so a correction can put the same post back at the same address.
func (s *Service) WithdrawPost(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
	reason, explanation string,
) (Post, error) {
	said, err := checkWithdrawal(reason, explanation)
	if err != nil {
		return Post{}, err
	}
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin withdrawal: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostNotPublic
	}
	public, err := publicRevision(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if err := overtakeSchedule(ctx, tx, editor, locked, StatusWithdrawn); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		insert into post_withdrawals (id, post_id, revision_id, reason, explanation, withdrawn_by)
		values ($1, $2, $3, $4, $5, $6)
	`, uuid.New(), id, public, said.reason, said.explanation, editor.ID)
	if err != nil {
		return Post{}, fmt.Errorf("keep why the post was withdrawn: %w", err)
	}
	_, err = tx.Exec(ctx, `
		update posts set status = $2, updated_at = now() where id = $1
	`, id, StatusWithdrawn)
	if err != nil {
		return Post{}, fmt.Errorf("take the post out of public view: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into publication_events (id, post_id, revision_id, type) values ($1, $2, $3, $4)
	`, uuid.New(), id, public, EventWithdrawn)
	if err != nil {
		return Post{}, fmt.Errorf("record the withdrawal event: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.withdrawn",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, RevisionID: &public,
		Before: StatusPublished, After: StatusWithdrawn,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit withdrawal: %w", err)
	}
	return s.post(ctx, id)
}

// RepublishPost puts a withdrawn post back at the address and under the date it
// already had, showing whichever edition the author names.
func (s *Service) RepublishPost(
	ctx context.Context,
	editor Editor,
	id, revisionID uuid.UUID,
	version int,
) (Post, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Post{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin republication: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	if locked.Status != StatusWithdrawn {
		return Post{}, ErrPostNotWithdrawn
	}
	if _, err := lockedRevision(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	withdrawn, err := publicRevision(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set status = $2, public_revision_id = $3,
		       updated_public_at = case when $4 then now() else updated_public_at end,
		       updated_at = now()
		 where id = $1
	`, id, StatusPublished, revisionID, revisionID != withdrawn)
	if err != nil {
		return Post{}, fmt.Errorf("put the post back in public view: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into publication_events (id, post_id, revision_id, type) values ($1, $2, $3, $4)
	`, uuid.New(), id, revisionID, EventPublished)
	if err != nil {
		return Post{}, fmt.Errorf("record the republication event: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Credential: editor.Credential(), Action: "post.republished",
		GrantID: locked.GrantID, TokenID: editor.Token,
		PostID: &id, RevisionID: &revisionID,
		Before: StatusWithdrawn, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit republication: %w", err)
	}
	return s.post(ctx, id)
}

// WithdrawnPost answers the tombstone behind one address, current or former,
// and always names the address the tombstone itself lives at.
func (s *Service) WithdrawnPost(ctx context.Context, slug string) (Tombstone, error) {
	var found Tombstone
	err := s.pool.QueryRow(ctx, `
		select post.slug, withdrawal.explanation
		  from posts post
		  join post_withdrawals withdrawal on withdrawal.post_id = post.id
		 where post.status = $2
		   and (post.slug = $1
		        or post.id = (select post_id from post_slugs where slug = $1))
		 order by withdrawal.withdrawn_at desc
		 limit 1
	`, slug, StatusWithdrawn).Scan(&found.Slug, &found.Explanation)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.retiredAddress(ctx, slug)
	}
	if err != nil {
		return Tombstone{}, fmt.Errorf("read a withdrawn address: %w", err)
	}
	return found, nil
}

// said is a checked withdrawal, with the private half and the public half apart.
type said struct {
	reason      string
	explanation string
}

func checkWithdrawal(reason, explanation string) (said, error) {
	held := said{reason: oneParagraph(reason), explanation: oneParagraph(explanation)}
	if held.reason == "" {
		return said{}, FieldError{
			Field:   "reason",
			Message: "Say why the post is coming down. Only Illarin reads this.",
		}
	}
	if len(held.reason) > withdrawalTextLimit {
		return said{}, FieldError{
			Field:   "reason",
			Message: fmt.Sprintf("Keep the reason under %d characters.", withdrawalTextLimit),
		}
	}
	if len(held.explanation) > withdrawalTextLimit {
		return said{}, FieldError{
			Field:   "explanation",
			Message: fmt.Sprintf("Keep the public explanation under %d characters.", withdrawalTextLimit),
		}
	}
	if held.explanation != "" && held.explanation == held.reason {
		return said{}, FieldError{
			Field:   "explanation",
			Message: "The public explanation is written for readers. Write it separately.",
		}
	}
	return held, nil
}

// oneParagraph answers what somebody typed as a single run of prose, so a
// tombstone never carries the shape of the box it was written in.
func oneParagraph(written string) string {
	return strings.Join(strings.Fields(written), " ")
}

// publicRevision reads the edition a post is showing readers inside the
// transaction that is about to change it.
func publicRevision(ctx context.Context, tx pgx.Tx, id uuid.UUID) (uuid.UUID, error) {
	var revisionID *uuid.UUID
	err := tx.QueryRow(ctx, `select public_revision_id from posts where id = $1`, id).
		Scan(&revisionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("read the edition the post is showing: %w", err)
	}
	if revisionID == nil {
		return uuid.Nil, ErrPostNotPublic
	}
	return *revisionID, nil
}

// withdrawalsFor reads the newest withdrawal each named post has, which is the
// one a post still out of public view is under.
func (s *Service) withdrawalsFor(
	ctx context.Context,
	ids []uuid.UUID,
) (map[uuid.UUID]Withdrawal, error) {
	rows, err := s.pool.Query(ctx, `
		select distinct on (withdrawal.post_id)
		       withdrawal.post_id, withdrawal.reason, withdrawal.explanation,
		       actor.username, withdrawal.withdrawn_at
		  from post_withdrawals withdrawal
		  left join users actor on actor.id = withdrawal.withdrawn_by
		 where withdrawal.post_id = any($1)
		 order by withdrawal.post_id, withdrawal.withdrawn_at desc
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("read why posts were withdrawn: %w", err)
	}
	defer rows.Close()
	latest := make(map[uuid.UUID]Withdrawal, len(ids))
	for rows.Next() {
		var postID uuid.UUID
		var one Withdrawal
		var actor *string
		if err := rows.Scan(&postID, &one.Reason, &one.Explanation, &actor, &one.At); err != nil {
			return nil, fmt.Errorf("read why a post was withdrawn: %w", err)
		}
		if actor != nil {
			one.By = *actor
		}
		latest[postID] = one
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read why posts were withdrawn: %w", err)
	}
	return latest, nil
}
