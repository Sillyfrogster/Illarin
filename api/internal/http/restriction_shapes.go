package http

import (
	"time"
)

type ProfileRestriction struct {
	Reason       string    `json:"reason"`
	RestrictedAt time.Time `json:"restrictedAt"`
	RestrictedBy *string   `json:"restrictedBy,omitempty"`
}

type RestrictProfileRequest struct {
	Reason string `json:"reason"`
}
