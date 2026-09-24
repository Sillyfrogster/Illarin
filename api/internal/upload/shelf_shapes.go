package upload

import (
	"time"

	"github.com/google/uuid"
)

type ShelfPieceKind string

const (
	ShelfPieceKindSection ShelfPieceKind = "section"
	ShelfPieceKindPicture ShelfPieceKind = "picture"
)

type ShelfSource string

const (
	ShelfSourceReadme ShelfSource = "readme"
	ShelfSourcePasted ShelfSource = "pasted"
)

type AddMarkdownRequest struct {
	Markdown string `json:"markdown"`
}

type PlaceShelfPieceRequest struct {
	MediaId   *uuid.UUID `json:"mediaId,omitempty"`
	Position  *int       `json:"position,omitempty"`
	ElementId *uuid.UUID `json:"elementId,omitempty"`
}

type ShelfPiece struct {
	Id      uuid.UUID      `json:"id"`
	Kind    ShelfPieceKind `json:"kind"`
	Section string         `json:"section"`
	Text    string         `json:"text,omitempty"`
	Address string         `json:"address,omitempty"`
	Name    string         `json:"name,omitempty"`
	BlockId *uuid.UUID     `json:"blockId,omitempty"`
	Media   *struct {
		Height   int       `json:"height"`
		Id       uuid.UUID `json:"id"`
		ThumbUrl string    `json:"thumbUrl"`
		Width    int       `json:"width"`
	} `json:"media,omitempty"`
}

type ShelfImport struct {
	Id        uuid.UUID    `json:"id"`
	Source    ShelfSource  `json:"source"`
	Title     string       `json:"title"`
	CreatedAt time.Time    `json:"createdAt"`
	Pieces    []ShelfPiece `json:"pieces"`
}

type Shelf struct {
	Imports []ShelfImport `json:"imports"`
}
