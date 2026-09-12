package publication

import "github.com/Sillyfrogster/Illarin/api/internal/outbox"

var DeliveryDelays = outbox.Delays

var DeliveryAttempts = outbox.Attempts

const (
	SettledArrived     = outbox.Arrived
	SettledExhausted   = outbox.Exhausted
	SettledRefused     = outbox.Refused
	SettledGone        = outbox.Gone
	SettledRemoved     = outbox.Removed
	SettledDisabled    = outbox.Disabled
	SettledMoved       = outbox.Moved
	SettledUnconfirmed = outbox.Unconfirmed
)
