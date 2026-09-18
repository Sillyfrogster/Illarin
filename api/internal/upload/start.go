package upload

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

var typesAskedForAnApp = map[string]struct{}{"preset": {}, "theme": {}}

func TypeAsksForAnApp(workType string) bool {
	_, asked := typesAskedForAnApp[workType]
	return asked
}

func Apps(workType string) []string {
	if !TypeAsksForAnApp(workType) {
		return nil
	}
	var supported []string
	switch workType {
	case "preset":
		for _, app := range preset.Apps() {
			supported = append(supported, string(app))
		}
	case "theme":
		for _, app := range theme.Apps() {
			supported = append(supported, string(app))
		}
	}
	return supported
}

func seedElements(workType string, app string) ([]block.Element, error) {
	if !TypeAsksForAnApp(workType) {
		if app != "" {
			return nil, ErrAppNotAnswered
		}
		return nil, nil
	}
	switch workType {
	case "preset":
		chosen := preset.App(app)
		if !chosen.Known() {
			return nil, ErrAppNotAnswered
		}
		return preset.Seed(chosen)
	case "theme":
		chosen := theme.App(app)
		if !chosen.Known() {
			return nil, ErrAppNotAnswered
		}
		return theme.Seed(chosen)
	default:
		return nil, ErrAppNotAnswered
	}
}
func (s *Service) StartFromNothing(
	ctx context.Context,
	ownerID uuid.UUID,
	workType string,
	app string,
) (uuid.UUID, error) {
	if _, ok := block.Definitions(workType); !ok || !s.reg.BuildsFromNothing(workType) {
		return uuid.Nil, ErrTypeNotBuildable
	}
	seeded, err := seedElements(workType, app)
	if err != nil {
		return uuid.Nil, err
	}
	blocks, err := block.Place(workType, seeded)
	if err != nil {
		return uuid.Nil, ErrTypeNotBuildable
	}

	a := work.Work{
		ID: uuid.New(), Type: workType, Tags: []string{},
		Visibility: work.VisibilityListed, Lifecycle: work.LifecycleDraft,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := work.InsertWork(ctx, tx, a, ownerID, nil); err != nil {
		return uuid.Nil, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return uuid.Nil, err
	}
	if err := s.writeSummary(ctx, tx, a.ID); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return a.ID, nil
}
