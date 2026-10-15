package character

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
)

const embeddedPrefix = "embeded://"

type CharXModule struct{}

func (CharXModule) ID() string { return CharX }

func (CharXModule) Declaration() format.Declaration { return declaration(CharX) }

func (CharXModule) OwnedSpecs() []string { return []string{V3} }

func (m CharXModule) Match(file format.Inspection) (format.Match, bool) {
	return format.MatchByDeclaration(file, m.Declaration())
}

func (m CharXModule) Parse(
	ctx context.Context,
	file format.Inspection,
	match format.Match,
) (format.Parsed, error) {
	read, err := readCard(file, match, 3, m.ID())
	if err != nil {
		return format.Parsed{}, err
	}
	lumiverse, err := readLumiverseNames(ctx, file)
	if err != nil {
		return format.Parsed{}, err
	}
	pictures := archivedImages(read, file, lumiverse)
	parsed, err := read.parsed(m.ID(), pictures)
	if err != nil {
		return format.Parsed{}, err
	}
	members, err := archivedMembers(ctx, file, pictures)
	if err != nil {
		return format.Parsed{}, err
	}
	parsed.Remainder = append(parsed.Remainder, members...)
	return parsed, nil
}

const (
	cardEntry = "card.json"
	// MemberNamespace marks a preserved file that came out of an archive.
	MemberNamespace = "archive:"
)

