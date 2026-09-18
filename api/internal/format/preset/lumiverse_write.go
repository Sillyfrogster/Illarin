package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

func (LumiverseModule) Write(
	_ context.Context,
	work format.ExportWork,
) (format.Artifact, error) {
	held := preservedBy(work.Preserved)
	named := slotsByApp[Lumiverse]

	body := map[string]json.RawMessage{
		lvName:   keys.Must(work.Header.Name),
		lvBlocks: keys.Must(writeLumiverseBlocks(work, held)),
	}
	keys.WriteIfSet(body, lvDescription, work.Header.Blurb != "", work.Header.Blurb)
	keys.WriteIfSet(body, lvVersion, work.Header.WorkVersion != "", work.Header.WorkVersion)
	if saved := writeSavedValues(work, held); len(saved) > 0 {
		body[lvSaved] = keys.Must(saved)
	}
	for _, group := range []struct {
		key   string
		role  block.Role
		slots []slot
	}{
		{lvSamplers, block.RoleSamplerSettings, named.samplers},
		{lvCompletion, block.RoleCompletionSettings, named.completion},
		{lvAdvanced, block.RoleAdvancedSettings, named.advanced},
	} {
		writeNested(body, group.key, writeLumiverseSettings(work, group.role, group.slots))
	}
	writeNested(body, lvBehaviour, writeLumiverseNudges(work, named.nudges))
	if written := writeLumiverseScripts(work, held); len(written) > 0 {
		body[lvScripts] = keys.Must(written)
		body[lvExtensions] = keys.Must(map[string][]map[string]json.RawMessage{
			lvScripts: written,
		})
	}

	restoreLumiversePreserved(body, held)
	document, err := json.Marshal(body)
	if err != nil {
		return format.Artifact{}, fmt.Errorf("write the preset: %w", err)
	}
	return format.Artifact{
		Body: document, MediaType: "application/json", Extension: ".json",
	}, nil
}

func writeNested(body map[string]json.RawMessage, key string, values map[string]json.RawMessage) {
	if len(values) == 0 {
		return
	}
	body[key] = keys.Must(values)
}

func writeLumiverseBlocks(
	work format.ExportWork,
	held kept,
) []map[string]json.RawMessage {
	list := fragments(work)
	names := lumiverseBlockNames(list, held)
	forms := make(map[uuid.UUID][]block.Variable)
	for _, variable := range variables(work) {
		if variable.FragmentID != nil {
			forms[*variable.FragmentID] = append(forms[*variable.FragmentID], variable)
		}
	}
	headings := make(map[uuid.UUID]block.PromptGroup, len(list.Groups))
	for _, group := range list.Groups {
		headings[group.ID] = group
	}

	written := make([]map[string]json.RawMessage, 0, len(list.Fragments)+len(list.Groups))
	placed := make(map[uuid.UUID]bool, len(list.Groups))
	next := 0
	place := func(upTo uuid.UUID) {
		for ; next < len(list.Groups); next++ {
			group := list.Groups[next]
			placed[group.ID] = true
			written = append(written, writeLumiverseHeading(group, names, held))
			if group.ID == upTo {
				next++
				return
			}
		}
	}
	for _, fragment := range list.Fragments {
		if at := fragment.GroupID; at != nil && !placed[*at] {
			if _, known := headings[*at]; known {
				place(*at)
			}
		}
		written = append(written, writeLumiverseFragment(fragment, names, forms, held))
	}
	place(uuid.Nil)
	return written
}

func lumiverseBlockNames(list block.PromptList, held kept) map[uuid.UUID]string {
	names := make(map[uuid.UUID]string, len(list.Groups)+len(list.Fragments))
	for _, group := range list.Groups {
		names[group.ID] = itemName(held, lumiverseCategoryNamespace, group.ID, lvBlockID)
	}
	for _, fragment := range list.Fragments {
		names[fragment.ID] = itemName(held, lumiverseBlockNamespace, fragment.ID, lvBlockID)
	}
	return names
}

