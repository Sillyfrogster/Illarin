package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/jscode"
	"github.com/google/uuid"
)

const SpindleID = "extension_spindle"

var spindleIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// Spindle reads Spindle extension archives, reading their code for readingTime or the default when that is zero
type Spindle struct {
	readingTime time.Duration
}

func (Spindle) ID() string { return SpindleID }

func (Spindle) Declaration() format.Declaration {
	written := format.DirectionalRoleSupport{
		Read:  format.RoleSupport{Grade: format.SupportFull},
		Write: format.RoleSupport{Grade: format.SupportFull},
	}
	return format.Declaration{
		ID: SpindleID, Label: "Spindle extension", Type: Type,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Type: format.RecognitionEntry, Containers: []format.Container{format.ZIP},
			Entry: spindleManifest,
		}},
		Roles: map[block.Role]format.DirectionalRoleSupport{
			block.RoleExtensionGrantedPermissions:  written,
			block.RoleExtensionApprovedPermissions: written,
			block.RoleExtensionDetails:             written,
			block.RoleExtensionLinks:               written,
			block.RoleExtensionAdditions:           written,
		},
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes, ArchiveFiles: MaxArchiveFiles,
		},
		ConsumedKeys: []string{
			"version", "name", "identifier", "author", "github", "homepage", "description",
			"permissions", "entry_backend", "entry_frontend", "minimum_lumiverse_version",
		},
		Preservation:  format.PreservationDeclaration{Body: SpindleID},
		TestedOrigins: []string{SpindleID},
		KeepsUpload:   true,
	}
}

func (module Spindle) Match(file format.Inspection) (format.Match, bool) {
	return matchArchive(file, module.Declaration(), spindleManifest)
}

type spindleManifestFields struct {
	Version, Name, Identifier, Author, GitHub, Homepage string
	Description, MinimumVersion                         string
	Permissions                                         []string
}

func (s Spindle) Parse(ctx context.Context, file format.Inspection, match format.Match) (format.Parsed, error) {
	if err := checkArchive(file); err != nil {
		return format.Parsed{}, err
	}
	if !hasEntry(file, spindleManifest) {
		return format.Parsed{}, refuseMisplaced(file, spindleManifest)
	}
	payload, ok := match.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the matched payload is missing", SpindleID)
	}
	manifest, err := readSpindleManifest(payload.Root)
	if err != nil {
		return format.Parsed{}, err
	}
	code, err := spindleCode(file, payload.Root)
	if err != nil {
		return format.Parsed{}, err
	}
	readme, err := readReadme(ctx, file)
	if err != nil {
		return format.Parsed{}, err
	}
	limit := s.readingTime
	if limit == 0 {
		limit = readingTime
	}
	adds, err := readAdditions(ctx, file, code, spindleRules, limit)
	if err != nil {
		return format.Parsed{}, err
	}
	header := format.Header{
		Name: manifest.Name, WorkVersion: manifest.Version,
		CreditedAuthor: manifest.Author, Identifier: manifest.Identifier,
	}
	if utf8.RuneCountInString(manifest.Description) <= format.MaxBlurbRunes {
		header.Blurb = manifest.Description
	}
	elements := spindleElements(manifest)
	if adds != nil {
		elements = append(elements, *adds)
	}
	return format.Parsed{
		Type: Type, Format: SpindleID, Header: header, Elements: elements, Readme: readme,
	}, nil
}

func readSpindleManifest(root map[string]json.RawMessage) (spindleManifestFields, error) {
	var manifest spindleManifestFields
	required := []struct {
		key    string
		target *string
	}{
		{"version", &manifest.Version}, {"name", &manifest.Name},
		{"identifier", &manifest.Identifier}, {"author", &manifest.Author},
		{"github", &manifest.GitHub}, {"homepage", &manifest.Homepage},
	}
	for _, field := range required {
		if json.Unmarshal(root[field.key], field.target) != nil || strings.TrimSpace(*field.target) == "" {
			return spindleManifestFields{}, refuse("%s has no %s", spindleManifest, field.key)
		}
	}
	if !spindleIdentifier.MatchString(manifest.Identifier) {
		return spindleManifestFields{}, refuse(
			"%s identifier %q must start with a lowercase letter and hold only lowercase letters, digits and underscores",
			spindleManifest, manifest.Identifier,
		)
	}
	raw, present := root["permissions"]
	if !present || json.Unmarshal(raw, &manifest.Permissions) != nil || manifest.Permissions == nil {
		return spindleManifestFields{}, refuse("%s permissions must be a list of permission names", spindleManifest)
	}
	_ = json.Unmarshal(root["description"], &manifest.Description)
	_ = json.Unmarshal(root["minimum_lumiverse_version"], &manifest.MinimumVersion)
	return manifest, nil
}

type spindleSide struct {
	key, built, source string
}

var spindleSides = []spindleSide{
	{key: "entry_backend", built: "dist/backend.js", source: "src/backend.ts"},
	{key: "entry_frontend", built: "dist/frontend.js", source: "src/frontend.ts"},
}

