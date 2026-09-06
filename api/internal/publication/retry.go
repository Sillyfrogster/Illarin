package publication

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
)

// deliveryDelays is the gap before each attempt of one run. The first attempt
// is immediate and the gaps after it leave the run about 76 hours long.
var deliveryDelays = []time.Duration{
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

// DeliveryAttempts is how many attempts one run of a delivery makes.
var DeliveryAttempts = len(deliveryDelays)

// deliveryJitter is how far either side of the agreed gap an attempt may fall,
// so a shared outage does not bring every receiver back at the same instant.
const deliveryJitter = 0.1

// The reasons a delivery is settled, kept apart from what one attempt found so
// that exhausted work reads differently from work an endpoint turned away.
const (
	SettledArrived   = "arrived"
	SettledExhausted = "exhausted"
	SettledRefused   = "refused"
	SettledGone      = "gone"
	SettledRemoved   = "removed"
	SettledDisabled  = "disabled"
	SettledMoved     = "moved"
)

// deliveryDelay answers how long to wait before the next attempt of a run that
// has already made the given number, and false once the run is spent.
func deliveryDelay(made int, spread float64) (time.Duration, bool) {
	if made < 0 || made >= DeliveryAttempts {
		return 0, false
	}
	agreed := deliveryDelays[made]
	return agreed + time.Duration(float64(agreed)*deliveryJitter*(spread*2-1)), true
}

// verdict is what one answer from an endpoint means for the work behind it.
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

// arrived is the verdict on an endpoint that took the event.
var arrived = verdict{Outcome: AttemptDelivered, Reason: SettledArrived}

// unreachable is the verdict on an endpoint Illarin never got an answer from.
var unreachable = verdict{
	Outcome: AttemptUnreachable, Detail: "Illarin could not reach it.", Retry: true,
}

// readAnswer turns what an endpoint said into what Illarin does next. A
// redirect counts as a wrong address rather than as somewhere to follow.
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

// stopped is the verdict on work Illarin will not send because the destination
// behind it is no longer somewhere it sends.
func stopped(reason string) verdict {
	return verdict{Outcome: AttemptRefused, Detail: whyStopped[reason], Reason: reason}
}

var whyStopped = map[string]string{
	SettledRemoved:  "The destination was removed.",
	SettledDisabled: "The destination was disabled.",
	SettledMoved:    "The destination moved to another address.",
	SettledGone:     "The destination answered 410 and receives nothing further.",
}

// spread is where inside the jitter window one attempt falls.
func spread() float64 { return rand.Float64() }
