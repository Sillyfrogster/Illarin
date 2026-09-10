package publication

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
)

var DeliveryDelays = []time.Duration{
	0,
	5 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
	2 * time.Hour,
	5 * time.Hour,
	10 * time.Hour,
	14 * time.Hour,
	20 * time.Hour,
	24 * time.Hour,
}

var DeliveryAttempts = len(DeliveryDelays)

const deliveryJitter = 0.1

const (
	SettledArrived   = "arrived"
	SettledExhausted = "exhausted"
	SettledRefused   = "refused"
	SettledGone      = "gone"
	SettledRemoved   = "removed"
	SettledDisabled  = "disabled"
	SettledMoved     = "moved"
)

func deliveryDelay(made int, spread float64) (time.Duration, bool) {
	if made < 0 || made >= DeliveryAttempts {
		return 0, false
	}
	agreed := DeliveryDelays[made]
	return agreed + time.Duration(float64(agreed)*deliveryJitter*(spread*2-1)), true
}

type verdict struct {
	Outcome string
	Detail  string
	Status  *int
	Took    time.Duration
	Retry   bool
	After   time.Duration
	Reason  string
	Gone    bool
}

var arrived = verdict{Outcome: AttemptDelivered, Reason: SettledArrived}

var unreachable = verdict{
	Outcome: AttemptUnreachable, Detail: "Illarin could not reach it.", Retry: true,
}

func readAnswer(answer outbound.Answer) verdict {
	said := fmt.Sprintf("It answered %d.", answer.Status)
	switch {
	case answer.Status >= http.StatusOK && answer.Status < http.StatusMultipleChoices:
		return arrived
	case answer.Status == http.StatusTooManyRequests:
		return verdict{
			Outcome: AttemptRefused, Detail: "It asked Illarin to wait.",
			Retry: true, After: answer.RetryAfter,
		}
	case answer.Status == http.StatusGone:
		return verdict{
			Outcome: AttemptRefused,
			Detail:  "It answered 410, so nothing is sent there again.",
			Reason:  SettledGone, Gone: true,
		}
	case answer.Status == http.StatusRequestTimeout,
		answer.Status == http.StatusTooEarly,
		answer.Status >= http.StatusInternalServerError:
		return verdict{Outcome: AttemptRefused, Detail: said, Retry: true}
	default:
		return verdict{Outcome: AttemptRefused, Detail: said, Reason: SettledRefused}
	}
}

func stopped(reason string) verdict {
	return verdict{Outcome: AttemptRefused, Detail: whyStopped[reason], Reason: reason}
}

var whyStopped = map[string]string{
	SettledRemoved:  "The destination was removed.",
	SettledDisabled: "The destination was disabled.",
	SettledMoved:    "The destination moved to another address.",
	SettledGone:     "The destination answered 410 and receives nothing further.",
}

func spread() float64 { return rand.Float64() }
