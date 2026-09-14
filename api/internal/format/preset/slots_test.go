package preset

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
)

func TestEveryAppOfferedHasSlotNamesForEveryGroup(t *testing.T) {
	t.Parallel()
	for _, app := range Apps() {
		if !app.Known() || app.Label() == "" {
			t.Errorf("%s is offered and has no name of its own", app)
		}
		elements, err := Seed(app)
		if err != nil {
			t.Fatalf("seed %s: %v", app, err)
		}
		if len(elements) != 4 {
			t.Fatalf("%s seeded %d elements, want three settings groups and the nudges",
				app, len(elements))
		}
		for _, role := range []block.Role{
			block.RoleSamplerSettings, block.RoleCompletionSettings,
			block.RoleAdvancedSettings, block.RolePromptNudges,
		} {
			found := false
			for _, element := range elements {
				if element.Role != role {
					continue
				}
				found = true
				if len(block.ItemIDs(element.Content)) == 0 {
					t.Errorf("%s seeded %s with no slot names", app, role)
				}
			}
			if !found {
				t.Errorf("%s seeded nothing for %s", app, role)
			}
		}
	}
}

func TestTheSeedSuppliesNamesAndNoValues(t *testing.T) {
	t.Parallel()
	elements, err := Seed(SillyTavern)
	if err != nil {
		t.Fatalf("seed SillyTavern: %v", err)
	}
	for _, element := range elements {
		group, ok := element.Content.(block.SettingGroup)
		if !ok {
			continue
		}
		if group.Supplied() != 0 {
			t.Errorf("%s arrived with %d settings filled in", element.Role, group.Supplied())
		}
		for _, setting := range group.Settings {
			if setting.Name == "" {
				t.Errorf("%s carries a slot with no name", element.Role)
			}
			if !setting.Type.Known() {
				t.Errorf("%s carries %s at unknown type %q",
					element.Role, setting.Name, setting.Type)
			}
		}
	}
}

func TestAnAppWithNoSlotNamesIsRefused(t *testing.T) {
	t.Parallel()
	if _, err := Seed(App("koboldcpp")); err == nil {
		t.Error("an app Illarin knows nothing about was seeded anyway")
	}
}

func TestTheTwoAppsShareAlmostNoSettingsNames(t *testing.T) {
	t.Parallel()
	names := map[App]map[string]struct{}{}
	for _, app := range Apps() {
		elements, err := Seed(app)
		if err != nil {
			t.Fatalf("seed %s: %v", app, err)
		}
		names[app] = map[string]struct{}{}
		for _, element := range elements {
			group, ok := element.Content.(block.SettingGroup)
			if !ok {
				continue
			}
			for _, setting := range group.Settings {
				names[app][setting.Name] = struct{}{}
			}
		}
	}
	shared := []string{}
	for name := range names[SillyTavern] {
		if _, both := names[Lumiverse][name]; both {
			shared = append(shared, name)
		}
	}
	if len(shared) > 2 {
		t.Errorf("the two apps share %v, want no more than temperature and seed", shared)
	}
}
