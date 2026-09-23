package upload

import (
	"github.com/google/uuid"
)

type PlaceFoundImageRequest struct {
	MediaId *uuid.UUID `json:"mediaId,omitempty"`
}

type FoundImage struct {
	Address string     `json:"address"`
	BlockId *uuid.UUID `json:"blockId,omitempty"`
	Id      uuid.UUID  `json:"id"`
	Media   *struct {
		Height   int       `json:"height"`
		Id       uuid.UUID `json:"id"`
		ThumbUrl string    `json:"thumbUrl"`
		Width    int       `json:"width"`
	} `json:"media,omitempty"`
	Name    string `json:"name"`
	Section string `json:"section"`
}

type FoundImageList struct {
	Pictures []FoundImage `json:"pictures"`
}
