package profile

import (
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/google/uuid"
)

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

type SaveFeaturedRequest struct {
	WorkIds []uuid.UUID `json:"workIds"`
}

type Profile struct {
	Avatar         *ProfilePicture   `json:"avatar,omitempty"`
	Banner         *ProfilePicture   `json:"banner,omitempty"`
	Biography      string            `json:"biography"`
	ContactEmail   string            `json:"contactEmail"`
	DisplayName    string            `json:"displayName"`
	Featured       []page.BrowseWork `json:"featured"`
	Followers      int               `json:"followers"`
	Following      *bool             `json:"following,omitempty"`
	Handle         string            `json:"handle"`
	Id             uuid.UUID         `json:"id"`
	IsOwner        bool              `json:"isOwner"`
	Links          []ProfileLink     `json:"links"`
	RecentVersions []RecentVersion   `json:"recentVersions"`
	Restricted     bool              `json:"restricted"`
	Tint           string            `json:"tint"`
	Works          int               `json:"works"`
}

type ProfilePicture struct {
	Height int    `json:"height"`
	Url    string `json:"url"`
	Width  int    `json:"width"`
}

type RecentVersion struct {
	Cover        *string   `json:"cover" tstype:"string | null,required"`
	Initial      bool      `json:"initial"`
	Number       int       `json:"number"`
	RecordedAt   time.Time `json:"recordedAt"`
	Summary      string    `json:"summary"`
	VersionLabel string    `json:"versionLabel"`
	WorkId       uuid.UUID `json:"workId"`
	WorkName     string    `json:"workName"`
	WorkType     string    `json:"workType"`
}

type CreatorFollow struct {
	Followers int  `json:"followers"`
	Following bool `json:"following"`
}
