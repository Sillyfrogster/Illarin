package upload

import (
	"context"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/google/uuid"
)

var kindsAskedForAnApp = map[string]struct{}{"preset": {}, "theme": {}}

func KindAsksForAnApp(kind string) bool {
	_, asked := kindsAskedForAnApp[kind]
	return asked
}

func Apps(kind string) []string {
	if !KindAsksForAnApp(kind) {
		return nil
	}
	var supported []string
	switch kind {
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

func seedElements(kind string, app string) ([]block.Element, error) {
	if !KindAsksForAnApp(kind) {
		if app != "" {
			return nil, ErrAppNotAnswered
		}
		return nil, nil
	}
	switch kind {
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
	kind string,
	app string,
) (uuid.UUID, error) {
	if _, ok := block.Catalog(kind); !ok || !s.reg.BuildsFromNothing(kind) {
		return uuid.Nil, ErrKindNotBuildable
	}
	seeded, err := seedElements(kind, app)
	if err != nil {
		return uuid.Nil, err
	}
	blocks, err := block.Place(kind, seeded)
	if err != nil {
		return uuid.Nil, ErrKindNotBuildable
	}

	a := asset.Asset{
		ID: uuid.New(), Kind: kind, Tags: []string{},
		Discovery: asset.DiscoveryListed, Lifecycle: asset.LifecycleDraft,
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := asset.InsertAsset(ctx, tx, a, ownerID, nil); err != nil {
		return uuid.Nil, err
	}
	if err := block.Insert(ctx, tx, a.ID, blocks); err != nil {
		return uuid.Nil, err
	}
	if err := s.assets.WriteProjections(ctx, tx, a.ID); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return a.ID, nil
}
