package staff

import "time"

type RestrictedProfile struct {
	Reason       string    `json:"reason"`
	RestrictedAt time.Time `json:"restrictedAt"`
	RestrictedBy *string   `json:"restrictedBy,omitempty"`
}

type RestrictProfileRequest struct {
	Reason string `json:"reason"`
}

type TakeDownWorkRequest struct {
	Reason string `json:"reason"`
}
