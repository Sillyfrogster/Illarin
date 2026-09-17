package blog_test

import "time"

type destinationChoice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	State     string   `json:"state"`
	Events    []string `json:"events"`
	Role      string   `json:"role"`
	ByDefault bool     `json:"byDefault"`
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