func writeLumiverseHeading(
	group block.PromptGroup,
	names map[uuid.UUID]string,
	held kept,
) map[string]json.RawMessage {
	written := map[string]json.RawMessage{
		lvBlockID:     keys.Must(names[group.ID]),
		lvBlockName:   keys.Must(group.Name),
		lvBlockMarker: keys.Must(lvHeadingMarker),
	}
	keys.MergeAbsent(written, held.item(lumiverseCategoryNamespace, group.ID))
	return written
}

func writeLumiverseFragment(
	fragment block.PromptFragment,
	names map[uuid.UUID]string,
	forms map[uuid.UUID][]block.Variable,
	held kept,
) map[string]json.RawMessage {
	written := map[string]json.RawMessage{
		lvBlockID:      keys.Must(names[fragment.ID]),
		lvBlockName:    keys.Must(fragment.Name),
		lvBlockText:    keys.Must(fragment.Text),
		lvBlockEnabled: keys.Must(fragment.Enabled),
	}
	keys.WriteIfSet(written, lvBlockRole, fragment.Role != "", fragment.Role)
	keys.WriteIfSet(written, lvBlockPosition, fragment.Placement != "", fragment.Placement)
	keys.WriteIfSet(written, lvBlockMarker, fragment.Marker != "", fragment.Marker)
	keys.WriteIfSet(written, lvBlockDepth, fragment.Depth != nil, fragment.Depth)
	if fragment.GroupID != nil {
		written[lvBlockGroup] = keys.Must(names[*fragment.GroupID])
	}
	if form := forms[fragment.ID]; len(form) > 0 {
		defined := make([]map[string]json.RawMessage, 0, len(form))
		for _, variable := range form {
			defined = append(defined, writeLumiverseVariable(variable, held))
		}
		written[lvBlockVars] = keys.Must(defined)
	}
	keys.MergeAbsent(written, held.item(lumiverseBlockNamespace, fragment.ID))
	return written
}

func writeLumiverseVariable(variable block.Variable, held kept) map[string]json.RawMessage {
	fields := map[string]json.RawMessage{
		lvVarID:     keys.Must(itemName(held, lumiverseVariableNamespace, variable.ID, lvVarID)),
		lvVarName:   keys.Must(variable.Name),
		lvVarWidget: keys.Must(variable.Widget),
	}
	keys.WriteIfSet(fields, lvVarLabel, variable.Label != "", variable.Label)
	keys.WriteIfSet(fields, lvVarDescription, variable.Description != "", variable.Description)
	keys.WriteIfSet(fields, lvVarSeparator, variable.Separator != "", variable.Separator)
	keys.WriteIfSet(fields, lvVarRows, variable.Rows > 0, variable.Rows)
	if variable.Default != nil {
		fields[lvVarDefault] = writeFreeValue(*variable.Default)
	}
	if len(variable.Options) > 0 {
		choices := make([]map[string]json.RawMessage, 0, len(variable.Options))
		for _, option := range variable.Options {
			choices = append(choices, map[string]json.RawMessage{
				lvOptionKey:   keys.Must(option.Named()),
				lvOptionLabel: keys.Must(option.Label),
				lvOptionValue: keys.Must(option.Value),
			})
		}
		fields[lvVarOptions] = keys.Must(choices)
	}
	if bounds := variable.Range; bounds != nil {
		keys.WriteIfSet(fields, lvVarMin, bounds.Min != nil, bounds.Min)
		keys.WriteIfSet(fields, lvVarMax, bounds.Max != nil, bounds.Max)
		keys.WriteIfSet(fields, lvVarStep, bounds.Step != nil, bounds.Step)
	}
	keys.MergeAbsent(fields, held.item(lumiverseVariableNamespace, variable.ID))
	return fields
}

