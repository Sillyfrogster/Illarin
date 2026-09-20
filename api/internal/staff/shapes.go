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

// Report covers the last 30 complete days, the 30 before them as one total each, and the works downloaded most
type Report struct {
	From     string       `json:"from"`
	Through  string       `json:"through"`
	Days     []ReportDay  `json:"days"`
	Previous ReportTotals `json:"previous"`
	TopWorks []ReportWork `json:"topWorks"`
}

type ReportTotals struct {
	Visits    int `json:"visits"`
	Downloads int `json:"downloads"`
	Sends     int `json:"sends"`
	SignUps   int `json:"signUps"`
	Publishes int `json:"publishes"`
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
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Cover     *string `json:"cover,omitempty"`
	Downloads int     `json:"downloads"`
}
