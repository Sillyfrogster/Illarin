package connect

import (
	"errors"

	"github.com/google/uuid"
)

// ErrNotDeliverable says the work cannot be sent to an instance
var ErrNotDeliverable = errors.New("that work cannot be sent to an instance")

// DeliveryTarget is one format the work can be written in for a send
type DeliveryTarget struct {
	Format string
	Label  string
}

// DeliveryPicture is one image that travels with the work
type DeliveryPicture struct {
	MediaID uuid.UUID
	Role    string
	IsCover bool
	URL     string
}

// Deliverable is what a send carries about the work it is sending
type Deliverable struct {
	Type              string
	Name              string
	ContentGeneration int
	Targets           []DeliveryTarget
	HasOriginal       bool
	Pictures          []DeliveryPicture
	// InstallCapabilities lists what an instance must declare, any one of them, before the work is sent to it.
	InstallCapabilities []string
}
