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

// Report covers the last 30 complete days and names the works downloaded most
type Report struct {
	From     string       `json:"from"`
	Through  string       `json:"through"`
	Days     []ReportDay  `json:"days"`
	TopWorks []ReportWork `json:"topWorks"`
}

type ReportDay struct {
	Day       string `json:"day"`
	Visits    int    `json:"visits"`
	Downloads int    `json:"downloads"`
	Sends     int    `json:"sends"`
	SignUps   int    `json:"signUps"`
	Publishes int    `json:"publishes"`
}

type ReportWork struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Downloads int    `json:"downloads"`
}
