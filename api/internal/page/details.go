package page

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

const MaxNameRunes = 200

var (
	ErrNameTooLong        = errors.New("the name is too long")
	ErrBlurbTooLong       = errors.New("the blurb is too long")
	ErrRatingUnanswerable = errors.New("a published work needs an adult content answer")
)

type Details struct {
	OwnerID uuid.UUID
	WorkID  uuid.UUID
	Name    string
	Blurb   string
	IsNSFW  *bool
}

func (s *Service) SetDetails(ctx context.Context, in Details, candidate *work.Candidate) error {
	name := strings.TrimSpace(in.Name)
	if utf8.RuneCountInString(name) > MaxNameRunes {
		return fmt.Errorf("%w: %d characters is past %d", ErrNameTooLong,
			utf8.RuneCountInString(name), MaxNameRunes)
	}
	if utf8.RuneCountInString(in.Blurb) > format.MaxBlurbRunes {
		return fmt.Errorf("%w: %d characters is past %d", ErrBlurbTooLong,
			utf8.RuneCountInString(in.Blurb), format.MaxBlurbRunes)
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := candidate.Lock(ctx, tx, in.OwnerID, in.WorkID); err != nil {
		return err
	}

	var lifecycle string
	if err := tx.QueryRow(ctx, `select lifecycle from works where id = $1`, in.WorkID).Scan(&lifecycle); err != nil {
		return err
	}

	if in.IsNSFW == nil && work.Lifecycle(lifecycle) != work.LifecycleDraft {
		return ErrRatingUnanswerable
	}
	if err := s.works.ChangeContent(ctx, tx, in.WorkID, func() error {
		if _, err := tx.Exec(ctx, `
			update works set name = $2, blurb = $3, is_nsfw = $4, updated_at = now()
			 where id = $1
		`, in.WorkID, name, in.Blurb, in.IsNSFW); err != nil {
			return fmt.Errorf("save work header: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return candidate.Commit(ctx, tx, in.WorkID)
}
