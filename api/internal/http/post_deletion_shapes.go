package http

import (
	"time"
)

type PostDeletion struct {
	At    time.Time `json:"at"`
	By    string    `json:"by"`
	Until time.Time `json:"until"`
}
