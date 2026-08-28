package publication

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	distinctionNameLimit        = 48
	distinctionExplanationLimit = 200
	showcaseLimit               = 6
)

// SourceManual is the assignment source for an award the authority made by hand.
const SourceManual = "manual"

type Form string

const (
	FormPosition Form = "position"
	FormTitle    Form = "title"
	FormBadge    Form = "badge"
)

// Distinction is one defined position, title or badge.
type Distinction struct {
	ID          uuid.UUID
	Form        Form
	Name        string
	Explanation string
	Mark        *Mark
	Position    int
	Retired     bool
}

// Assignment is the record that one account holds one distinction.
type Assignment struct {
	ID          uuid.UUID
	Distinction Distinction
	IssuedBy    *uuid.UUID
	Source      string
	AssignedAt  time.Time
	Active      bool
	Position    int
}

// Showcase is the distinction set a profile shows a visitor.
type Showcase struct {
	Positions []Distinction
	Titles    []Distinction
	Badges    []Distinction
}

// DistinctionEdit is the definition text the authority supplies.
type DistinctionEdit struct {
	Form        Form
	Name        string
	Explanation string
}

// DistinctionUpdate carries only the parts of a definition a request named.
type DistinctionUpdate struct {
	Name        *string
	Explanation *string
	Retired     *bool
}

// Distinctions answers every definition, retired ones included.
func (s *Service) Distinctions(ctx context.Context) ([]Distinction, error) {
	rows, err := s.pool.Query(ctx, selectDistinctions+` order by form, definition.position, definition.created_at`)
	if err != nil {
		return nil, fmt.Errorf("read distinctions: %w", err)
	}
	defer rows.Close()
	return collectDistinctions(rows)
}

