package http

import (
	"time"

	"github.com/google/uuid"
)

type PostSchedule struct {
	At             time.Time         `json:"at"`
	CreatedAt      time.Time         `json:"createdAt"`
	CreatedBy      string            `json:"createdBy"`
	Id             uuid.UUID         `json:"id"`
	RevisionId     uuid.UUID         `json:"revisionId"`
	RevisionNumber int               `json:"revisionNumber"`
	State          PostScheduleState `json:"state"`
	StoppedBecause *string           `json:"stoppedBecause,omitempty"`
}

type PostScheduleState string

const (
	PostScheduleStateCancelled  PostScheduleState = "cancelled"
	PostScheduleStatePending    PostScheduleState = "pending"
	PostScheduleStatePublished  PostScheduleState = "published"
	PostScheduleStatePublishing PostScheduleState = "publishing"
	PostScheduleStateStopped    PostScheduleState = "stopped"
)

type ReplacePostScheduleRequest struct {
	At                 time.Time    `json:"at"`
	DestinationIds     *[]uuid.UUID `json:"destinationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RevisionId         uuid.UUID    `json:"revisionId"`
	RoleDestinationIds *[]uuid.UUID `json:"roleDestinationIds,omitempty"`
}

type SchedulePostRequest struct {
	At                 time.Time    `json:"at"`
	DestinationIds     *[]uuid.UUID `json:"destinationIds,omitempty"`
	Note               *string      `json:"note,omitempty"`
	RoleDestinationIds *[]uuid.UUID `json:"roleDestinationIds,omitempty"`
	Version            int          `json:"version"`
}
