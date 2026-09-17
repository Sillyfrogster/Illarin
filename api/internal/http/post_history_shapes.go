package http

import (
	"time"

	"github.com/google/uuid"
)

type PostAction struct {
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	After      *string   `json:"after,omitempty"`
	At         time.Time `json:"at"`
	Before     *string   `json:"before,omitempty"`
	Credential string    `json:"credential"`
	Id         uuid.UUID `json:"id"`
	Revision   *int      `json:"revision,omitempty"`
}

type PostActionList struct {
	Actions []PostAction `json:"actions"`
}

type PostRevision struct {
	CapturedAt  time.Time           `json:"capturedAt"`
	CapturedBy  string              `json:"capturedBy"`
	CapturedFor PostRevisionReason  `json:"capturedFor"`
	Category    PublicationCategory `json:"category"`
	Id          uuid.UUID           `json:"id"`
	Number      int                 `json:"number"`
	Public      bool                `json:"public"`
	Slug        string              `json:"slug"`
	Summary     string              `json:"summary"`
	Title       string              `json:"title"`
}

type PostRevisionList struct {
	Revisions []PostRevision `json:"revisions"`
}

type PostRevisionReason string

const (
	Checkpoint  PostRevisionReason = "checkpoint"
	Publication PostRevisionReason = "publication"
	Schedule    PostRevisionReason = "schedule"
)

type PostVersionRequest struct {
	Version int `json:"version"`
}
