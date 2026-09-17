package profile

import "github.com/google/uuid"

type ProfileLink struct {
	Address string `json:"address"`
	Label   string `json:"label"`
}

type SaveProfileRequest struct {
	Biography    string        `json:"biography"`
	ContactEmail string        `json:"contactEmail"`
	DisplayName  string        `json:"displayName"`
	Links        []ProfileLink `json:"links"`
}

type Profile struct {
	Avatar       *ProfileAvatar `json:"avatar,omitempty"`
	Biography    string         `json:"biography"`
	ContactEmail string         `json:"contactEmail"`
	DisplayName  string         `json:"displayName"`
	Handle       string         `json:"handle"`
	Id           uuid.UUID      `json:"id"`
	Links        []ProfileLink  `json:"links"`
	Restricted   bool           `json:"restricted"`
}

type ProfileAvatar struct {
	Height int    `json:"height"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
}
