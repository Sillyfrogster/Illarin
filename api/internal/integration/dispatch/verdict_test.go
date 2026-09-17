package dispatch

import (
	"net/http"
	"testing"
	"time"
)

func TestTheScheduleSpansTenAttemptsWithinTenPercentJitter(t *testing.T) {
	t.Parallel()
	if Attempts != 10 {
		t.Fatalf("Attempts = %d, want 10", Attempts)
	}
	for made, agreed := range Delays {
		low, held := Delay(made, 0)
		high, _ := Delay(made, 1)
		if !held {
			t.Fatalf("attempt %d is not on the schedule", made)
		}
		if low != agreed-agreed/10 || high != agreed+agreed/10 {
			t.Errorf("attempt %d spreads %s to %s, want %s either side of %s",
				made, low, high, agreed/10, agreed)
		}
	}
	if _, held := Delay(Attempts, 0.5); held {
		t.Error("an attempt past the schedule was given a delay")
	}
	if _, held := Delay(-1, 0.5); held {
		t.Error("a negative attempt was given a delay")
	}
}

func TestWhatAnEndpointAnswersDecidesTheVerdict(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		status  int
		outcome string
		retry   bool
		reason  string
		gone    bool
	}{
		{http.StatusOK, OutcomeDelivered, false, Arrived, false},
		{http.StatusNoContent, OutcomeDelivered, false, Arrived, false},
		{http.StatusMovedPermanently, OutcomeRefused, false, Refused, false},
		{http.StatusBadRequest, OutcomeRefused, false, Refused, false},
		{http.StatusGone, OutcomeRefused, false, Gone, true},
		{http.StatusRequestTimeout, OutcomeRefused, true, "", false},
		{http.StatusTooEarly, OutcomeRefused, true, "", false},
		{http.StatusTooManyRequests, OutcomeRefused, true, "", false},
		{http.StatusInternalServerError, OutcomeRefused, true, "", false},
	} {
		said := ReadAnswer(Answer{Status: one.status, RetryAfter: 7 * time.Second})
		if said.Outcome != one.outcome || said.Retry != one.retry ||
			said.Reason != one.reason || said.Gone != one.gone {
			t.Errorf("%d read as %+v", one.status, said)
		}
		if one.status == http.StatusTooManyRequests && said.After != 7*time.Second {
			t.Errorf("429 waits %s, want the Retry-After of 7s", said.After)
		}
	}
}

func TestDiscordIsOnlyDeliveredWhenItNamesTheMessageItMade(t *testing.T) {
	t.Parallel()
	said, message := ReadAnnouncement(Answer{
		Status: http.StatusOK, Body: []byte(`{"id":"123456789012345678"}`),
	})
	if said.Outcome != OutcomeDelivered || message != "123456789012345678" {
		t.Errorf("a confirmed send read as %+v with message %q", said, message)
	}
	said, message = ReadAnnouncement(Answer{Status: http.StatusNoContent})
	if said.Outcome != OutcomeUnconfirmed || said.Reason != Unconfirmed || said.Retry || message != "" {
		t.Errorf("an unconfirmed send read as %+v with message %q", said, message)
	}
	said, _ = ReadAnnouncement(Answer{Status: http.StatusBadGateway})
	if said.Retry || said.Outcome != OutcomeUnconfirmed {
		t.Errorf("a Discord 502 read as %+v, want an unconfirmed send", said)
	}
}

func TestAStoppedVerdictNamesItsReason(t *testing.T) {
	t.Parallel()
	said := Stopped(Disabled)
	if said.Reason != Disabled || said.Outcome != OutcomeRefused || said.Detail == "" || said.Retry {
		t.Errorf("Stopped(disabled) = %+v", said)
	}
	own := Cancelled("withheld", "The asset is withheld.")
	if own.Reason != "withheld" || own.Detail != "The asset is withheld." || own.Retry {
		t.Errorf("Cancelled = %+v", own)
	}
}