// Define records a new distinction below the ones its form already has.
func (s *Service) Define(ctx context.Context, actor uuid.UUID, in DistinctionEdit) (Distinction, error) {
	edit, err := validateDistinction(in)
	if err != nil {
		return Distinction{}, err
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Distinction{}, fmt.Errorf("begin distinction definition: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		insert into profile_distinctions (id, form, name, explanation, position)
		values ($1, $2, $3, $4, coalesce(
			(select max(position) + 1 from profile_distinctions where form = $2), 0
		))
	`, id, string(edit.Form), edit.Name, edit.Explanation)
	if err != nil {
		return Distinction{}, fmt.Errorf("define distinction: %w", err)
	}
	if err := recordAudit(ctx, tx, actor, "distinction.defined", &id, nil, nil); err != nil {
		return Distinction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Distinction{}, fmt.Errorf("commit distinction definition: %w", err)
	}
	return s.distinction(ctx, id)
}

// Update changes what one distinction says and whether it is still current.
func (s *Service) Update(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	in DistinctionUpdate,
) (Distinction, error) {
	current, err := s.distinction(ctx, id)
	if err != nil {
		return Distinction{}, err
	}
	edit := DistinctionEdit{Form: current.Form, Name: current.Name, Explanation: current.Explanation}
	if in.Name != nil {
		edit.Name = *in.Name
	}
	if in.Explanation != nil {
		edit.Explanation = *in.Explanation
	}
	checked, err := validateDistinction(edit)
	if err != nil {
		return Distinction{}, err
	}
	retired := current.Retired
	if in.Retired != nil {
		retired = *in.Retired
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Distinction{}, fmt.Errorf("begin distinction update: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update profile_distinctions
		   set name = $2,
		       explanation = $3,
		       retired_at = case when $4 then coalesce(retired_at, now()) else null end,
		       updated_at = now()
		 where id = $1
	`, id, checked.Name, checked.Explanation, retired)
	if err != nil {
		return Distinction{}, fmt.Errorf("update distinction: %w", err)
	}
	if err := recordAudit(ctx, tx, actor, "distinction.updated", &id, nil, nil); err != nil {
		return Distinction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Distinction{}, fmt.Errorf("commit distinction update: %w", err)
	}
	return s.distinction(ctx, id)
}

// Order puts one form's definitions in the order it is given them.
func (s *Service) Order(
	ctx context.Context,
	actor uuid.UUID,
	form Form,
	ids []uuid.UUID,
) ([]Distinction, error) {
	if !knownForm(form) {
		return nil, FieldError{Field: "form", Message: "That is not a distinction form."}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin distinction order: %w", err)
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `
		select id from profile_distinctions where form = $1 for update
	`, string(form))
	if err != nil {
		return nil, fmt.Errorf("lock distinctions: %w", err)
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
			update profile_distinctions set position = $2, updated_at = now() where id = $1
		`, id, position)
		if err != nil {
			return nil, fmt.Errorf("order distinction: %w", err)
		}
	}
	if err := recordAudit(ctx, tx, actor, "distinction.ordered", nil, nil, nil); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit distinction order: %w", err)
	}
	return s.Distinctions(ctx)
}

// SetMark gives a recognition an Illarin-hosted mark, which makes it a badge.
func (s *Service) SetMark(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
	file io.Reader,
) (Distinction, error) {
	current, err := s.distinction(ctx, id)
	if err != nil {
		return Distinction{}, err
	}
	if !earned(current.Form) {
		return Distinction{}, ErrDistinctionForm
	}
	stored, prepared, err := s.media.Accept(ctx, file)
	if err != nil {
		return Distinction{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Distinction{}, fmt.Errorf("begin mark change: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := setForm(ctx, tx, id, FormBadge); err != nil {
		return Distinction{}, err
	}
	err = s.replaceMark(ctx, tx, distinctionMarks, id, stored.ID, prepared.Width, prepared.Height)
	if err != nil {
		return Distinction{}, err
	}
	if err := recordAudit(ctx, tx, actor, "distinction.marked", &id, nil, nil); err != nil {
		return Distinction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Distinction{}, fmt.Errorf("commit mark change: %w", err)
	}
	return s.distinction(ctx, id)
}

// ClearMark takes a badge's mark away, which leaves it a title.
func (s *Service) ClearMark(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Distinction, error) {
	current, err := s.distinction(ctx, id)
	if err != nil {
		return Distinction{}, err
	}
	if !earned(current.Form) {
		return Distinction{}, ErrDistinctionForm
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Distinction{}, fmt.Errorf("begin mark removal: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.dropMark(ctx, tx, distinctionMarks, id); err != nil {
		return Distinction{}, err
	}
	if err := setForm(ctx, tx, id, FormTitle); err != nil {
		return Distinction{}, err
	}
	if err := recordAudit(ctx, tx, actor, "distinction.unmarked", &id, nil, nil); err != nil {
		return Distinction{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Distinction{}, fmt.Errorf("commit mark removal: %w", err)
	}
	return s.distinction(ctx, id)
}

// setForm moves one recognition between the two presentations it may take, and only a badge may hold a mark, so it runs before a mark is attached and after one is taken away.
func setForm(ctx context.Context, tx pgx.Tx, id uuid.UUID, form Form) error {
	_, err := tx.Exec(ctx, `
		update profile_distinctions set form = $2, updated_at = now() where id = $1
	`, id, string(form))
	if err != nil {
		return fmt.Errorf("set the distinction form: %w", err)
	}
	return nil
}

// earned answers whether a form is recognition rather than a job.
func earned(form Form) bool {
	return form == FormTitle || form == FormBadge
}

func (s *Service) distinction(ctx context.Context, id uuid.UUID) (Distinction, error) {
	rows, err := s.pool.Query(ctx, selectDistinctions+` where definition.id = $1`, id)
	if err != nil {
		return Distinction{}, fmt.Errorf("read distinction: %w", err)
	}
	defer rows.Close()
	found, err := collectDistinctions(rows)
	if err != nil {
		return Distinction{}, err
	}
	if len(found) == 0 {
		return Distinction{}, ErrNotFound
	}
	return found[0], nil
}

const selectDistinctions = `
	select definition.id, definition.form, definition.name, definition.explanation,
	       definition.position, definition.retired_at is not null,
	       mark.id, mark.width, mark.height
	  from profile_distinctions definition
	  left join publication_media mark
	         on mark.id = definition.mark_media_id and mark.blob_id is not null
`

func collectDistinctions(rows pgx.Rows) ([]Distinction, error) {
	found := make([]Distinction, 0, 8)
	for rows.Next() {
		var one Distinction
		var form string
		var markID *uuid.UUID
		var width, height *int
		err := rows.Scan(
			&one.ID, &form, &one.Name, &one.Explanation,
			&one.Position, &one.Retired, &markID, &width, &height,
		)
		if err != nil {
			return nil, fmt.Errorf("read a distinction: %w", err)
		}
		one.Form = Form(form)
		one.Mark = scanMark(markID, width, height)
		found = append(found, one)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read distinctions: %w", err)
	}
	return found, nil
}

func collectIDs(rows pgx.Rows) ([]uuid.UUID, error) {
	defer rows.Close()
	found := make([]uuid.UUID, 0, 8)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read an identifier: %w", err)
		}
		found = append(found, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read identifiers: %w", err)
	}
	return found, nil
}

func sameSet(present, given []uuid.UUID) bool {
	if len(present) != len(given) {
		return false
	}
	remaining := make(map[uuid.UUID]bool, len(present))
	for _, id := range present {
		remaining[id] = true
	}
	for _, id := range given {
		if !remaining[id] {
			return false
		}
		delete(remaining, id)
	}
	return len(remaining) == 0
}

func knownForm(form Form) bool {
	return form == FormPosition || form == FormTitle || form == FormBadge
}

func validateDistinction(in DistinctionEdit) (DistinctionEdit, error) {
	if !knownForm(in.Form) {
		return DistinctionEdit{}, FieldError{
			Field:   "form",
			Message: "A distinction is a position, a title or a badge.",
		}
	}
	name, err := plainField("name", in.Name, distinctionNameLimit)
	if err != nil {
		return DistinctionEdit{}, err
	}
	if name == "" {
		return DistinctionEdit{}, FieldError{Field: "name", Message: "Give the distinction a name."}
	}
	explanation, err := plainField("explanation", in.Explanation, distinctionExplanationLimit)
	if err != nil {
		return DistinctionEdit{}, err
	}
	return DistinctionEdit{Form: in.Form, Name: name, Explanation: explanation}, nil
}

func plainField(field, raw string, limit int) (string, error) {
	value := strings.TrimSpace(raw)
	for _, char := range value {
		if unicode.IsControl(char) {
			return "", FieldError{
				Field:   field,
				Message: "Use plain text without control characters.",
			}
		}
	}
	if len([]rune(value)) > limit {
		return "", FieldError{
			Field:   field,
			Message: fmt.Sprintf("Keep this to %d characters or fewer.", limit),
		}
	}
	return value, nil
}
