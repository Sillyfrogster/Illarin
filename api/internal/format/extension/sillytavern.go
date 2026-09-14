package extension

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/jscode"
	"github.com/Sillyfrogster/Illarin/api/internal/probe"
	"github.com/google/uuid"
)

const (
	SillyTavernID = "extension_sillytavern"

	thirdPartyFolder = "third-party/"
)

type SillyTavern struct{}

func (SillyTavern) ID() string { return SillyTavernID }

func (SillyTavern) Declaration() format.Declaration {
	written := format.DirectionalRoleSupport{
		Read:  format.RoleSupport{Grade: format.SupportFull},
		Write: format.RoleSupport{Grade: format.SupportFull},
	}
	return format.Declaration{
		ID: SillyTavernID, Label: "SillyTavern extension", Kind: Kind,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Kind: format.RecognitionEntry, Containers: []probe.Container{probe.ZIP},
			Entry: sillyTavernManifest,
		}},
		Roles: map[block.Role]format.DirectionalRoleSupport{
			block.RoleExtensionDependencies: written,
			block.RoleExtensionDetails:      written,
			block.RoleExtensionLinks:        written,
			block.RoleExtensionAdditions:    written,
		},
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes, ArchiveFiles: MaxArchiveFiles,
		},
		ConsumedKeys: []string{
			"display_name", "js", "author", "version", "description", "dependencies",
			"minimum_client_version", "homePage", "homepage",
		},
		Preservation:  format.PreservationDeclaration{Body: SillyTavernID},
		TestedOrigins: []string{SillyTavernID},
		KeepsUpload:   true,
	}
}

func (module SillyTavern) Claim(file probe.Inspection) (format.Claim, bool) {
	return claimArchive(file, module.Declaration(), sillyTavernManifest)
}

type sillyTavernManifestFields struct {
	DisplayName, Script, Author                string
	Version, Description, MinimumVersion, Home string
	Dependencies                               []string
}

func (SillyTavern) Parse(ctx context.Context, file probe.Inspection, claim format.Claim) (format.Parsed, error) {
	if err := checkArchive(file); err != nil {
		return format.Parsed{}, err
	}
	if !hasEntry(file, sillyTavernManifest) {
		return format.Parsed{}, refuseMisplaced(file, sillyTavernManifest)
	}
	payload, ok := claim.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the claimed payload is missing", SillyTavernID)
	}
	manifest, err := readSillyTavernManifest(payload.Root)
	if err != nil {
		return format.Parsed{}, err
	}
	script := path.Clean(manifest.Script)
	if !insideArchive(script) {
		return format.Parsed{}, refuse("%s js points outside the archive", sillyTavernManifest)
	}
	if !hasEntry(file, script) {
		return format.Parsed{}, refuse("%s names js %s, and it is not in the archive", sillyTavernManifest, manifest.Script)
	}
	readme, err := readReadme(ctx, file)
	if err != nil {
		return format.Parsed{}, err
	}
	adds, err := readAdditions(ctx, file, []string{script}, sillyTavernRules, readingTime)
	if err != nil {
		return format.Parsed{}, err
	}
	header := format.Header{
		Name: manifest.DisplayName, AssetVersion: manifest.Version,
		CreditedAuthor: manifest.Author, Identifier: repositoryFolder(manifest.Home),
	}
	if utf8.RuneCountInString(manifest.Description) <= format.MaxBlurbRunes {
		header.Blurb = manifest.Description
	}
	elements := sillyTavernElements(manifest)
	if adds != nil {
		elements = append(elements, *adds)
	}
	return format.Parsed{
		Kind: Kind, Format: SillyTavernID, Header: header, Elements: elements, Readme: readme,
	}, nil
}

func readSillyTavernManifest(root map[string]json.RawMessage) (sillyTavernManifestFields, error) {
	var manifest sillyTavernManifestFields
	required := []struct {
		key    string
		target *string
	}{
		{"display_name", &manifest.DisplayName}, {"js", &manifest.Script}, {"author", &manifest.Author},
	}
	for _, field := range required {
		if json.Unmarshal(root[field.key], field.target) != nil || strings.TrimSpace(*field.target) == "" {
			return sillyTavernManifestFields{}, refuse("%s has no %s", sillyTavernManifest, field.key)
		}
	}
	_ = json.Unmarshal(root["version"], &manifest.Version)
	_ = json.Unmarshal(root["description"], &manifest.Description)
	_ = json.Unmarshal(root["minimum_client_version"], &manifest.MinimumVersion)
	if json.Unmarshal(root["homePage"], &manifest.Home) != nil || manifest.Home == "" {
		_ = json.Unmarshal(root["homepage"], &manifest.Home)
	}
	// SillyTavern loads an extension whose dependencies are not a list, so a malformed entry is skipped.
	var dependencies []json.RawMessage
	_ = json.Unmarshal(root["dependencies"], &dependencies)
	for _, raw := range dependencies {
		var name string
		if json.Unmarshal(raw, &name) == nil && name != "" {
			manifest.Dependencies = append(manifest.Dependencies, name)
		}
	}
	return manifest, nil
}

