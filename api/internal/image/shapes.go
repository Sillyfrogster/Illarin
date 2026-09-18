package image

import "github.com/google/uuid"

type MediaRole string

const (
	MediaRoleAvatar           MediaRole = "avatar"
	MediaRoleAvatarAlt        MediaRole = "avatar_alt"
	MediaRoleExpression       MediaRole = "expression"
	MediaRoleGallery          MediaRole = "gallery"
	MediaRolePackItem         MediaRole = "pack_item"
	MediaRolePerspectiveLayer MediaRole = "perspective_layer"
)

type MediaList struct {
	Items []Media `json:"items"`
}

type GetMediaVariantParams struct {
	Expires   *string `json:"expires,omitempty"`
	Signature *string `json:"signature,omitempty"`
}

type Media struct {
	WorkId            uuid.UUID `json:"workId"`
	DerivativeVersion int       `json:"derivativeVersion"`
	Height            int       `json:"height"`
	Id                uuid.UUID `json:"id"`
	Role              MediaRole `json:"role"`
	Width             int       `json:"width"`
}
