// Package dispatch sends announcements and retries failed attempts
package dispatch

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"regexp"
	"time"
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

var MaxTries = len(Delays)

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

// Delay says how long to wait before the try numbered made, counting from zero
func Delay(made int, spread float64) (time.Duration, bool) {
	if made < 0 || made >= MaxTries {
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
	Outcome: OutcomeUnreachable, Detail: "Illarin could not reach the integration.", Retry: true,
}

var DiscordUnconfirmed = Verdict{
	Outcome: OutcomeUnconfirmed, Reason: Unconfirmed,
	Detail: "Discord may have posted this announcement, but did not confirm it. Check the channel before sending again; another send may create a duplicate.",
}

// ReadAnswer turns an endpoint's response into what happens to the attempt next.
func ReadAnswer(answer Answer) Verdict {
	said := fmt.Sprintf("It answered %d.", answer.Status)
	switch {
	case answer.Status >= http.StatusOK && answer.Status < http.StatusMultipleChoices:
		return ArrivedVerdict
	case answer.Status == http.StatusTooManyRequests:
		return Verdict{
			Outcome: OutcomeRefused, Detail: "The integration asked Illarin to retry later.",
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
func ReadAnnouncement(answer Answer) (Verdict, string) {
	if answer.Status == http.StatusNotFound {
		return Verdict{Outcome: OutcomeRefused, Reason: Disabled, Gone: true,
			Detail: "The Discord webhook is missing. This integration has been disabled."}, ""
	}
	if answer.Status >= http.StatusInternalServerError || answer.Status == http.StatusRequestTimeout {
		return DiscordUnconfirmed, ""
	}
	if answer.Status < http.StatusOK || answer.Status >= http.StatusMultipleChoices {
		return ReadAnswer(answer), ""
	}
	message := MessageID(answer.Body)
	if message == "" {
		return DiscordUnconfirmed, ""
	}
	return ArrivedVerdict, message
}

// Stopped is the verdict for work Illarin ends because of the integration.
func Stopped(reason string) Verdict {
	return Cancelled(reason, whyStopped[reason])
}

// Cancelled is the verdict for work Illarin ends for its own reason.
func Cancelled(reason, detail string) Verdict {
	return Verdict{Outcome: OutcomeRefused, Detail: detail, Reason: reason}
}

var whyStopped = map[string]string{
	Removed:  "The integration was removed.",
	Disabled: "The integration was disabled.",
	Moved:    "The integration moved to another address.",
	Gone:     "The integration no longer exists and receives no further announcements.",
}

func MessageID(body []byte) string {
	var message struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &message) != nil || !messageSnowflake.MatchString(message.ID) {
		return ""
	}
	return message.ID
}

var messageSnowflake = regexp.MustCompile(`^[0-9]{17,20}$`)
