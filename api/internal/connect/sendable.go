package connect

import (
	"errors"

	"github.com/google/uuid"
)

// ErrNotSendable says the work cannot be sent to a connected app
var ErrNotSendable = errors.New("that work cannot be sent to a connected app")

// SendFormat is one format the work can be written in for a send
type SendFormat struct {
	Format string
	Label  string
}

// SendPicture is one image that travels with the work
type SendPicture struct {
	MediaID uuid.UUID
	Role    string
	IsCover bool
	URL     string
}

// Sendable is what a send carries about the work it is sending
type Sendable struct {
	Type          string
	Name          string
	VersionNumber int
	Formats       []SendFormat
	HasOriginal   bool
	Pictures      []SendPicture
	// InstallCapabilities lists what a connected app must declare, any one of them, before the work is sent to it.
	InstallCapabilities []string
}
