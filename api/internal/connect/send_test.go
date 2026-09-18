package connect

import (
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func offered(ids ...string) []DeliveryTarget {
	targets := make([]DeliveryTarget, 0, len(ids))
	for _, id := range ids {
		targets = append(targets, DeliveryTarget{Format: id, Label: id})
	}
	return targets
}

func TestTheInstancePicksTheFirstFormatItAcceptsThatIllarinOffers(t *testing.T) {
	t.Parallel()
	chosen, label, found := chooseTarget(
		[]string{"card_v3", "card_v2"}, offered("card_v2", "card_v3"), false,
	)

	if !found || chosen != "card_v3" || label != "card_v3" {
		t.Fatalf("chooseTarget = %q, %q, %t, want card_v3", chosen, label, found)
	}
}

func TestAnAcceptedFormatIllarinDoesNotOfferSelectsNothing(t *testing.T) {
	t.Parallel()
	chosen, _, found := chooseTarget(
		[]string{"invented_by_the_client", "card_v2"}, offered("card_v2"), false,
	)

	if !found || chosen != "card_v2" {
		t.Fatalf("chooseTarget = %q, %t, want card_v2", chosen, found)
	}
}

func TestAnAssetWithAnUploadedFileFallsBackToRaw(t *testing.T) {
	t.Parallel()
	chosen, _, found := chooseTarget([]string{"card_v3"}, offered("card_v2"), true)

	if !found || chosen != format.RawTarget {
		t.Fatalf("chooseTarget = %q, %t, want raw", chosen, found)
	}
}

func TestNothingIsChosenWhenNoFormatFitsAndThereIsNoUploadedFile(t *testing.T) {
	t.Parallel()
	if _, _, found := chooseTarget([]string{"card_v3"}, offered("card_v2"), false); found {
		t.Fatal("chooseTarget found a target with nothing to fall back to")
	}
}

func TestASecondWaitSupersedesTheFirst(t *testing.T) {
	t.Parallel()
	waiting := newHub(4)
	instanceID := uuid.New()

	first, admitted := waiting.hold(instanceID)
	if !admitted {
		t.Fatal("the first wait was not admitted")
	}
	second, admitted := waiting.hold(instanceID)
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

func TestOneInstanceNeverHoldsTwoPlacesAtOnce(t *testing.T) {
	t.Parallel()
	waiting := newHub(1)
	instanceID := uuid.New()

	if _, admitted := waiting.hold(instanceID); !admitted {
		t.Fatal("the first wait was not admitted")
	}
	if _, admitted := waiting.hold(instanceID); !admitted {
		t.Fatal("the same instance was refused its own place back")
	}
	if _, admitted := waiting.hold(uuid.New()); admitted {
		t.Fatal("a second instance was admitted past the limit")
	}
}

func TestReleasingLeavesAWaitThatAlreadySupersededItRegistered(t *testing.T) {
	t.Parallel()
	waiting := newHub(2)
	instanceID := uuid.New()

	first, _ := waiting.hold(instanceID)
	second, _ := waiting.hold(instanceID)
	waiting.release(instanceID, first)
	waiting.signal(instanceID)

	select {
	case <-second.work:
	default:
		t.Fatal("the live wait lost its wake-up when a superseded one was released")
	}
}

func TestAnAssetThatNeedsNoCapabilityGoesToAnyInstance(t *testing.T) {
	t.Parallel()
	if !installs(nil, Deliverable{}) {
		t.Fatal("an asset with no install capability was refused")
	}
}

func TestAnExtensionGoesOnlyToAnInstanceDeclaringItsAppsInstallCapability(t *testing.T) {
	t.Parallel()
	sendable := Deliverable{InstallCapabilities: []string{"chat.lumiverse:extension-install"}}

	if installs([]string{"app.sillytavern:extension-install", "org.example:extension-install"}, sendable) {
		t.Fatal("another app's capability, or an unknown one, let the extension through")
	}
	if !installs([]string{"org.example:media-sidecars", "chat.lumiverse:extension-install"}, sendable) {
		t.Fatal("the declared capability did not let the extension through")
	}
}
