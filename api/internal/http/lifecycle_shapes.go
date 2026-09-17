package http

import (
	"time"

	"github.com/google/uuid"
)

type AssetUpdate struct {
	ContentChanged    bool      `json:"contentChanged"`
	ContentGeneration int       `json:"contentGeneration"`
	Id                uuid.UUID `json:"id"`
	Notes             string    `json:"notes"`
	Number            int       `json:"number"`
	RecordedAt        time.Time `json:"recordedAt"`
	Summary           string    `json:"summary"`
	VersionLabel      string    `json:"versionLabel"`
}

type AssetUpdateRequest struct {
	AnnounceUnlisted *bool        `json:"announceUnlisted,omitempty"`
	DestinationIds   *[]uuid.UUID `json:"destinationIds,omitempty"`
	Notes            *string      `json:"notes,omitempty"`
	Notify           *bool        `json:"notify,omitempty"`
	Summary          string       `json:"summary"`
	VersionLabel     *string      `json:"versionLabel,omitempty"`
}
