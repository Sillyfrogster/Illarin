// Package outbox keeps announcement work durable and retries it on one shared schedule.
package outbox

import (
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/discord"
	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
)

// Delays is the gap before each attempt, the first being immediate.
var Delays = []time.Duration{
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

var Attempts = len(Delays)

const jitter = 0.1

const (
	Pending            = "pending"
	Sending            = "sending"
	Delivered          = "delivered"
	Failed             = "failed"
	Unconfirmed        = "unconfirmed"
	OutcomeDelivered   = "delivered"
	OutcomeRefused     = "refused"
	OutcomeUnreachable = "unreachable"
	OutcomeUnconfirmed = "unconfirmed"
)

const (
	Arrived   = "arrived"
	Exhausted = "exhausted"
	Refused   = "refused"
	Gone      = "gone"
	Removed   = "removed"
	Disabled  = "disabled"
	Moved     = "moved"
)

// Delay says how long to wait before the attempt numbered made, counting from zero.
func Delay(made int, spread float64) (time.Duration, bool) {
	if made < 0 || made >= Attempts {
		return 0, false
	}
	agreed := Delays[made]
	return agreed + time.Duration(float64(agreed)*jitter*(spread*2-1)), true
}

func Spread() float64 { return rand.Float64() }

type Verdict struct {
	Outcome string
	Detail  string
	Status  *int
	Took    time.Duration
	Retry   bool
	After   time.Duration
	Reason  string
	Gone    bool
}

var ArrivedVerdict = Verdict{Outcome: OutcomeDelivered, Reason: Arrived}

var Unreachable = Verdict{
	Outcome: OutcomeUnreachable, Detail: "Illarin could not reach the destination.", Retry: true,
}

// ReadAnswer turns an endpoint's response into what happens to the delivery next.
func ReadAnswer(answer outbound.Answer) Verdict {
	said := fmt.Sprintf("It answered %d.", answer.Status)
	switch {
	case answer.Status >= http.StatusOK && answer.Status < http.StatusMultipleChoices:
		return ArrivedVerdict
	case answer.Status == http.StatusTooManyRequests:
		return Verdict{
			Outcome: OutcomeRefused, Detail: "The destination asked Illarin to retry later.",
			Retry: true, After: answer.RetryAfter,
		}
	case answer.Status == http.StatusGone:
		return Verdict{
			Outcome: OutcomeRefused,
			Detail:  "It answered 410, so nothing is sent there again.",
			Reason:  Gone, Gone: true,
		}
	case answer.Status == http.StatusRequestTimeout,
		answer.Status == http.StatusTooEarly,
		answer.Status >= http.StatusInternalServerError:
		return Verdict{Outcome: OutcomeRefused, Detail: said, Retry: true}
	default:
		return Verdict{Outcome: OutcomeRefused, Detail: said, Reason: Refused}
	}
}

// ReadAnnouncement reads a confirming Discord send, returning the message it made.
func ReadAnnouncement(answer outbound.Answer) (Verdict, string) {
	if answer.Status < http.StatusOK || answer.Status >= http.StatusMultipleChoices {
		return ReadAnswer(answer), ""
	}
	message := discord.MessageID(answer.Body)
	if message == "" {
		return Verdict{
			Outcome: OutcomeUnconfirmed,
			Detail:  "Discord accepted the request without returning a message ID.",
			Reason:  Unconfirmed,
		}, ""
	}
	return ArrivedVerdict, message
}

// Stopped is the verdict for work Illarin ends because of the destination.
func Stopped(reason string) Verdict {
	return Cancelled(reason, whyStopped[reason])
}

// Cancelled is the verdict for work Illarin ends for its own reason.
func Cancelled(reason, detail string) Verdict {
	return Verdict{Outcome: OutcomeRefused, Detail: detail, Reason: reason}
}

var whyStopped = map[string]string{
	Removed:  "The destination was removed.",
	Disabled: "The destination was disabled.",
	Moved:    "The destination moved to another address.",
	Gone:     "The destination answered 410 and receives nothing further.",
}
