package image

import (
	"io"
	"mime/multipart"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

func readMediaMetadata(parts *multipart.Reader) (AddMediaRequest, error) {
	part, err := api.NextPart(parts, api.MetadataPart)
	if err != nil {
		return AddMediaRequest{}, err
	}
	var metadata AddMediaRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return AddMediaRequest{}, api.FormRefusal{
			Reason: "the " + api.MetadataPart + " part is not valid JSON",
			Cause:  err,
		}
	}
	return metadata, nil
}

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
