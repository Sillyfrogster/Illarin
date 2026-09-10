package character

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
)

const embeddedPrefix = "embeded://"

type CharXModule struct{}

func (CharXModule) ID() string { return CharX }

func (CharXModule) Declaration() format.Declaration { return declaration(CharX) }

func (CharXModule) OwnedSpecs() []string { return []string{V3} }

func (m CharXModule) Claim(file probe.Inspection) (format.Claim, bool) {
	return format.ClaimByDeclaration(file, m.Declaration())
}

func (m CharXModule) Parse(
	_ context.Context,
	file probe.Inspection,
	claim format.Claim,
) (format.Parsed, error) {
	read, err := readCard(file, claim, 3, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	return read.parsed(m.ID(), archivedImages(read, file))
}

type cardAsset struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
}

// archivedImages routes every bundled image, naming the ones the card names.
func archivedImages(read card, file probe.Inspection) []format.Media {
	var assets []cardAsset
	if raw, ok := read.fields["assets"]; ok {
		_ = json.Unmarshal(raw, &assets)
	}
	named := make(map[uint32]bool)
	found := make([]format.Media, 0, len(file.Images))
	hasAvatar := false
	for _, asset := range assets {
		path, embedded := strings.CutPrefix(asset.URI, embeddedPrefix)
		if !embedded {
			continue
		}
		image, located := archivedImage(file, path)
		if !located || named[image] {
			continue
		}
		named[image] = true
		role, wanted := assetRole(asset, hasAvatar)
		if !wanted {
			continue
		}
		if role == media.Avatar {
			hasAvatar = true
		}
		found = append(found, format.Media{
			Role: role, ImageID: image, ElementRole: elementRole(role), Name: asset.Name,
		})
	}
	for _, image := range file.Images {
		if image.Locator.Container != probe.ZIP || named[image.ID] {
			continue
		}
		role := archivedRole(image.Locator.Name, hasAvatar)
		if role == media.Avatar {
			hasAvatar = true
		}
		found = append(found, format.Media{
			Role: role, ImageID: image.ID, ElementRole: elementRole(role),
		})
	}
	return found
}

// archivedRole reads the layout the spec recommends, and guesses nothing beyond it.
func archivedRole(name string, hasAvatar bool) media.Role {
	folder := strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
	switch {
	case strings.HasPrefix(folder, "assets/icon/"):
		if hasAvatar {
			return media.AvatarAlt
		}
		return media.Avatar
	case strings.HasPrefix(folder, "assets/emotion/"):
		return media.Expression
	default:
		return media.Gallery
	}
}

func elementRole(role media.Role) block.Role {
	switch role {
	case media.Expression:
		return block.RoleExpressions
	case media.Gallery:
		return block.RoleGallery
	default:
		return ""
	}
}

func assetRole(asset cardAsset, hasAvatar bool) (media.Role, bool) {
	switch asset.Type {
	case "icon":
		if !hasAvatar && asset.Name == "main" {
			return media.Avatar, true
		}
		return media.AvatarAlt, true
	case "emotion":
		return media.Expression, true
	case "user_icon":
		return "", false
	default:
		return media.Gallery, true
	}
}

func archivedImage(file probe.Inspection, path string) (uint32, bool) {
	wanted := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, image := range file.Images {
		if image.Locator.Container != probe.ZIP {
			continue
		}
		if strings.TrimPrefix(image.Locator.Name, "./") == wanted {
			return image.ID, true
		}
	}
	return 0, false
}
