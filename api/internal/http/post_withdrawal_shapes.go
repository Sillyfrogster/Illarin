package http

import (
	"time"

	"github.com/google/uuid"
)

type PostWithdrawal struct {
	At          time.Time `json:"at"`
	By          string    `json:"by"`
	Explanation string    `json:"explanation"`
	Reason      string    `json:"reason"`
}

type RepublishPostRequest struct {
	DestinationIds *[]uuid.UUID `json:"destinationIds,omitempty"`
	Note           *string      `json:"note,omitempty"`
	RevisionId     uuid.UUID    `json:"revisionId"`
	Version        int          `json:"version"`
}

type WithdrawPostRequest struct {
	DestinationIds *[]uuid.UUID `json:"destinationIds,omitempty"`
	Explanation    *string      `json:"explanation,omitempty"`
	Note           *string      `json:"note,omitempty"`
	Reason         string       `json:"reason"`
	Version        int          `json:"version"`
}

type WithdrawnPost struct {
	Explanation string `json:"explanation"`
	Slug        string `json:"slug"`
}
