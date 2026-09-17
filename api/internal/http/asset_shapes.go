package http

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

type WithholdAssetRequest struct {
	Reason string `json:"reason"`
}

type WorkingCopyVersion = int64

type GetMediaVariantParams struct {
	Expires   *string `json:"expires,omitempty"`
	Signature *string `json:"signature,omitempty"`
}

type Media struct {
	AssetId           uuid.UUID `json:"assetId"`
	DerivativeVersion int       `json:"derivativeVersion"`
	Height            int       `json:"height"`
	Id                uuid.UUID `json:"id"`
	Role              MediaRole `json:"role"`
	Width             int       `json:"width"`
}
