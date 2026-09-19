package blog

import "github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"

var TryDelays = dispatch.Delays

var MaxTries = dispatch.MaxTries

const (
	SettledArrived     = dispatch.Arrived
	SettledExhausted   = dispatch.Exhausted
	SettledRefused     = dispatch.Refused
	SettledGone        = dispatch.Gone
	SettledRemoved     = dispatch.Removed
	SettledDisabled    = dispatch.Disabled
	SettledMoved       = dispatch.Moved
	SettledUnconfirmed = dispatch.Unconfirmed
)
