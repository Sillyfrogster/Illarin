package publication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrNotPostAdmin    = errors.New("only an Illarin admin may correct a published post")
	ErrPostUnpublished = errors.New("the post has not been published")
)

// CorrectAddress moves a published post to a new permalink and keeps the address it left pointing at the post.
func (s *Service) CorrectAddress(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	candidate string,
) (Post, error) {
	if !editor.Admin {
		return Post{}, ErrNotPostAdmin
	}
	slug, err := checkSlug(candidate)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin address correction: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostUnpublished
	}
	if slug == locked.Slug {
		return Post{}, FieldError{Field: "slug", Message: "The post already lives at that address."}
	}
	taken, err := addressTaken(ctx, tx, id, slug)
	if err != nil {
		return Post{}, err
	}
	if taken {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err := reserveAddress(ctx, tx, id, editor.ID, slug); err != nil {
		return Post{}, err
	}
	_, err = tx.Exec(ctx, `update posts set slug = $2, updated_at = now() where id = $1`, id, slug)
	if isUniqueViolation(err) {
		return Post{}, FieldError{Field: "slug", Message: "Another post already has that address."}
	}
	if err != nil {
		return Post{}, fmt.Errorf("move the post to its corrected address: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.address.corrected", GrantID: locked.GrantID,
		PostID: &id, Before: StatusPublished, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit address correction: %w", err)
	}
	return s.post(ctx, id)
}

// CorrectByline replaces the attribution a published post carries, keeping its author, grant and every captured revision.
func (s *Service) CorrectByline(
	ctx context.Context,
	editor Editor,
	id uuid.UUID,
	handle string,
) (Post, error) {
	if !editor.Admin {
		return Post{}, ErrNotPostAdmin
	}
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return Post{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Post{}, fmt.Errorf("begin byline correction: %w", err)
	}
	defer tx.Rollback(ctx)
	locked, err := lockPost(ctx, tx, id)
	if err != nil {
		return Post{}, err
	}
	if locked.Status != StatusPublished {
		return Post{}, ErrPostUnpublished
	}
	if err := replaceByline(ctx, tx, id, accountID, locked.GrantID); err != nil {
		return Post{}, err
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: editor.ID, Action: "post.byline.corrected", GrantID: locked.GrantID,
		PostID: &id, SubjectID: &accountID, Before: StatusPublished, After: StatusPublished,
	})
	if err != nil {
		return Post{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Post{}, fmt.Errorf("commit byline correction: %w", err)
	}
	return s.post(ctx, id)
}
