package http

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
