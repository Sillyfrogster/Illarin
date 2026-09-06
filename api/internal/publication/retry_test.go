package publication

import (
	"net/http"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
)

func TestOneRunSpansAboutSeventySixHours(t *testing.T) {
	span := time.Duration(0)
	for _, gap := range DeliveryDelays {
		span += gap
	}

	if span < 75*time.Hour || span > 77*time.Hour {
		t.Errorf("a run spans %s, want about 76 hours", span)
	}
	if DeliveryDelays[0] != 0 {
		t.Errorf("the first attempt waits %s, want none", DeliveryDelays[0])
	}
}

func TestJitterKeepsAnAttemptWithinATenthOfItsGap(t *testing.T) {
	for place, gap := range DeliveryDelays {
		for _, spread := range []float64{0, 0.5, 1} {
			waited, again := deliveryDelay(place, spread)
			if !again {
				t.Fatalf("the run has no attempt %d", place+1)
			}
			if waited < gap*9/10 || waited > gap*11/10 {
				t.Errorf("attempt %d at spread %v waits %s, want about %s",
					place+1, spread, waited, gap)
			}
		}
	}
	if middle, _ := deliveryDelay(1, 0.5); middle != DeliveryDelays[1] {
		t.Errorf("the middle of the window is %s, want %s", middle, DeliveryDelays[1])
	}
}

func TestARunEndsAfterItsLastAttempt(t *testing.T) {
	if _, again := deliveryDelay(DeliveryAttempts, 0.5); again {
		t.Error("the run offered an attempt past its last")
	}
	if _, again := deliveryDelay(DeliveryAttempts-1, 0.5); !again {
		t.Error("the run refused its last attempt")
	}
}

func TestEveryAnswerAnEndpointGivesHasOneMeaning(t *testing.T) {
	for _, one := range []struct {
		status int
		retry  bool
		reason string
		gone   bool
	}{
		{http.StatusOK, false, SettledArrived, false},
		{http.StatusAccepted, false, SettledArrived, false},
		{http.StatusNoContent, false, SettledArrived, false},
		{http.StatusMovedPermanently, false, SettledRefused, false},
		{http.StatusTemporaryRedirect, false, SettledRefused, false},
		{http.StatusBadRequest, false, SettledRefused, false},
		{http.StatusUnauthorized, false, SettledRefused, false},
		{http.StatusNotFound, false, SettledRefused, false},
		{http.StatusGone, false, SettledGone, true},
		{http.StatusRequestTimeout, true, "", false},
		{http.StatusTooEarly, true, "", false},
		{http.StatusTooManyRequests, true, "", false},
		{http.StatusInternalServerError, true, "", false},
		{http.StatusServiceUnavailable, true, "", false},
	} {
		said := readAnswer(outbound.Answer{Status: one.status})

		if said.Retry != one.retry {
			t.Errorf("%d retries %v, want %v", one.status, said.Retry, one.retry)
		}
		if said.Reason != one.reason {
			t.Errorf("%d settles as %q, want %q", one.status, said.Reason, one.reason)
		}
		if said.Gone != one.gone {
			t.Errorf("%d disables the endpoint %v, want %v", one.status, said.Gone, one.gone)
		}
	}
}

func TestAnAnswerCarriesNoResponseBodyIntoTheRecord(t *testing.T) {
	said := readAnswer(outbound.Answer{
		Status: http.StatusInternalServerError, Body: []byte("the stack trace"),
	})

	if said.Detail != "It answered 500." {
		t.Errorf("detail = %q, want the status alone", said.Detail)
	}
}
