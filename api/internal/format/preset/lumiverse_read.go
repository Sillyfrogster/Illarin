package preset

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

type readBlocks struct {
	list           block.PromptList
	variables      []block.Variable
	leftovers      map[uuid.UUID]itemLeftover
	privatePrompts []format.PrivatePrompt
}

func readLumiverseBlocks(
	blocks []json.RawMessage,
	saved map[string]map[string]json.RawMessage,
) (readBlocks, error) {
	read := readBlocks{leftovers: make(map[uuid.UUID]itemLeftover, len(blocks))}
	headings := make(map[string]uuid.UUID, len(blocks))
	privateKeys := make(map[string]bool)
	above := uuid.Nil

	for _, raw := range blocks {
		fields := keys.Object(raw)
		if len(fields) == 0 {
			continue
		}
		var fileID, marker string
		keys.Take(fields, lvBlockID, &fileID)
		keys.Take(fields, lvBlockMarker, &marker)

		if marker == lvHeadingMarker {
			if err := validateLumiverseHeadingIsPublic(fields); err != nil {
				return readBlocks{}, err
			}
			group := block.PromptGroup{ID: block.NewItemID()}
			keys.Take(fields, lvBlockName, &group.Name)
			read.list.Groups = append(read.list.Groups, group)
			headings[fileID] = group.ID
			above = group.ID
			read.keep(group.ID, lumiverseCategoryNamespace, fields, fileID)
			continue
		}

		fragment := block.PromptFragment{ID: block.NewItemID(), Marker: marker}
		keys.Take(fields, lvBlockName, &fragment.Name)
		keys.Take(fields, lvBlockRole, &fragment.Role)
		textPresent := keys.Take(fields, lvBlockText, &fragment.Text)
		keys.Take(fields, lvBlockEnabled, &fragment.Enabled)
		keys.Take(fields, lvBlockPosition, &fragment.Placement)
		var depth int
		if keys.Take(fields, lvBlockDepth, &depth) {
			fragment.Depth = &depth
		}
		if !fragment.Role.Known() {
			fields[lvBlockRole] = keys.Must(fragment.Role)
			fragment.Role = ""
		}
		if !fragment.Placement.Known() {
			fields[lvBlockPosition] = keys.Must(fragment.Placement)
			fragment.Placement = ""
		}
		fragment.GroupID = fragmentHeading(fields, headings, above)
		private, isPrivate, err := readLumiversePrivatePrompt(
			fields, fragment.ID, fragment.Marker, fragment.Text, textPresent,
		)
		if err != nil {
			return readBlocks{}, err
		}
		if isPrivate {
			if privateKeys[private.SourceKey] {
				return readBlocks{}, fmt.Errorf(
					"private prompt key %q appears more than once", private.SourceKey,
				)
			}
			privateKeys[private.SourceKey] = true
			fragment.Private = true
			fragment.Text = ""
			read.privatePrompts = append(read.privatePrompts, private)
		}

		read.variables = append(
			read.variables, readLumiverseVariables(fields, fragment.ID, saved[fileID], &read)...,
		)
		read.list.Fragments = append(read.list.Fragments, fragment)
		read.keep(fragment.ID, lumiverseBlockNamespace, fields, fileID)
	}
	return read, nil
}

func validateLumiverseHeadingIsPublic(fields map[string]json.RawMessage) error {
	if _, present := fields[lvPrivateKey]; present {
		return errors.New("private prompt metadata belongs on a prompt fragment")
	}
	if _, present := fields[lvPrivateKeyLegacy]; present {
		return errors.New("private prompt metadata belongs on a prompt fragment")
	}
	raw, present := fields[lvPrivateFlag]
	if !present {
		return nil
	}
	var isPrivate bool
	if json.Unmarshal(raw, &isPrivate) != nil || isPrivate {
		return errors.New("private prompt metadata belongs on a prompt fragment")
	}
	return nil
}

