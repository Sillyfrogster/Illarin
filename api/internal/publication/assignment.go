package publication

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) Assignments(ctx context.Context, handle string) ([]Assignment, error) {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	return s.assignmentsFor(ctx, accountID, false)
}

func (s *Service) Assign(
	ctx context.Context,
	actor uuid.UUID,
	handle string,
	distinctionID uuid.UUID,
) (Assignment, error) {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return Assignment{}, err
	}
	given, err := s.distinction(ctx, distinctionID)
	if err != nil {
		return Assignment{}, err
	}
	if given.Retired {
		return Assignment{}, ErrRetiredAssigning
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Assignment{}, fmt.Errorf("begin distinction assignment: %w", err)
	}
	defer tx.Rollback(ctx)
	var locked uuid.UUID
	if err := tx.QueryRow(ctx, `select id from users where id = $1 for update`, accountID).Scan(&locked); err != nil {
		return Assignment{}, fmt.Errorf("lock the account being given a distinction: %w", err)
	}
	id := uuid.New()
	_, err = tx.Exec(ctx, `
		insert into profile_distinction_assignments
		       (id, user_id, distinction_id, issued_by, source, position)
		values ($1, $2, $3, $4, $5, coalesce(
			(select max(position) + 1 from profile_distinction_assignments
			  where user_id = $2 and active), 0
		))
	`, id, accountID, distinctionID, actor, SourceManual)
	if isUniqueViolation(err) {
		return Assignment{}, ErrAlreadyAssigned
	}
	if err != nil {
		return Assignment{}, fmt.Errorf("assign distinction: %w", err)
	}
	if err := recordAudit(ctx, tx, actor, "assignment.made", &distinctionID, &id, &accountID); err != nil {
		return Assignment{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Assignment{}, fmt.Errorf("commit distinction assignment: %w", err)
	}
	made, err := s.assignmentsFor(ctx, accountID, false)
	if err != nil {
		return Assignment{}, err
	}
	for _, assignment := range made {
		if assignment.ID == id {
			return assignment, nil
		}
	}
	return Assignment{}, ErrNotFound
}

func (s *Service) Withdraw(
	ctx context.Context,
	actor uuid.UUID,
	handle string,
	assignmentID uuid.UUID,
) error {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin distinction withdrawal: %w", err)
	}
	defer tx.Rollback(ctx)
	var distinctionID uuid.UUID
	err = tx.QueryRow(ctx, `
		update profile_distinction_assignments
		   set active = false, deactivated_at = now()
		 where id = $1 and user_id = $2 and active
		returning distinction_id
	`, assignmentID, accountID).Scan(&distinctionID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("withdraw distinction: %w", err)
	}
	err = recordAudit(ctx, tx, actor, "assignment.withdrawn", &distinctionID, &assignmentID, &accountID)
	if err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit distinction withdrawal: %w", err)
	}
	return nil
}

func (s *Service) OrderAssignments(
	ctx context.Context,
	actor uuid.UUID,
	handle string,
	ids []uuid.UUID,
) ([]Assignment, error) {
	accountID, err := s.accountByHandle(ctx, handle)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin assignment order: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		select id from profile_distinction_assignments
		 where user_id = $1 and active for update
	`, accountID)
	if err != nil {
		return nil, fmt.Errorf("lock assignments: %w", err)
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
			update profile_distinction_assignments set position = $2 where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order assignment: %w", err)
		}
	}
	if err := recordAudit(ctx, tx, actor, "assignment.ordered", nil, nil, &accountID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit assignment order: %w", err)
	}
	return s.assignmentsFor(ctx, accountID, false)
}

func (s *Service) Showcase(ctx context.Context, accountID uuid.UUID) (Showcase, error) {
	held, err := s.assignmentsFor(ctx, accountID, true)
	if err != nil {
		return Showcase{}, err
	}
	shown := Showcase{
		Positions: make([]Distinction, 0, len(held)),
		Titles:    make([]Distinction, 0, showcaseLimit),
		Badges:    make([]Distinction, 0, showcaseLimit),
	}
	for _, assignment := range held {
		switch assignment.Distinction.Form {
		case FormPosition:
			shown.Positions = append(shown.Positions, assignment.Distinction)
		case FormTitle:
			if len(shown.Titles) < showcaseLimit {
				shown.Titles = append(shown.Titles, assignment.Distinction)
			}
		case FormBadge:
			if len(shown.Badges) < showcaseLimit {
				shown.Badges = append(shown.Badges, assignment.Distinction)
			}
		}
	}
	return shown, nil
}

func (s *Service) assignmentsFor(
	ctx context.Context,
	accountID uuid.UUID,
	publicOnly bool,
) ([]Assignment, error) {
	rows, err := s.pool.Query(ctx, `
		select assignment.id, assignment.issued_by, assignment.source, assignment.assigned_at,
		       assignment.active, assignment.position,
		       definition.id, definition.form, definition.name, definition.explanation,
		       definition.position, definition.retired_at is not null,
		       mark.id, mark.width, mark.height
		  from profile_distinction_assignments assignment
		  join profile_distinctions definition on definition.id = assignment.distinction_id
		  left join publication_media mark
		         on mark.id = definition.mark_media_id and mark.blob_id is not null
		 where assignment.user_id = $1
		   and (not $2 or (assignment.active and definition.retired_at is null))
		 order by case when definition.form = 'position' then 0 else 1 end,
		          case when definition.form = 'position'
		               then definition.position else assignment.position end,
		          assignment.assigned_at
	`, accountID, publicOnly)
	if err != nil {
		return nil, fmt.Errorf("read distinction assignments: %w", err)
	}
	defer rows.Close()
	held := make([]Assignment, 0, 8)
	for rows.Next() {
		var one Assignment
		var form string
		var markID *uuid.UUID
		var width, height *int
		err := rows.Scan(
			&one.ID, &one.IssuedBy, &one.Source, &one.AssignedAt, &one.Active, &one.Position,
			&one.Distinction.ID, &form, &one.Distinction.Name, &one.Distinction.Explanation,
			&one.Distinction.Position, &one.Distinction.Retired, &markID, &width, &height,
		)
		if err != nil {
			return nil, fmt.Errorf("read a distinction assignment: %w", err)
		}
		one.Distinction.Form = Form(form)
		one.Distinction.Mark = scanMark(markID, width, height)
		held = append(held, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read distinction assignments: %w", err)
	}
	return held, nil
}
