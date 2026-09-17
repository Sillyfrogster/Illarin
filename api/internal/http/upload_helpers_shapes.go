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

type AddPostMediaRequest struct {
	Purpose PostMediaPurpose `json:"purpose"`
}

type CreateAssetRequest struct {
	Blurb     *string                      `json:"blurb,omitempty"`
	Confirmed bool                         `json:"confirmed"`
	Discovery *CreateAssetRequestDiscovery `json:"discovery,omitempty"`
	IsNsfw    *bool                        `json:"isNsfw,omitempty"`
	Name      *string                      `json:"name,omitempty"`
	Tags      *[]string                    `json:"tags,omitempty"`
}

type CreateAssetRequestDiscovery string

const (
	CreateAssetRequestDiscoveryListed   CreateAssetRequestDiscovery = "listed"
	CreateAssetRequestDiscoveryUnlisted CreateAssetRequestDiscovery = "unlisted"
)
