package format

import (
	"slices"
	"testing"
)

func TestAnExtensionNeedsTheInstallCapabilityOfAnAppThatReadsItsFormat(t *testing.T) {
	t.Parallel()
	needed := InstallCapabilities("extension", []string{"extension_spindle"})

	if !slices.Equal(needed, []string{"chat.lumiverse:extension-install"}) {
		t.Fatalf("InstallCapabilities = %v, want Lumiverse's install capability", needed)
	}
}

func TestASillyTavernExtensionNeedsSillyTavernsCapability(t *testing.T) {
	t.Parallel()
	needed := InstallCapabilities("extension", []string{"extension_sillytavern"})

	if !slices.Equal(needed, []string{"app.sillytavern:extension-install"}) {
		t.Fatalf("InstallCapabilities = %v, want SillyTavern's install capability", needed)
	}
}

func TestOtherTypesNeedNoCapability(t *testing.T) {
	t.Parallel()
	for _, workType := range []string{"character", "lorebook", "preset", "theme", "pack"} {
		if needed := InstallCapabilities(workType, []string{"chara_card_v3", "extension_spindle"}); len(needed) != 0 {
			t.Errorf("%s needs %v, want nothing", workType, needed)
		}
	}
}

func TestAnExtensionInAFormatNoAppReadsCannotBeInstalledAnywhere(t *testing.T) {
	t.Parallel()
	if needed := InstallCapabilities("extension", []string{"invented"}); len(needed) != 0 {
		t.Fatalf("InstallCapabilities = %v, want nothing", needed)
	}
}