func readLumiversePrivatePrompt(
	fields map[string]json.RawMessage,
	fragmentID uuid.UUID,
	marker string,
	text string,
	textPresent bool,
) (format.PrivatePrompt, bool, error) {
	flagRaw, hasFlag := fields[lvPrivateFlag]
	_, hasKey := fields[lvPrivateKey]
	_, hasLegacyKey := fields[lvPrivateKeyLegacy]
	if !hasFlag && !hasKey && !hasLegacyKey {
		return format.PrivatePrompt{}, false, nil
	}

	var isPrivate bool
	if !hasFlag || json.Unmarshal(flagRaw, &isPrivate) != nil {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"a private prompt needs %q set to true or false", lvPrivateFlag,
		)
	}
	if !isPrivate {
		if hasKey || hasLegacyKey {
			return format.PrivatePrompt{}, false, fmt.Errorf(
				"a private prompt key needs %q set to true", lvPrivateFlag,
			)
		}
		return format.PrivatePrompt{}, false, nil
	}
	if marker != "" {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"private prompt cannot also be a %q marker", marker,
		)
	}
	if hasKey == hasLegacyKey {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"a private prompt needs exactly one of %q or %q", lvPrivateKey, lvPrivateKeyLegacy,
		)
	}
	if !textPresent {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"private prompt content must be text",
		)
	}

	keyName := lvPrivateKey
	if hasLegacyKey {
		keyName = lvPrivateKeyLegacy
	}
	var sourceKey string
	if json.Unmarshal(fields[keyName], &sourceKey) != nil ||
		sourceKey == "" || sourceKey != strings.TrimSpace(sourceKey) ||
		len([]rune(sourceKey)) > 256 {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"private prompt key must be 1 to 256 trimmed characters",
		)
	}

	delete(fields, lvPrivateFlag)
	delete(fields, keyName)
	placeholder := "{{presetBlock::" + sourceKey + "}}"
	trimmed := strings.TrimSpace(text)
	reuse := trimmed == placeholder
	if !reuse && strings.HasPrefix(trimmed, "{{presetBlock::") && strings.HasSuffix(trimmed, "}}") {
		return format.PrivatePrompt{}, false, fmt.Errorf(
			"private prompt placeholder does not match key %q", sourceKey,
		)
	}
	if reuse {
		text = ""
	}
	return format.PrivatePrompt{
		FragmentID: fragmentID, SourceKey: sourceKey,
		Text: text, ReuseExisting: reuse,
	}, true, nil
}

func (r *readBlocks) keep(
	id uuid.UUID,
	namespace string,
	fields map[string]json.RawMessage,
	fileID string,
) {
	if fileID != "" {
		fields[lvBlockID] = keys.Must(fileID)
	}
	if len(fields) > 0 {
		r.leftovers[id] = itemLeftover{namespace: namespace, fields: fields}
	}
}

func fragmentHeading(
	fields map[string]json.RawMessage,
	headings map[string]uuid.UUID,
	above uuid.UUID,
) *uuid.UUID {
	raw, present := fields[lvBlockGroup]
	if !present {
		if above == uuid.Nil {
			return nil
		}
		return &above
	}
	var named string
	if keys.IsNull(raw) || json.Unmarshal(raw, &named) != nil {
		delete(fields, lvBlockGroup)
		return nil
	}
	heading, known := headings[named]
	if !known {
		return nil
	}
	delete(fields, lvBlockGroup)
	return &heading
}

func readLumiverseVariables(
	fields map[string]json.RawMessage,
	fragmentID uuid.UUID,
	saved map[string]json.RawMessage,
	read *readBlocks,
) []block.Variable {
	var defined []json.RawMessage
	if !keys.Take(fields, lvBlockVars, &defined) {
		return nil
	}
	variables := make([]block.Variable, 0, len(defined))
	for _, raw := range defined {
		definition := keys.Object(raw)
		if len(definition) == 0 {
			continue
		}
		variable := block.Variable{ID: block.NewItemID(), FragmentID: &fragmentID}
		keys.Take(definition, lvVarName, &variable.Name)
		keys.Take(definition, lvVarWidget, &variable.Widget)
		keys.Take(definition, lvVarLabel, &variable.Label)
		keys.Take(definition, lvVarDescription, &variable.Description)
		keys.Take(definition, lvVarSeparator, &variable.Separator)
		keys.Take(definition, lvVarRows, &variable.Rows)
		if !variable.Widget.Known() {
			definition[lvVarWidget] = keys.Must(variable.Widget)
			variable.Widget = block.WidgetText
		}
		variable.Options = readLumiverseOptions(definition)
		variable.Range = readLumiverseRange(definition)
		if raw, present := definition[lvVarDefault]; present {
			if value, readable := readFreeValue(raw); readable {
				delete(definition, lvVarDefault)
				variable.Default = value
			}
		}
		if value, readable := readFreeValue(saved[variable.Name]); readable {
			variable.Value = value
		}
		variables = append(variables, variable)
		if len(definition) > 0 {
			read.leftovers[variable.ID] = itemLeftover{
				namespace: lumiverseVariableNamespace, fields: definition,
			}
		}
	}
	return variables
}

func readLumiverseOptions(definition map[string]json.RawMessage) []block.VariableOption {
	var listed []json.RawMessage
	if !keys.Take(definition, lvVarOptions, &listed) {
		return nil
	}
	options := make([]block.VariableOption, 0, len(listed))
	for _, raw := range listed {
		choice := keys.Object(raw)
		option := block.VariableOption{}
		keys.Take(choice, lvOptionKey, &option.Key)
		keys.Take(choice, lvOptionLabel, &option.Label)
		keys.Take(choice, lvOptionValue, &option.Value)
		options = append(options, option)
	}
	return options
}

func readLumiverseRange(definition map[string]json.RawMessage) *block.VariableRange {
	bounds := block.VariableRange{}
	var min, max, step float64
	bounded := keys.Take(definition, lvVarMin, &min)
	if bounded {
		bounds.Min = &min
	}
	if keys.Take(definition, lvVarMax, &max) {
		bounds.Max, bounded = &max, true
	}
	if keys.Take(definition, lvVarStep, &step) {
		bounds.Step, bounded = &step, true
	}
	if !bounded {
		return nil
	}
	return &bounds
}

