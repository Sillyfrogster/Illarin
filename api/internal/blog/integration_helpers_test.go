package blog_test

import "time"

type integrationChoice struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	State         string   `json:"state"`
	Announcements []string `json:"announcements"`
	Role          string   `json:"role"`
	ByDefault     bool     `json:"byDefault"`
}

type deliveryAttempt struct {
	Run         int       `json:"run"`
	Number      int       `json:"number"`
	Outcome     string    `json:"outcome"`
	Status      *int      `json:"status"`
	Detail      string    `json:"detail"`
	TookMs      int       `json:"tookMs"`
	AttemptedAt time.Time `json:"attemptedAt"`
}
