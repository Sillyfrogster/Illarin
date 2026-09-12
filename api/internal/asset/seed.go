package asset

import (
	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
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