func readFreeValue(raw json.RawMessage) (*block.Value, bool) {
	if len(raw) == 0 || keys.IsNull(raw) {
		return nil, false
	}
	var held any
	if json.Unmarshal(raw, &held) != nil {
		return nil, false
	}
	switch held.(type) {
	case float64:
		var number float64
		_ = json.Unmarshal(raw, &number)
		return &block.Value{Number: &number}, true
	case bool:
		var yes bool
		_ = json.Unmarshal(raw, &yes)
		return &block.Value{Boolean: &yes}, true
	case string:
		var text string
		_ = json.Unmarshal(raw, &text)
		return &block.Value{Text: &text}, true
	case []any:
		var strings []string
		if json.Unmarshal(raw, &strings) != nil {
			return nil, false
		}
		return &block.Value{Strings: strings}, true
	}
	return nil, false
}

func readLumiverseScripts(
	source map[string]json.RawMessage,
) ([]block.Script, map[uuid.UUID]map[string]json.RawMessage) {
	listed, present := takeLumiverseScriptList(source)
	if !present {
		return nil, nil
	}
	if len(listed) == 0 {
		source[lvScripts] = keys.Must([]json.RawMessage{})
		return nil, nil
	}

	scripts := make([]block.Script, 0, len(listed))
	fields := make(map[uuid.UUID]map[string]json.RawMessage, len(listed))
	for _, raw := range listed {
		script := keys.Object(raw)
		if len(script) == 0 {
			continue
		}
		read := block.Script{ID: block.NewItemID(), Enabled: true}
		keys.Take(script, lvScriptName, &read.Name)
		keys.Take(script, lvScriptDescription, &read.Description)
		keys.Take(script, lvScriptFind, &read.Find)
		keys.Take(script, lvScriptFlags, &read.Flags)
		keys.Take(script, lvScriptReplace, &read.Replace)
		keys.Take(script, lvScriptTrim, &read.Trim)
		keys.Take(script, lvScriptRunOnEdit, &read.RunOnEdit)
		var minDepth, maxDepth int
		if keys.Take(script, lvScriptMinDepth, &minDepth) {
			read.MinDepth = &minDepth
		}
		if keys.Take(script, lvScriptMaxDepth, &maxDepth) {
			read.MaxDepth = &maxDepth
		}
		var disabled bool
		if keys.Take(script, lvScriptDisabled, &disabled) {
			read.Enabled = !disabled
		}
		read.Targets = takeScriptTargets(script, lvScriptOver, lumiverseScriptTargets)
		read.Affects = takeScriptEffects(script, lvScriptChanges, lumiverseScriptEffects)
		scripts = append(scripts, read)
		if len(script) > 0 {
			fields[read.ID] = script
		}
	}
	return scripts, fields
}

// takeLumiverseScriptList accepts both Lumiverse layouts. Older exports put
// the list at the top level and mirrored it under extensions; newer exports
// bundle it only under extensions. Where both are present, the top-level copy
// remains authoritative for compatibility with files Illarin already wrote.
func takeLumiverseScriptList(source map[string]json.RawMessage) ([]json.RawMessage, bool) {
	var listed []json.RawMessage
	if keys.Take(source, lvScripts, &listed) {
		removeLumiverseScriptExtension(source)
		return listed, true
	}

	extensions := keys.Object(source[lvExtensions])
	if !keys.Take(extensions, lvScripts, &listed) {
		return nil, false
	}
	writeLumiverseExtensions(source, extensions)
	return listed, true
}

func removeLumiverseScriptExtension(source map[string]json.RawMessage) {
	extensions := keys.Object(source[lvExtensions])
	if len(extensions) == 0 {
		return
	}
	delete(extensions, lvScripts)
	writeLumiverseExtensions(source, extensions)
}

func writeLumiverseExtensions(
	source map[string]json.RawMessage,
	extensions map[string]json.RawMessage,
) {
	if len(extensions) == 0 {
		delete(source, lvExtensions)
		return
	}
	source[lvExtensions], _ = json.Marshal(extensions)
}

func takeScriptTargets(
	script map[string]json.RawMessage,
	key string,
	known map[string]block.ScriptTarget,
) []block.ScriptTarget {
	var named []string
	if !keys.Take(script, key, &named) {
		return nil
	}
	targets := make([]block.ScriptTarget, 0, len(named))
	for _, name := range named {
		target, wording := known[name]
		if !wording {
			script[key] = keys.Must(named)
			return nil
		}
		targets = append(targets, target)
	}
	return targets
}

func takeScriptEffects(
	script map[string]json.RawMessage,
	key string,
	known map[string]block.ScriptEffect,
) []block.ScriptEffect {
	var named []string
	if !keys.Take(script, key, &named) {
		return nil
	}
	effects := make([]block.ScriptEffect, 0, len(named))
	for _, name := range named {
		effect, wording := known[name]
		if !wording {
			script[key] = keys.Must(named)
			return nil
		}
		effects = append(effects, effect)
	}
	return effects
}
