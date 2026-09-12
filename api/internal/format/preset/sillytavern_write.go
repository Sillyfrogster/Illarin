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

func (SillyTavernModule) Write(
	_ context.Context,
	asset format.ExportAsset,
) (format.Artifact, error) {
	held := preservedBy(asset.Preserved)
	named := slotsByApp[SillyTavern]
	list := fragments(asset)
	identifiers := sillyTavernIdentifiers(list, held)

	body := map[string]json.RawMessage{
		stPrompts: keys.Must(writeSillyTavernPrompts(list, identifiers, held)),
		stOrder:   keys.Must(writeSillyTavernOrder(list, identifiers, held)),
	}
	for _, group := range []struct {
		role  block.Role
		slots []slot
	}{
		{block.RoleSamplerSettings, named.samplers},
		{block.RoleCompletionSettings, named.completion},
		{block.RoleAdvancedSettings, named.advanced},
	} {
		for _, setting := range settings(asset, group.role) {
			if setting.Value == nil || !slices.ContainsFunc(group.slots, func(s slot) bool {
				return s.name == setting.Name
			}) {
				continue
			}
			body[setting.Name] = writeValue(setting)
		}
	}
	for _, text := range nudges(asset) {
		if slices.Contains(named.nudges, text.Name) {
			body[text.Name] = keys.Must(text.Text)
		}
	}
	if written := writeSillyTavernScripts(asset, held); len(written) > 0 {
		body[stExtensions] = keys.Must(map[string][]map[string]json.RawMessage{
			stScripts: written,
		})
	}

	restoreSillyTavernPreserved(body, held)
	document, err := json.Marshal(body)
	if err != nil {
		return format.Artifact{}, fmt.Errorf("write the preset: %w", err)
	}
	return format.Artifact{
		Body: document, MediaType: "application/json", Extension: ".json",
	}, nil
}

func sillyTavernIdentifiers(list block.PromptList, held kept) map[uuid.UUID]string {
	identifiers := make(map[uuid.UUID]string, len(list.Fragments))
	for _, fragment := range list.Fragments {
		identifiers[fragment.ID] = itemName(
			held, sillyTavernPromptNamespace, fragment.ID, stIdentifier,
		)
	}
	return identifiers
}

func writeSillyTavernPrompts(
	list block.PromptList,
	identifiers map[uuid.UUID]string,
	held kept,
) []map[string]json.RawMessage {
	written := make([]map[string]json.RawMessage, 0, len(list.Fragments))
	for _, fragment := range list.Fragments {
		fields := map[string]json.RawMessage{
			stIdentifier: keys.Must(identifiers[fragment.ID]),
			stName:       keys.Must(fragment.Name),
		}
		keys.WriteIfSet(fields, stMarker, fragment.Marker != "", true)
		keys.WriteIfSet(fields, stRole, fragment.Role != "", fragment.Role)
		keys.WriteIfSet(fields, stText,
			fragment.Text != "" || fragment.Marker == "", fragment.Text)
		if fragment.Placement == block.InHistory {
			fields[stPosition] = keys.Must(stInHistory)
			keys.WriteIfSet(fields, stDepth, fragment.Depth != nil, fragment.Depth)
		}
		keys.MergeAbsent(fields, held.item(sillyTavernPromptNamespace, fragment.ID))
		written = append(written, fields)
	}
	return written
}

func writeSillyTavernOrder(
	list block.PromptList,
	identifiers map[uuid.UUID]string,
	held kept,
) []map[string]json.RawMessage {
	sent := make([]map[string]json.RawMessage, 0, len(list.Fragments))
	for _, fragment := range list.Fragments {
		sent = append(sent, map[string]json.RawMessage{
			stIdentifier: keys.Must(identifiers[fragment.ID]),
			stEnabled:    keys.Must(fragment.Enabled),
		})
	}
	live := map[string]json.RawMessage{
		stCharacter: keys.Must(stLiveOrder),
		stOrderList: keys.Must(sent),
	}

	var others []map[string]json.RawMessage
	_ = json.Unmarshal(held.object(sillyTavernNamespace)[stOrder], &others)
	return append(others, live)
}

func writeSillyTavernScripts(
	asset format.ExportAsset,
	held kept,
) []map[string]json.RawMessage {
	list := scripts(asset)
	written := make([]map[string]json.RawMessage, 0, len(list))
	for _, script := range list {
		fields := map[string]json.RawMessage{
			stScriptFind:     keys.Must(script.Find),
			stScriptReplace:  keys.Must(script.Replace),
			stScriptDisabled: keys.Must(!script.Enabled),
		}
		keys.WriteIfSet(fields, stScriptName, script.Name != "", script.Name)
		keys.WriteIfSet(fields, stScriptTrim, script.Trim != nil, script.Trim)
		keys.WriteIfSet(fields, stScriptOnEdit, script.RunOnEdit, script.RunOnEdit)
		keys.WriteIfSet(fields, stScriptMinDepth, script.MinDepth != nil, script.MinDepth)
		keys.WriteIfSet(fields, stScriptMaxDepth, script.MaxDepth != nil, script.MaxDepth)
		keys.WriteIfSet(fields, stScriptOver, len(script.Targets) > 0,
			numbersFor(script.Targets))
		if len(script.Affects) > 0 {
			fields[stScriptDisplay] = keys.Must(
				slices.Contains(script.Affects, block.EffectDisplay) &&
					!slices.Contains(script.Affects, block.EffectPrompt),
			)
			fields[stScriptPrompt] = keys.Must(
				slices.Contains(script.Affects, block.EffectPrompt) &&
					!slices.Contains(script.Affects, block.EffectDisplay),
			)
		}
		keys.MergeAbsent(fields, held.item(sillyTavernScriptNamespace, script.ID))
		written = append(written, fields)
	}
	return written
}

func numbersFor(targets []block.ScriptTarget) []float64 {
	numbered := make([]float64, 0, len(targets))
	for _, target := range targets {
		for number, wording := range sillyTavernScriptTargets {
			if wording == target {
				numbered = append(numbered, number)
				break
			}
		}
	}
	slices.Sort(numbered)
	return numbered
}

func restoreSillyTavernPreserved(body map[string]json.RawMessage, held kept) {
	preserved := held.object(sillyTavernNamespace)
	delete(preserved, stOrder)
	keys.MergeAbsent(body, preserved)
	sillyTavernPreservation.restoreExtensions(body, held)
}
