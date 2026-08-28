package publication

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const categoryLabelLimit = 48

// Category is one kind of post the publication sorts its writing into.
type Category struct {
	ID       uuid.UUID
	Slug     string
	Label    string
	Position int
	Retired  bool
}

// CategoryUpdate carries only the parts of a category a request named.
type CategoryUpdate struct {
	Label   *string
	Retired *bool
}

// Categories answers every seeded category, retired ones included.
func (s *Service) Categories(ctx context.Context) ([]Category, error) {
	rows, err := s.pool.Query(ctx, selectCategories+` order by category.position, category.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read publication categories: %w", err)
	}
	defer rows.Close()
	return collectCategories(rows)
}

// UpdateCategory changes a category's label and whether it is still current.
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
		update publication_categories
		   set label = $2,
		       retired_at = case when $3 then coalesce(retired_at, now()) else null end,
		       updated_at = now()
		 where id = $1
	`, id, checked, retired)
	if err != nil {
		return Category{}, fmt.Errorf("update publication category: %w", err)
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "category.updated", CategoryID: &id}); err != nil {
		return Category{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Category{}, fmt.Errorf("commit category update: %w", err)
	}
	return s.category(ctx, id)
}

// OrderCategories puts the categories in the order it is given them.
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
	rows, err := tx.Query(ctx, `select id from publication_categories for update`)
	if err != nil {
		return nil, fmt.Errorf("lock publication categories: %w", err)
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
			update publication_categories set position = $2, updated_at = now() where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order publication category: %w", err)
		}
	}
	if err := recordPublicationAudit(ctx, tx, change{Actor: actor, Action: "category.ordered"}); err != nil {
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
		return Category{}, fmt.Errorf("read publication category: %w", err)
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
	  from publication_categories category
`

func collectCategories(rows pgx.Rows) ([]Category, error) {
	found := make([]Category, 0, 4)
	for rows.Next() {
		var one Category
		err := rows.Scan(&one.ID, &one.Slug, &one.Label, &one.Position, &one.Retired)
		if err != nil {
			return nil, fmt.Errorf("read a publication category: %w", err)
		}
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read publication categories: %w", err)
	}
	return found, nil
}
