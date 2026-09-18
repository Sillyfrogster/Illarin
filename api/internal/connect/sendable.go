package connect

import (
	"errors"

	"github.com/google/uuid"
)

// ErrNotDeliverable says the asset cannot be sent to an instance
var ErrNotDeliverable = errors.New("that asset cannot be sent to an instance")

// DeliveryTarget is one format the asset can be written in for a send
type DeliveryTarget struct {
	Format string
	Label  string
}

// DeliveryPicture is one image that travels with the asset
type DeliveryPicture struct {
	MediaID uuid.UUID
	Role    string
	IsCover bool
	URL     string
}

// Deliverable is what a send carries about the asset it is sending
type Deliverable struct {
	Kind              string
	Name              string
	ContentGeneration int
	Targets           []DeliveryTarget
	HasOriginal       bool
	Pictures          []DeliveryPicture
	// InstallCapabilities lists what an instance must declare, any one of them, before the asset is sent to it.
	InstallCapabilities []string
}