// repositoryFolder is the folder SillyTavern clones a repository into, the last part of its address.
func repositoryFolder(home string) string {
	address, ok := repositoryAddress(home)
	if !ok {
		return ""
	}
	folder := strings.TrimSuffix(path.Base(strings.TrimRight(address.Path, "/")), ".git")
	if folder == "." || folder == "/" {
		return ""
	}
	return folder
}

func repositoryAddress(home string) (*url.URL, bool) {
	address, err := url.Parse(home)
	if err != nil || (address.Scheme != "https" && address.Scheme != "http") || address.Host == "" {
		return nil, false
	}
	return address, true
}

// sillyTavernRules read what an extension registers with SillyTavern, whichever import or context it reached the call through.
var sillyTavernRules = []rule{
	{method: "addCommandObject", on: onAnything, read: slashCommand},
	{method: "registerSlashCommand", on: onAnything, read: first(commands, slash)},
	{method: "registerMacro", on: onAnything, read: first(macros, macro)},
	{method: "register", on: owner("macros"), read: first(macros, macro)},
	{method: "registerFunctionTool", on: onAnything, read: named(tools, "name", asWritten)},
	{method: "on", on: owner("eventSource"), read: eventHook},
	{method: "once", on: owner("eventSource"), read: eventHook},
	{method: "makeFirst", on: owner("eventSource"), read: eventHook},
	{method: "makeLast", on: owner("eventSource"), read: eventHook},
}

// slashCommand lists a command built by SlashCommand.fromProps, named as it is typed.
func slashCommand(call jscode.Call) []addition {
	built := call.Arg(0).Call()
	if len(built.Callee) == 0 || built.Callee[len(built.Callee)-1] != "fromProps" {
		return nil
	}
	return one(commands, built.Arg(0).Property("name"), slash)
}

// eventHook lists an event the extension hooks into, named as SillyTavern names it.
func eventHook(call jscode.Call) []addition {
	event := call.Arg(0)
	member := event.Member()
	if len(member) >= 2 && (member[len(member)-2] == "event_types" || member[len(member)-2] == "eventTypes") {
		return []addition{{group: hooks, name: member[len(member)-1]}}
	}
	return one(hooks, event, asWritten)
}

func sillyTavernElements(manifest sillyTavernManifestFields) []block.Element {
	details := block.FieldList{Fields: []block.FieldItem{}}
	if manifest.Version != "" {
		details.Fields = append(details.Fields, block.FieldItem{
			ID: block.NewItemID(), Name: "Version", Value: manifest.Version,
		})
	}
	if manifest.MinimumVersion != "" {
		details.Fields = append(details.Fields, block.FieldItem{
			ID: block.NewItemID(), Name: "Minimum SillyTavern version", Value: manifest.MinimumVersion,
		})
	}
	if len(details.Fields) == 0 {
		details.Fields = append(details.Fields, block.FieldItem{
			ID: block.NewItemID(), Name: "Script", Value: manifest.Script,
		})
	}
	elements := []block.Element{
		{ID: uuid.New(), Type: block.TypeFieldList, Role: block.RoleExtensionDetails, Content: details},
	}
	if _, ok := repositoryAddress(manifest.Home); ok {
		elements = append(elements, block.Element{
			ID: uuid.New(), Type: block.TypeLinkList, Role: block.RoleExtensionLinks,
			Content: block.LinkList{Links: []block.LinkItem{
				{ID: block.NewItemID(), Label: "Repository", URL: manifest.Home},
			}},
		})
	}
	if dependencies := dependencyTexts(manifest.Dependencies); len(dependencies.Texts) > 0 {
		elements = append(elements, block.Element{
			ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleExtensionDependencies, Content: dependencies,
		})
	}
	return elements
}

// dependencyTexts names each dependency by the identifier it refers to, which only a third-party extension has.
func dependencyTexts(names []string) block.TextSet {
	dependencies := block.TextSet{Texts: []block.TextItem{}}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			continue
		}
		seen[name] = true
		folder, _ := strings.CutPrefix(name, thirdPartyFolder)
		if folder == name || strings.Contains(folder, "/") {
			folder = ""
		}
		dependencies.Texts = append(dependencies.Texts, block.TextItem{ID: block.NewItemID(), Name: folder, Text: name})
	}
	return dependencies
}

func (SillyTavern) Write(_ context.Context, asset format.ExportAsset) (format.Artifact, error) {
	if len(asset.Upload) == 0 {
		return format.Artifact{}, errors.New("write the SillyTavern extension: the uploaded archive is missing")
	}
	return format.Artifact{Body: asset.Upload, MediaType: "application/zip", Extension: ".zip"}, nil
}
