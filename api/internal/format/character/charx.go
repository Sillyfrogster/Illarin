package character

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
)

const embeddedPrefix = "embeded://"

type CharXModule struct{}

func (CharXModule) ID() string { return CharX }

func (CharXModule) Declaration() format.Declaration { return declaration(CharX) }

func (CharXModule) OwnedSpecs() []string { return []string{V3} }

func (m CharXModule) Claim(file format.Inspection) (format.Claim, bool) {
	return format.ClaimByDeclaration(file, m.Declaration())
}

func (m CharXModule) Parse(
	ctx context.Context,
	file format.Inspection,
	claim format.Claim,
) (format.Parsed, error) {
	read, err := readCard(file, claim, 3, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	images := archivedImages(read, file)
	parsed, err := read.parsed(m.ID(), images)
	if err != nil {
		return format.Parsed{}, err
	}
	members, err := archivedMembers(ctx, file)
	if err != nil {
		return format.Parsed{}, err
	}
	parsed.Remainder = append(parsed.Remainder, members...)
	return parsed, nil
}

const (
	cardEntry = "card.json"
	// MemberNamespace marks a preserved file that came out of an archive.
	MemberNamespace        = "archive:"
	maxArchiveMemberBytes  = 1 << 20
	maxArchiveMembersBytes = 4 << 20
)

// archivedMembers keeps the archived files Illarin reads nothing from.
func archivedMembers(ctx context.Context, file format.Inspection) ([]format.Remainder, error) {
	pictures := make(map[string]bool, len(file.Images))
	for _, image := range file.Images {
		if image.Locator.Container == format.ZIP {
			pictures[image.Locator.Name] = true
		}
	}
	kept := make([]format.Remainder, 0)
	budget := uint64(maxArchiveMembersBytes)
	for _, entry := range file.ZIPEntries {
		if entry.Directory || entry.Name == cardEntry || pictures[entry.Name] {
			continue
		}
		if entry.UncompressedSize > maxArchiveMemberBytes || entry.UncompressedSize > budget {
			return nil, format.LimitExceeded(fmt.Errorf(
				"the archived %s is %d bytes, past what a card may carry beside it",
				entry.Name, entry.UncompressedSize,
			))
		}
		payload, err := readArchivedMember(ctx, file, entry.Name)
		if err != nil {
			return nil, err
		}
		budget -= entry.UncompressedSize
		kept = append(kept, format.Remainder{
			Owner:     format.OwnerWork,
			Namespace: MemberNamespace + entry.Name,
			Payload:   payload,
		})
	}
	return kept, nil
}

func readArchivedMember(
	ctx context.Context,
	file format.Inspection,
	name string,
) (json.RawMessage, error) {
	opened, err := file.OpenZIPEntry(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("open the archived %s: %w", name, err)
	}
	defer opened.Close()
	held, err := io.ReadAll(io.LimitReader(opened, maxArchiveMemberBytes))
	if err != nil {
		return nil, fmt.Errorf("read the archived %s: %w", name, err)
	}
	payload, err := json.Marshal(archivedMember{
		Bytes: base64.StdEncoding.EncodeToString(held),
	})
	if err != nil {
		return nil, fmt.Errorf("keep the archived %s: %w", name, err)
	}
	return payload, nil
}

type archivedMember struct {
	Bytes string `json:"bytes"`
}

// ArchivedMember reads back the bytes a preserved archived file holds.
func ArchivedMember(payload []byte) ([]byte, bool) {
	var held archivedMember
	if err := json.Unmarshal(payload, &held); err != nil {
		return nil, false
	}
	given, err := base64.StdEncoding.DecodeString(held.Bytes)
	if err != nil {
		return nil, false
	}
	return given, true
}

// ArchivedMemberName is the archive entry a preserved file goes back to.
func ArchivedMemberName(namespace string) (string, bool) {
	name, marked := strings.CutPrefix(namespace, MemberNamespace)
	if !marked || name == "" || name == cardEntry {
		return "", false
	}
	if path.IsAbs(name) || strings.Contains(name, "\\") || path.Clean(name) != name {
		return "", false
	}
	if name == ".." || strings.HasPrefix(name, "../") {
		return "", false
	}
	return name, true
}

type cardFile struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
}

// archivedImages routes every bundled image, naming the ones the card names.
func archivedImages(read card, file format.Inspection) []format.Media {
	var files []cardFile
	if raw, ok := read.fields["assets"]; ok {
		_ = json.Unmarshal(raw, &files)
	}
	named := make(map[uint32]bool)
	found := make([]format.Media, 0, len(file.Images))
	hasAvatar := false
	for _, entry := range files {
		path, embedded := strings.CutPrefix(entry.URI, embeddedPrefix)
		if !embedded {
			continue
		}
		image, located := archivedImage(file, path)
		if !located || named[image] {
			continue
		}
		named[image] = true
		role, wanted := cardFileRole(entry, hasAvatar)
		if !wanted {
			continue
		}
		if role == media.Avatar {
			hasAvatar = true
		}
		found = append(found, format.Media{
			Role: role, ImageID: image, ElementRole: elementRole(role), Name: entry.Name,
		})
	}
	for _, image := range file.Images {
		if image.Locator.Container != format.ZIP || named[image.ID] {
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

func cardFileRole(file cardFile, hasAvatar bool) (media.Role, bool) {
	switch file.Type {
	case "icon":
		if !hasAvatar && file.Name == "main" {
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

func archivedImage(file format.Inspection, path string) (uint32, bool) {
	wanted := strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "./")
	for _, image := range file.Images {
		if image.Locator.Container != format.ZIP {
			continue
		}
		if strings.TrimPrefix(image.Locator.Name, "./") == wanted {
			return image.ID, true
		}
	}
	return 0, false
}
