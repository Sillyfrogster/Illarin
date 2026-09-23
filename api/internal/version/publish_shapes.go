package version

import (
	"time"

	"github.com/google/uuid"
)

type WorkVersion struct {
	ContentChanged bool      `json:"contentChanged"`
	Id             uuid.UUID `json:"id"`
	Notes          string    `json:"notes"`
	Number         int       `json:"number"`
	RecordedAt     time.Time `json:"recordedAt"`
	Summary        string    `json:"summary"`
	VersionLabel   string    `json:"versionLabel"`
}

type WorkVersionRequest struct {
	Discord      *bool   `json:"discord,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	Notify       *bool   `json:"notify,omitempty"`
	Summary      string  `json:"summary"`
	VersionLabel *string `json:"versionLabel,omitempty"`
}
