package upload

import (
	"context"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

// appsAsked lists the apps a type built from nothing is made for, as its format modules name them
func appsAsked(workType string) []string {
	switch workType {
	case "preset":
		return appIDs(preset.Apps())
	case "theme":
		return appIDs(theme.Apps())
	}
	return []string{}
}

func appIDs[A ~string](apps []A) []string {
	ids := make([]string, len(apps))
	for i, app := range apps {
		ids[i] = string(app)
	}
	return ids
}

// BuildChoices lists every type that can be built from nothing, with the apps each asks for
func (s *Service) BuildChoices() BuildChoices {
	choices := BuildChoices{Types: []BuildChoice{}}
	for _, workType := range slices.Sorted(slices.Values(block.Types())) {
		if s.buildable(workType) {
			choices.Types = append(choices.Types, BuildChoice{
				Type: workType, Apps: page.AppNames(appsAsked(workType)), Drafts: drafts(workType),
			})
		}
	}
	return choices
}

// drafts lays out the page each app's empty draft of the type opens with
func drafts(workType string) []Draft {
	apps := appsAsked(workType)
	if len(apps) == 0 {
		apps = []string{""}
	}
	out := make([]Draft, 0, len(apps))
	for _, app := range apps {
		placed, err := draftBlocks(workType, app)
		if err != nil {
			continue
		}
		served, err := block.ToBlocks(workType, placed)
		if err != nil {
			continue
		}
		out = append(out, Draft{App: app, Blocks: served})
	}
	return out
}

func draftBlocks(workType, app string) ([]block.Block, error) {
	seeded, err := seedElements(workType, app)
	if err != nil {
		return nil, err
	}
	placed, err := block.Place(workType, seeded)
	if err != nil {
		return nil, ErrTypeNotBuildable
	}
	return placed, nil
}

func (s *Service) buildable(workType string) bool {
	_, defined := block.Definitions(workType)
	return defined && s.reg.BuildsFromNothing(workType)
}

func seedElements(workType string, app string) ([]block.Element, error) {
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
	}
	if app != "" {
		return nil, ErrAppNotAnswered
	}
	return nil, nil
}

func (s *Service) StartFromNothing(
	ctx context.Context,
	ownerID uuid.UUID,
	workType string,
	app string,
) (uuid.UUID, error) {
	if !s.buildable(workType) {
		return uuid.Nil, ErrTypeNotBuildable
	}
	blocks, err := draftBlocks(workType, app)
	if err != nil {
		return uuid.Nil, err
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