// spindleCode names the file Lumiverse runs for each side the extension has, the built entry or the source it builds from.
func spindleCode(file format.Inspection, root map[string]json.RawMessage) ([]string, error) {
	var code []string
	for _, side := range spindleSides {
		var declared string
		_ = json.Unmarshal(root[side.key], &declared)
		entry := side.built
		if declared != "" {
			entry = path.Clean(declared)
		}
		if !insideArchive(entry) {
			return nil, refuse("%s %s points outside the archive", spindleManifest, side.key)
		}
		switch {
		case hasEntry(file, entry):
			code = append(code, entry)
		case hasEntry(file, side.source):
			code = append(code, side.source)
		case declared != "":
			return nil, refuse("%s names %s %s, and neither it nor %s is in the archive",
				spindleManifest, side.key, declared, side.source)
		}
	}
	if len(code) == 0 {
		return nil, refuse("the archive has none of dist/backend.js, dist/frontend.js, src/backend.ts or src/frontend.ts")
	}
	return code, nil
}

// spindleObject names the object Lumiverse hands a backend, which a bundler may number to keep it apart from another.
var spindleObject = regexp.MustCompile(`^spindle\d*$`)

// spindleRules read what a backend registers on the spindle object and what a frontend adds through ctx.ui.
var spindleRules = []rule{
	{method: "registerTool", on: onSpindle, read: named(tools, "name", asWritten)},
	{method: "registerMacro", on: onSpindle, read: named(macros, "name", macro)},
	{method: "register", on: onSpindleCommands, read: paletteCommands},
	{method: "registerInterceptor", on: onSpindle, read: hook("Prompt interceptor")},
	{method: "registerContextHandler", on: onSpindle, read: hook("Context handler")},
	{method: "registerMessageContentProcessor", on: onSpindle, read: hook("Message content processor")},
	{method: "registerMacroInterceptor", on: onSpindle, read: hook("Macro interceptor")},
	{method: "registerWorldInfoInterceptor", on: onSpindle, read: hook("World info interceptor")},
	{method: "registerDrawerTab", on: owner("ui"), read: titledSurface("Drawer tab", "title")},
	{method: "registerSettingsTab", on: owner("ui"), read: titledSurface("Settings tab", "title")},
	{method: "registerCharacterEditorTab", on: owner("ui"), read: titledSurface("Character editor tab", "title")},
	{method: "registerConnectionEditorTab", on: owner("ui"), read: titledSurface("Connection editor tab", "title")},
	{method: "registerPresetEditorTab", on: owner("ui"), read: titledSurface("Preset editor tab", "title")},
	{method: "registerPresetEditorToolbarItem", on: owner("ui"), read: titledSurface("Preset editor toolbar item", "ariaLabel")},
	{method: "registerInputBarAction", on: owner("ui"), read: titledSurface("Input bar action", "label")},
	{method: "requestDockPanel", on: owner("ui"), read: titledSurface("Dock panel", "title")},
	{method: "createFloatWidget", on: owner("ui"), read: surface("Float widget")},
	{method: "mountApp", on: owner("ui"), read: surface("App mount")},
}

func onSpindle(callee []string) bool {
	return len(callee) >= 2 && spindleObject.MatchString(callee[len(callee)-2])
}

// onSpindleCommands accepts spindle.commands.register, which fills the command palette.
func onSpindleCommands(callee []string) bool {
	return len(callee) >= 3 && callee[len(callee)-2] == "commands" && spindleObject.MatchString(callee[len(callee)-3])
}

// paletteCommands lists each command palette entry by its label, or by its id where the label is not written out.
func paletteCommands(call jscode.Call) []addition {
	var found []addition
	for _, entry := range call.Arg(0).Items() {
		name, ok := literalName(entry.Property("label"))
		if !ok {
			name, ok = literalName(entry.Property("id"))
		}
		if ok {
			found = append(found, addition{group: commands, name: name})
		}
	}
	return found
}

func spindleElements(manifest spindleManifestFields) []block.Element {
	granted := block.TextSet{Texts: []block.TextItem{}}
	approved := block.TextSet{Texts: []block.TextItem{}}
	seen := make(map[string]bool, len(manifest.Permissions))
	for _, name := range manifest.Permissions {
		if seen[name] {
			continue
		}
		seen[name] = true
		item := block.TextItem{ID: block.NewItemID(), Name: name, Text: describePermission(name)}
		if approvedByAdmin[name] {
			approved.Texts = append(approved.Texts, item)
		} else {
			granted.Texts = append(granted.Texts, item)
		}
	}
	details := block.FieldList{Fields: []block.FieldItem{
		{ID: block.NewItemID(), Name: "Version", Value: manifest.Version},
	}}
	if manifest.MinimumVersion != "" {
		details.Fields = append(details.Fields, block.FieldItem{
			ID: block.NewItemID(), Name: "Minimum Lumiverse version", Value: manifest.MinimumVersion,
		})
	}
	links := block.LinkList{Links: []block.LinkItem{
		{ID: block.NewItemID(), Label: "Repository", URL: manifest.GitHub},
	}}
	if manifest.Homepage != manifest.GitHub {
		links.Links = append(links.Links, block.LinkItem{
			ID: block.NewItemID(), Label: "Homepage", URL: manifest.Homepage,
		})
	}
	return []block.Element{
		{ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleExtensionGrantedPermissions, Content: granted},
		{ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleExtensionApprovedPermissions, Content: approved},
		{ID: uuid.New(), Type: block.TypeFieldList, Role: block.RoleExtensionDetails, Content: details},
		{ID: uuid.New(), Type: block.TypeLinkList, Role: block.RoleExtensionLinks, Content: links},
	}
}

func (Spindle) Write(_ context.Context, work format.ExportWork) (format.MainFile, error) {
	if len(work.Upload) == 0 {
		return format.MainFile{}, errors.New("write the Spindle extension: the uploaded archive is missing")
	}
	return format.MainFile{Body: work.Upload, MediaType: "application/zip", Extension: ".zip"}, nil
}