// archivedMembers keeps every archived file except the pictures Illarin read into the work.
func archivedMembers(
	ctx context.Context,
	file format.Inspection,
	read []format.Media,
) ([]format.Remainder, error) {
	taken := make(map[uint32]bool, len(read))
	for _, picture := range read {
		taken[picture.ImageID] = true
	}
	pictures := make(map[string]bool, len(read))
	for _, image := range file.Images {
		if image.Location.Container == format.ZIP && taken[image.ID] {
			pictures[image.Location.Name] = true
		}
	}
	kept := make([]format.Remainder, 0)
	for _, entry := range file.ZIPEntries {
		if entry.Directory || entry.Name == cardEntry || pictures[entry.Name] {
			continue
		}
		payload, err := readArchivedMember(ctx, file, entry.Name)
		if err != nil {
			return nil, err
		}
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
	held, err := io.ReadAll(opened)
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

const lumiverseModulesEntry = "lumiverse_modules.json"

type lumiverseNames struct {
	// expressions holds each expression's label by archive path.
	expressions map[string]string
	// untouched holds the alternate avatars and perspective layers, which only Lumiverse reads.
	untouched map[string]bool
}

// readLumiverseNames reads the pictures Lumiverse lists outside card.json, by archive path.
func readLumiverseNames(ctx context.Context, file format.Inspection) (lumiverseNames, error) {
	named := lumiverseNames{expressions: make(map[string]string), untouched: make(map[string]bool)}
	if !slices.ContainsFunc(file.ZIPEntries, func(entry format.ZIPEntry) bool {
		return entry.Name == lumiverseModulesEntry
	}) {
		return named, nil
	}
	opened, err := file.OpenZIPEntry(ctx, lumiverseModulesEntry)
	if err != nil {
		return named, fmt.Errorf("open the archived %s: %w", lumiverseModulesEntry, err)
	}
	defer opened.Close()
	var modules struct {
		Expressions struct {
			Mappings map[string]string `json:"mappings"`
		} `json:"expressions"`
		ExpressionGroups struct {
			Groups map[string]map[string]string `json:"groups"`
		} `json:"expression_groups"`
		AlternateAvatars []struct {
			Path string `json:"path"`
		} `json:"alternate_avatars"`
		PerspectiveLayers []struct {
			Path string `json:"path"`
		} `json:"landing_perspective_layers"`
	}
	if json.NewDecoder(opened).Decode(&modules) != nil {
		return named, nil
	}
	for label, entry := range modules.Expressions.Mappings {
		named.expressions[archivePath(entry)] = label
	}
	for group, labels := range modules.ExpressionGroups.Groups {
		for label, entry := range labels {
			named.expressions[archivePath(entry)] = group + "/" + label
		}
	}
	for _, avatar := range modules.AlternateAvatars {
		named.untouched[archivePath(avatar.Path)] = true
	}
	for _, layer := range modules.PerspectiveLayers {
		named.untouched[archivePath(layer.Path)] = true
	}
	return named, nil
}

// lumiverseModules lists the written expressions in the file Lumiverse reads them from
func lumiverseModules(records []cardFileRecord, preserved []format.Remainder) (archivedFile, bool) {
	mappings := make(map[string]string)
	groups := make(map[string]map[string]string)
	for _, record := range records {
		if record.Type != emotionFileType {
			continue
		}
		entry := strings.TrimPrefix(record.URI, embeddedPrefix)
		group, label, grouped := strings.Cut(record.Name, "/")
		if !grouped {
			mappings[record.Name] = entry
			continue
		}
		if groups[group] == nil {
			groups[group] = make(map[string]string)
		}
		groups[group][label] = entry
	}
	if len(mappings) == 0 && len(groups) == 0 {
		return archivedFile{}, false
	}
	modules := map[string]json.RawMessage{"version": keys.Must(1)}
	var kept struct {
		Expressions struct {
			Enabled           *bool  `json:"enabled"`
			DefaultExpression string `json:"defaultExpression"`
		} `json:"expressions"`
	}
	for _, row := range preserved {
		if name, archived := ArchivedMemberName(row.Namespace); archived && name == lumiverseModulesEntry {
			if data, readable := ArchivedMember(row.Payload); readable {
				_ = json.Unmarshal(data, &modules)
				_ = json.Unmarshal(data, &kept)
			}
		}
	}
	delete(modules, "expressions")
	delete(modules, "expression_groups")
	if len(mappings) > 0 {
		enabled := kept.Expressions.Enabled == nil || *kept.Expressions.Enabled
		fallback := kept.Expressions.DefaultExpression
		if _, mapped := mappings[fallback]; !mapped {
			fallback = ""
		}
		modules["expressions"] = keys.Must(map[string]any{
			"enabled": enabled, "defaultExpression": fallback, "mappings": mappings,
		})
	}
	if len(groups) > 0 {
		modules["expression_groups"] = keys.Must(map[string]any{"groups": groups})
	}
	return archivedFile{path: lumiverseModulesEntry, data: keys.Must(modules)}, true
}

func archivePath(name string) string {
	return strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
}

// archivedImages reads the bundled pictures a work shows, and leaves the extra icons and Lumiverse's own pictures in the archive.
func archivedImages(read card, file format.Inspection, lumiverse lumiverseNames) []format.Media {
	var files []cardFile
	if raw, ok := read.fields["assets"]; ok {
		_ = json.Unmarshal(raw, &files)
	}
	named := make(map[uint32]bool)
	found := make([]format.Media, 0, len(file.Images))
	hasAvatar := false
	spareIcons := make([]uint32, 0)
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
		if lumiverse.untouched[archivePath(path)] {
			continue
		}
		if entry.Type == iconFileType && (hasAvatar || entry.Name != mainIconFileName) {
			spareIcons = append(spareIcons, image)
			continue
		}
		role, wanted := cardFileRole(entry)
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
		path := archivePath(image.Location.Name)
		if image.Location.Container != format.ZIP || named[image.ID] || lumiverse.untouched[path] {
			continue
		}
		role := archivedRole(path)
		name, listed := lumiverse.expressions[path]
		if listed {
			role = media.Expression
		}
		if role == media.Avatar {
			if hasAvatar {
				continue
			}
			hasAvatar = true
		}
		found = append(found, format.Media{
			Role: role, ImageID: image.ID, ElementRole: elementRole(role), Name: name,
		})
	}
	if !hasAvatar && len(spareIcons) > 0 {
		found = append(found, format.Media{Role: media.Avatar, ImageID: spareIcons[0]})
	}
	return found
}

// archivedRole reads the layout the spec recommends, and guesses nothing beyond it.
func archivedRole(folder string) media.Role {
	switch {
	case strings.HasPrefix(folder, "assets/icon/"):
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

func cardFileRole(file cardFile) (media.Role, bool) {
	switch file.Type {
	case iconFileType:
		return media.Avatar, true
	case "emotion":
		return media.Expression, true
	case "user_icon":
		return "", false
	default:
		return media.Gallery, true
	}
}

func archivedImage(file format.Inspection, path string) (uint32, bool) {
	wanted := archivePath(path)
	for _, image := range file.Images {
		if image.Location.Container != format.ZIP {
			continue
		}
		if strings.TrimPrefix(image.Location.Name, "./") == wanted {
			return image.ID, true
		}
	}
	return 0, false
}
