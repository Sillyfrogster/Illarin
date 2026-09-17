package http

import (
	"time"

	"github.com/google/uuid"
)

type AssetIdentityRequest struct {
	Blurb  string `json:"blurb"`
	IsNsfw *bool  `json:"isNsfw" tstype:"boolean | null,required"`
	Name   string `json:"name"`
}

type AssetUpdate struct {
	ContentChanged    bool      `json:"contentChanged"`
	ContentGeneration int       `json:"contentGeneration"`
	Id                uuid.UUID `json:"id"`
	Notes             string    `json:"notes"`
	Number            int       `json:"number"`
	RecordedAt        time.Time `json:"recordedAt"`
	Summary           string    `json:"summary"`
	VersionLabel      string    `json:"versionLabel"`
}

type AssetUpdateRequest struct {
	AnnounceUnlisted *bool        `json:"announceUnlisted,omitempty"`
	DestinationIds   *[]uuid.UUID `json:"destinationIds,omitempty"`
	Notes            *string      `json:"notes,omitempty"`
	Notify           *bool        `json:"notify,omitempty"`
	Summary          string       `json:"summary"`
	VersionLabel     *string      `json:"versionLabel,omitempty"`
}

type PublishRefusal struct {
	Code      *PublishRefusalCode `json:"code,omitempty"`
	Error     string              `json:"error"`
	Readiness *[]ReadinessItem    `json:"readiness,omitempty"`
}

type PublishRefusalCode string

const (
	PublishRefusalCodeAlreadyPublished PublishRefusalCode = "already_published"
	PublishRefusalCodeNoChanges        PublishRefusalCode = "no_changes"
	PublishRefusalCodeNotReady         PublishRefusalCode = "not_ready"
)

type ReadinessItem struct {
	BlockId *uuid.UUID `json:"blockId,omitempty"`
	Detail  string     `json:"detail"`
	Id      string     `json:"id"`
	Label   string     `json:"label"`
	Met     bool       `json:"met"`
}