func writeSavedValues(
	work format.ExportWork,
	held kept,
) map[string]map[string]json.RawMessage {
	names := lumiverseBlockNames(fragments(work), held)
	saved := make(map[string]map[string]json.RawMessage)
	for _, variable := range variables(work) {
		if variable.FragmentID == nil || variable.Value == nil {
			continue
		}
		name, known := names[*variable.FragmentID]
		if !known {
			continue
		}
		if saved[name] == nil {
			saved[name] = make(map[string]json.RawMessage)
		}
		saved[name][variable.Name] = writeFreeValue(*variable.Value)
	}
	return saved
}

func writeLumiverseSettings(
	work format.ExportWork,
	role block.Role,
	named []slot,
) map[string]json.RawMessage {
	written := make(map[string]json.RawMessage)
	for _, setting := range settings(work, role) {
		if !slices.ContainsFunc(named, func(s slot) bool { return s.name == setting.Name }) {
			continue
		}
		written[setting.Name] = writeValue(setting)
	}
	return written
}

func writeLumiverseNudges(work format.ExportWork, names []string) map[string]json.RawMessage {
	written := make(map[string]json.RawMessage)
	for _, text := range nudges(work) {
		if slices.Contains(names, text.Name) {
			written[text.Name] = keys.Must(text.Text)
		}
	}
	return written
}

func writeLumiverseScripts(
	work format.ExportWork,
	held kept,
) []map[string]json.RawMessage {
	list := scripts(work)
	written := make([]map[string]json.RawMessage, 0, len(list))
	for _, script := range list {
		fields := map[string]json.RawMessage{
			lvScriptFind:     keys.Must(script.Find),
			lvScriptReplace:  keys.Must(script.Replace),
			lvScriptDisabled: keys.Must(!script.Enabled),
		}
		keys.WriteIfSet(fields, lvScriptName, script.Name != "", script.Name)
		keys.WriteIfSet(fields, lvScriptDescription, script.Description != "", script.Description)
		keys.WriteIfSet(fields, lvScriptFlags, script.Flags != "", script.Flags)
		keys.WriteIfSet(fields, lvScriptTrim, script.Trim != nil, script.Trim)
		keys.WriteIfSet(fields, lvScriptRunOnEdit, script.RunOnEdit, script.RunOnEdit)
		keys.WriteIfSet(fields, lvScriptMinDepth, script.MinDepth != nil, script.MinDepth)
		keys.WriteIfSet(fields, lvScriptMaxDepth, script.MaxDepth != nil, script.MaxDepth)
		keys.WriteIfSet(fields, lvScriptOver, len(script.Targets) > 0,
			namesFor(script.Targets, lumiverseScriptTargets))
		keys.WriteIfSet(fields, lvScriptChanges, len(script.Affects) > 0,
			namesFor(script.Affects, lumiverseScriptEffects))
		keys.MergeAbsent(fields, held.item(lumiverseScriptNamespace, script.ID))
		written = append(written, fields)
	}
	return written
}

func namesFor[T comparable](values []T, known map[string]T) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		for name, wording := range known {
			if wording == value {
				names = append(names, name)
				break
			}
		}
	}
	return names
}

func writeFreeValue(value block.Value) json.RawMessage {
	switch {
	case value.Number != nil:
		return keys.Must(*value.Number)
	case value.Boolean != nil:
		return keys.Must(*value.Boolean)
	case value.Text != nil:
		return keys.Must(*value.Text)
	default:
		return keys.Must(orEmptyStrings(value.Strings))
	}
}

func restoreLumiversePreserved(body map[string]json.RawMessage, held kept) {
	preserved := held.object(lumiverseNamespace)
	for _, key := range []string{lvSamplers, lvCompletion, lvAdvanced, lvBehaviour} {
		inside := keys.Object(preserved[key])
		if len(inside) == 0 {
			continue
		}
		written := keys.Object(body[key])
		keys.MergeAbsent(written, inside)
		body[key] = keys.Must(written)
		delete(preserved, key)
	}
	keys.MergeAbsent(body, preserved)
	lumiversePreservation.restoreExtensions(body, held)
}
