package blog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	postbody "github.com/Sillyfrogster/Illarin/api/internal/blog/body"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	RevisionCheckpoint  = "checkpoint"
	RevisionPublication = "publication"
	RevisionSchedule    = "schedule"
)

var ErrRevisionNotFound = errors.New("no such revision of that post")

type Revision struct {
	ID          uuid.UUID
	Number      int
	Title       string
	Summary     string
	Slug        string
	Category    Category
	CapturedFor string
	CapturedBy  string
	CapturedAt  time.Time
	Public      bool
}

func (s *Service) Revisions(ctx context.Context, editor Editor, id uuid.UUID) ([]Revision, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		select revision.id, revision.number, revision.title, revision.summary, revision.slug,
		       category.id, category.slug, category.label, category.position,
		       category.retired_at is not null,
		       revision.captured_for, keeper.username, revision.captured_at,
		       revision.id = post.public_revision_id
		  from post_revisions revision
		  join posts post on post.id = revision.post_id
		  join publication_categories category on category.id = revision.category_id
		  left join users keeper on keeper.id = revision.captured_by
		 where revision.post_id = $1
		 order by revision.number desc
	`, id)
	if err != nil {
		return nil, fmt.Errorf("read the editions a post has kept: %w", err)
	}
	defer rows.Close()
	kept := make([]Revision, 0, 8)
	for rows.Next() {
		var one Revision
		var keeper *string
		var public *bool
		err := rows.Scan(
			&one.ID, &one.Number, &one.Title, &one.Summary, &one.Slug,
			&one.Category.ID, &one.Category.Slug, &one.Category.Label,
			&one.Category.Position, &one.Category.Retired,
			&one.CapturedFor, &keeper, &one.CapturedAt, &public,
		)
		if err != nil {
			return nil, fmt.Errorf("read an edition a post has kept: %w", err)
		}
		if keeper != nil {
			one.CapturedBy = *keeper
		}
		one.Public = public != nil && *public
		kept = append(kept, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the editions a post has kept: %w", err)
	}
	return kept, nil
}

func (s *Service) Checkpoint(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	version int,
) (Revision, error) {
	current, err := s.post(ctx, id)
	if err != nil {
		return Revision{}, err
	}
	if err := s.mayManage(ctx, editor, current); err != nil {
		return Revision{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Revision{}, fmt.Errorf("begin checkpoint: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Revision{}, err
	}
	if locked.Version != version {
		return Revision{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	revisionID, err := captureRevision(ctx, tx, editor, locked, RevisionCheckpoint)
	if err != nil {
		return Revision{}, err
	}
	if err := carryUsesForward(ctx, tx, id, revisionID); err != nil {
		return Revision{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.checkpointed",
		GrantID: locked.GrantID,
		PostID:  &id, RevisionID: &revisionID, Before: locked.Status, After: locked.Status,
	})
	if err != nil {
		return Revision{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Revision{}, fmt.Errorf("commit checkpoint: %w", err)
	}
	return s.revision(ctx, id, revisionID)
}

func (s *Service) RestoreRevision(
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
		return Post{}, fmt.Errorf("begin restore: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Version != version {
		return Post{}, Stale{Version: locked.Version, UpdatedAt: locked.UpdatedAt}
	}
	kept, err := lockedRevision(ctx, tx, id, revisionID)
	if err != nil {
		return Post{}, err
	}
	body, err := postbody.Read(kept.Document)
	if err != nil {
		return Post{}, documentRefusal(err)
	}
	document, err := json.Marshal(body)
	if err != nil {
		return Post{}, fmt.Errorf("write the restored post body: %w", err)
	}
	address, err := restoredAddress(ctx, tx, locked, kept)
	if err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `
		update posts
		   set category_id = $2, title = $3, summary = $4, slug = $5,
		       document = $6, document_version = $7,
		       release_app_id = $8, release_version = $9, release_url = $10,
		       header_media_id = $11, header_alt = $12, header_caption = $13,
		       social_media_id = $14,
		       working_version = working_version + 1, updated_at = now()
		 where id = $1
	`, id, kept.CategoryID, kept.Title, kept.Summary, address,
		document, postbody.Version, kept.ReleaseAppID, kept.ReleaseVersion, kept.ReleaseAddress,
		kept.HeaderMediaID, kept.HeaderAlt, kept.HeaderCaption, kept.SocialMediaID)
	if err != nil {
		return Post{}, fmt.Errorf("restore the edition into the working copy: %w", err)
	}
	if err := carryUsesBack(ctx, tx, id, revisionID); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.revision.restored",
		GrantID: locked.GrantID,
		PostID:  &id, RevisionID: &revisionID, Before: locked.Status, After: locked.Status,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit restore: %w", err)
	}
	return s.post(ctx, id)
}

func restoredAddress(ctx context.Context, tx pgx.Tx, locked working, kept working) (*string, error) {
	if locked.PublishedAt != nil || kept.Slug == "" || kept.Slug == locked.Slug {
		return nullable(locked.Slug), nil
	}
	taken, err := addressTaken(ctx, tx, locked.ID, kept.Slug)
	if err != nil {
		return nil, err
	}
	if taken {
		return nullable(locked.Slug), nil
	}
	return nullable(kept.Slug), nil
}

func carryUsesBack(ctx context.Context, tx pgx.Tx, postID, revisionID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		delete from post_media_uses where post_id = $1 and revision_id is null
	`, postID)
	if err != nil {
		return fmt.Errorf("clear what the working copy refers to: %w", err)
	}
	_, err = tx.Exec(ctx, `
		insert into post_media_uses (media_id, post_id, revision_id)
		select media_id, post_id, null from post_media_uses where revision_id = $1
	`, revisionID)
	if err != nil {
		return fmt.Errorf("carry the pictures back into the working copy: %w", err)
	}
	return nil
}

