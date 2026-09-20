package blog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Writer struct {
	AccountID uuid.UUID
	Handle    string
	Since     time.Time
}

func (s *Service) Writers(ctx context.Context) ([]Writer, error) {
	rows, err := s.pool.Query(ctx, `
		select writer.user_id, account.username, writer.since
		  from blog_writers writer
		  join users account on account.id = writer.user_id
		 order by writer.since desc
	`)
	if err != nil {
		return nil, fmt.Errorf("read the writers: %w", err)
	}
	defer rows.Close()
	found := make([]Writer, 0, 8)
	for rows.Next() {
		var one Writer
		if err := rows.Scan(&one.AccountID, &one.Handle, &one.Since); err != nil {
			return nil, fmt.Errorf("read a writer: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read the writers: %w", err)
	}
	return found, nil
}

func (s *Service) IsWriter(ctx context.Context, accountID uuid.UUID) (bool, error) {
	return isWriter(ctx, s.pool, accountID)
}

func isWriter(ctx context.Context, q queryRower, accountID uuid.UUID) (bool, error) {
	var writer bool
	err := q.QueryRow(ctx, `
		select exists (select 1 from blog_writers where user_id = $1)
	`, accountID).Scan(&writer)
	if err != nil {
		return false, fmt.Errorf("read the writer switch: %w", err)
	}
	return writer, nil
}

// SwitchWriterOn lets the account with this handle write and publish posts
func (s *Service) SwitchWriterOn(ctx context.Context, actor uuid.UUID, handle string) (Writer, error) {
	var writer Writer
	var verified bool
	err := s.pool.QueryRow(ctx, `
		select id, username, email_verified_at is not null from users where username = $1
	`, handle).Scan(&writer.AccountID, &writer.Handle, &verified)
	if errors.Is(err, pgx.ErrNoRows) {
		return Writer{}, ErrAccountNotFound
	}
	if err != nil {
		return Writer{}, fmt.Errorf("find the account being switched on: %w", err)
	}
	if !verified {
		return Writer{}, ErrAccountUnverified
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Writer{}, fmt.Errorf("begin writer switch: %w", err)
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `
		insert into blog_writers (user_id, switched_by) values ($1, $2)
		on conflict (user_id) do update set since = blog_writers.since
		returning since
	`, writer.AccountID, actor).Scan(&writer.Since)
	if err != nil {
		return Writer{}, fmt.Errorf("switch the writer on: %w", err)
	}
	err = recordActivity(ctx, tx, change{Actor: actor, Action: "writer.on", SubjectID: &writer.AccountID})
	if err != nil {
		return Writer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Writer{}, fmt.Errorf("commit writer switch: %w", err)
	}
	return writer, nil
}

// SwitchWriterOff ends the account's authority to write and stops its scheduled posts
func (s *Service) SwitchWriterOff(ctx context.Context, actor, accountID uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin writer switch: %w", err)
	}
	defer tx.Rollback(ctx)
	removed, err := tx.Exec(ctx, `delete from blog_writers where user_id = $1`, accountID)
	if err != nil {
		return fmt.Errorf("switch the writer off: %w", err)
	}
	if removed.RowsAffected() == 0 {
		return ErrAccountNotFound
	}
	if err := stopSchedulesBy(ctx, tx, accountID); err != nil {
		return err
	}
	err = recordActivity(ctx, tx, change{Actor: actor, Action: "writer.off", SubjectID: &accountID})
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit writer switch: %w", err)
	}
	return nil
}

const categoryLabelLimit = 48

type Category struct {
	ID       uuid.UUID
	Slug     string
	Label    string
	Position int
	Retired  bool
}

type CategoryUpdate struct {
	Label   *string
	Retired *bool
}

func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+` order by category.position, category.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read blog categories: %w", err)
	}
	defer rows.Close()
	return collectCategories(rows)
}

// OpenCategories lists the categories a post can be filed under today
func (s *Service) OpenCategories(ctx context.Context) ([]Category, error) {
	live, err := s.Categories(ctx)
	if err != nil {
		return nil, err
	}
	open := make([]Category, 0, len(live))
	for _, one := range live {
		if !one.Retired {
			open = append(open, one)
		}
	}
	return open, nil
}

func (s *Service) UpdateCategory(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in CategoryUpdate,
) (Category, error) {
	current, err := s.category(ctx, id)
	if err != nil {
		return Category{}, err
	}
	label := current.Label
	if in.Label != nil {
		label = *in.Label
	}
	checked, err := plainField("label", label, categoryLabelLimit)
	if err != nil {
		return Category{}, err
	}
	if checked == "" {
		return Category{}, FieldError{Field: "label", Message: "Give the category a label."}
	}
	retired := current.Retired
	if in.Retired != nil {
		retired = *in.Retired
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Category{}, fmt.Errorf("begin category update: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update blog_categories
		   set label = $2,
		       retired_at = case when $3 then coalesce(retired_at, now()) else null end,
		       updated_at = now()
		 where id = $1
	`, id, checked, retired)
	if err != nil {
		return Category{}, fmt.Errorf("update blog category: %w", err)
	}
	if err := recordActivity(ctx, tx, change{Actor: actor, Action: "category.updated", CategoryID: &id}); err != nil {
		return Category{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Category{}, fmt.Errorf("commit category update: %w", err)
	}
	return s.category(ctx, id)
}

func (s *Service) OrderCategories(
	ctx context.Context,
	actor uuid.UUID,
	ids []uuid.UUID,
) ([]Category, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin category order: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `select id from blog_categories for update`)
	if err != nil {
		return nil, fmt.Errorf("lock blog categories: %w", err)
	}
	present, err := collectIDs(rows)
	if err != nil {
		return nil, err
	}
	if !sameSet(present, ids) {
		return nil, ErrIncompleteOrder
	}
	for position, id := range ids {
		_, err := tx.Exec(ctx, `
			update blog_categories set position = $2, updated_at = now() where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order blog category: %w", err)
		}
	}
	if err := recordActivity(ctx, tx, change{Actor: actor, Action: "category.ordered"}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit category order: %w", err)
	}
	return s.Categories(ctx)
}

func (s *Service) category(ctx context.Context, id uuid.UUID) (Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+` where category.id = $1`, id)
	if err != nil {
		return Category{}, fmt.Errorf("read blog category: %w", err)
	}
	defer rows.Close()
	found, err := collectCategories(rows)
	if err != nil {
		return Category{}, err
	}
	if len(found) == 0 {
		return Category{}, ErrCategoryNotFound
	}
	return found[0], nil
}

const selectCategories = `
	select category.id, category.slug, category.label, category.position,
	       category.retired_at is not null
	  from blog_categories category
`

func collectCategories(rows pgx.Rows) ([]Category, error) {
	found := make([]Category, 0, 4)
	for rows.Next() {
		var one Category
		err := rows.Scan(&one.ID, &one.Slug, &one.Label, &one.Position, &one.Retired)
		if err != nil {
			return nil, fmt.Errorf("read a blog category: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read blog categories: %w", err)
	}
	return found, nil
}
