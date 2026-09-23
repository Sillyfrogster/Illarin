package connect

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func offered(ids ...string) []SendFormat {
	formats := make([]SendFormat, 0, len(ids))
	for _, id := range ids {
		formats = append(formats, SendFormat{Format: id, Label: id})
	}
	return formats
}

func TestTheAppGetsTheFirstFormatItAcceptsThatIllarinOffers(t *testing.T) {
	t.Parallel()
	chosen, label, found := chooseFormat(
		[]string{"card_v3", "card_v2"}, offered("card_v2", "card_v3"), false,
	)

	if !found || chosen != "card_v3" || label != "card_v3" {
		t.Fatalf("chooseFormat = %q, %q, %t, want card_v3", chosen, label, found)
	}
}

func TestAnAcceptedFormatIllarinDoesNotOfferSelectsNothing(t *testing.T) {
	t.Parallel()
	chosen, _, found := chooseFormat(
		[]string{"invented_by_the_client", "card_v2"}, offered("card_v2"), false,
	)

	if !found || chosen != "card_v2" {
		t.Fatalf("chooseFormat = %q, %t, want card_v2", chosen, found)
	}
}

func TestAWorkWithAnUploadedFileFallsBackToRaw(t *testing.T) {
	t.Parallel()
	chosen, _, found := chooseFormat([]string{"card_v3"}, offered("card_v2"), true)

	if !found || chosen != format.Raw {
		t.Fatalf("chooseFormat = %q, %t, want raw", chosen, found)
	}
}

func TestNothingIsChosenWhenNoFormatFitsAndThereIsNoUploadedFile(t *testing.T) {
	t.Parallel()
	if _, _, found := chooseFormat([]string{"card_v3"}, offered("card_v2"), false); found {
		t.Fatal("chooseFormat found a format with nothing to fall back to")
	}
}

func TestASecondWaitSupersedesTheFirst(t *testing.T) {
	t.Parallel()
	waiting := newHub(4)
	appID := uuid.New()

	first, admitted := waiting.hold(appID)
	if !admitted {
		t.Fatal("the first wait was not admitted")
	}
	second, admitted := waiting.hold(appID)
	if !admitted {
		t.Fatal("the second wait was not admitted")
	}

	select {
	case <-first.superseded:
	default:
		t.Fatal("the first wait was left hanging by the second")
	}
	select {
	case <-second.superseded:
		t.Fatal("the second wait was superseded by itself")
	default:
	}
}

func TestOneConnectedAppNeverHoldsTwoPlacesAtOnce(t *testing.T) {
	t.Parallel()
	waiting := newHub(1)
	appID := uuid.New()

	if _, admitted := waiting.hold(appID); !admitted {
		t.Fatal("the first wait was not admitted")
	}
	if _, admitted := waiting.hold(appID); !admitted {
		t.Fatal("the same connected app was refused its own place back")
	}
	if _, admitted := waiting.hold(uuid.New()); admitted {
		t.Fatal("a second connected app was admitted past the limit")
	}
}

func TestReleasingLeavesAWaitThatAlreadySupersededItRegistered(t *testing.T) {
	t.Parallel()
	waiting := newHub(2)
	appID := uuid.New()

	first, _ := waiting.hold(appID)
	second, _ := waiting.hold(appID)
	waiting.release(appID, first)
	waiting.signal(appID)

	select {
	case <-second.work:
	default:
		t.Fatal("the live wait lost its wake-up when a superseded one was released")
	}
}

func TestAWorkThatNeedsNoCapabilityGoesToAnyConnectedApp(t *testing.T) {
	t.Parallel()
	if !installs(nil, Sendable{}) {
		t.Fatal("a work with no install capability was refused")
	}
}

func TestAnExtensionGoesOnlyToAConnectedAppDeclaringItsAppsInstallCapability(t *testing.T) {
	t.Parallel()
	sendable := Sendable{InstallCapabilities: []string{"chat.lumiverse:extension-install"}}

	if installs([]string{"app.sillytavern:extension-install", "org.example:extension-install"}, sendable) {
		t.Fatal("another app's capability, or an unknown one, let the extension through")
	}
	if !installs([]string{"org.example:media-sidecars", "chat.lumiverse:extension-install"}, sendable) {
		t.Fatal("the declared capability did not let the extension through")
	}
}