func carryUsesForward(ctx context.Context, tx pgx.Tx, postID, revisionID uuid.UUID) error {
	_, err := tx.Exec(ctx, `
		insert into post_media_uses (media_id, post_id, revision_id)
		select media_id, post_id, $2 from post_media_uses
		 where post_id = $1 and revision_id is null
	`, postID, revisionID)
	if err != nil {
		return fmt.Errorf("carry the pictures forward into the revision: %w", err)
	}
	return nil
}

func captureRevision(
	ctx context.Context,
	tx pgx.Tx,
	editor Editor,
	locked working,
	reason string,
) (uuid.UUID, error) {
	id := uuid.New()
	_, err := tx.Exec(ctx, `
		insert into post_revisions (id, post_id, number, title, summary, slug, category_id,
		                            document, document_version, release_app_id,
		                            release_version, release_url, header_media_id,
		                            header_alt, header_caption, social_media_id,
		                            captured_for, captured_by)
		values ($1, $2,
		        coalesce((select max(number) from post_revisions where post_id = $2), 0) + 1,
		        $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
	`, id, locked.ID, locked.Title, locked.Summary, locked.Slug, locked.CategoryID,
		locked.Document, postbody.Version, locked.ReleaseAppID,
		locked.ReleaseVersion, locked.ReleaseAddress, locked.HeaderMediaID,
		locked.HeaderAlt, locked.HeaderCaption, locked.SocialMediaID, reason, editor.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("capture the post revision: %w", err)
	}
	return id, nil
}

func lockedRevision(ctx context.Context, tx pgx.Tx, postID, id uuid.UUID) (working, error) {
	var kept working
	err := tx.QueryRow(ctx, `
		select revision.id, revision.category_id, revision.slug, revision.title, revision.summary,
		       revision.document, revision.release_app_id, revision.release_version,
		       revision.release_url, revision.header_media_id, revision.header_alt,
		       revision.header_caption, revision.social_media_id
		  from post_revisions revision
		 where revision.id = $1 and revision.post_id = $2
	`, id, postID).Scan(
		&kept.ID, &kept.CategoryID, &kept.Slug, &kept.Title, &kept.Summary,
		&kept.Document, &kept.ReleaseAppID, &kept.ReleaseVersion, &kept.ReleaseAddress,
		&kept.HeaderMediaID, &kept.HeaderAlt, &kept.HeaderCaption, &kept.SocialMediaID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return working{}, ErrRevisionNotFound
	}
	if err != nil {
		return working{}, fmt.Errorf("read the edition being restored: %w", err)
	}
	return kept, nil
}

func (s *Service) revision(ctx context.Context, postID, id uuid.UUID) (Revision, error) {
	var one Revision
	var keeper *string
	var public *bool
	err := s.pool.QueryRow(ctx, `
		select revision.id, revision.number, revision.title, revision.summary, revision.slug,
		       category.id, category.slug, category.label, category.position,
		       category.retired_at is not null,
		       revision.captured_for, keeper.username, revision.captured_at,
		       revision.id = post.public_revision_id
		  from post_revisions revision
		  join posts post on post.id = revision.post_id
		  join publication_categories category on category.id = revision.category_id
		  left join users keeper on keeper.id = revision.captured_by
		 where revision.id = $1 and revision.post_id = $2
	`, id, postID).Scan(
		&one.ID, &one.Number, &one.Title, &one.Summary, &one.Slug,
		&one.Category.ID, &one.Category.Slug, &one.Category.Label,
		&one.Category.Position, &one.Category.Retired,
		&one.CapturedFor, &keeper, &one.CapturedAt, &public,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Revision{}, ErrRevisionNotFound
	}
	if err != nil {
		return Revision{}, fmt.Errorf("read a kept edition: %w", err)
	}
	if keeper != nil {
		one.CapturedBy = *keeper
	}
	one.Public = public != nil && *public
	return one, nil
}

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
