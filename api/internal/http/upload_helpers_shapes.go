package http

type AddMediaRequest struct {
	Role AddMediaRequestRole `json:"role"`
}

type AddMediaRequestRole string

const (
	AddMediaRequestRoleAvatar           AddMediaRequestRole = "avatar"
	AddMediaRequestRoleAvatarAlt        AddMediaRequestRole = "avatar_alt"
	AddMediaRequestRoleExpression       AddMediaRequestRole = "expression"
	AddMediaRequestRoleGallery          AddMediaRequestRole = "gallery"
	AddMediaRequestRolePackItem         AddMediaRequestRole = "pack_item"
	AddMediaRequestRolePerspectiveLayer AddMediaRequestRole = "perspective_layer"
)
