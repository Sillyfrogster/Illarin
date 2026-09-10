package preset

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

type sent struct {
	identifier string
	enabled    bool
}

func takeLiveOrder(source map[string]json.RawMessage) ([]sent, bool) {
	var orders []map[string]json.RawMessage
	if !keys.Take(source, stOrder, &orders) {
		return nil, false
	}
	var live []sent
	found := false
	remaining := make([]map[string]json.RawMessage, 0, len(orders))
	for _, entry := range orders {
		var character int
		var listed []map[string]json.RawMessage
		named := keys.Take(entry, stCharacter, &character)
		if found || !named || character != stLiveOrder || !keys.Take(entry, stOrderList, &listed) {
			if named {
				entry[stCharacter] = keys.Must(character)
			}
			remaining = append(remaining, entry)
			continue
		}
		found = true
		for _, item := range listed {
			line := sent{}
			if !keys.Take(item, stIdentifier, &line.identifier) {
				continue
			}
			keys.Take(item, stEnabled, &line.enabled)
			live = append(live, line)
		}
	}
	if len(remaining) > 0 {
		source[stOrder], _ = json.Marshal(remaining)
	}
	return live, found
}

func readSillyTavernPrompts(
	prompts []json.RawMessage,
	live []sent,
	ordered bool,
) (block.PromptList, map[uuid.UUID]itemLeftover) {
	read := make([]block.PromptFragment, 0, len(prompts))
	named := make(map[string]int, len(prompts))
	leftovers := make(map[uuid.UUID]itemLeftover, len(prompts))

	for _, raw := range prompts {
		fields := keys.Object(raw)
		if len(fields) == 0 {
			continue
		}
		var identifier string
		keys.Take(fields, stIdentifier, &identifier)

		fragment := block.PromptFragment{ID: block.NewItemID(), Enabled: !ordered}
		keys.Take(fields, stName, &fragment.Name)
		keys.Take(fields, stRole, &fragment.Role)
		keys.Take(fields, stText, &fragment.Text)
		if !fragment.Role.Known() {
			fields[stRole] = keys.Must(fragment.Role)
			fragment.Role = ""
		}
		var marker bool
		if keys.Take(fields, stMarker, &marker) && marker {
			fragment.Marker = identifier
		}
		var position, depth int
		if keys.Take(fields, stPosition, &position) {
			if position == stInHistory {
				fragment.Placement = block.InHistory
				if keys.Take(fields, stDepth, &depth) {
					fragment.Depth = &depth
				}
			} else if position != stInOrder {
				fields[stPosition] = keys.Must(position)
			}
		}
		if identifier != "" {
			fields[stIdentifier] = keys.Must(identifier)
			if _, taken := named[identifier]; !taken {
				named[identifier] = len(read)
			}
		}
		read = append(read, fragment)
		if len(fields) > 0 {
			leftovers[fragment.ID] = itemLeftover{
				namespace: sillyTavernPromptNamespace, fields: fields,
			}
		}
	}

	list := block.PromptList{Fragments: make([]block.PromptFragment, 0, len(read))}
	placed := make([]bool, len(read))
	for _, line := range live {
		at, known := named[line.identifier]
		if !known || placed[at] {
			continue
		}
		placed[at] = true
		read[at].Enabled = line.enabled
		list.Fragments = append(list.Fragments, read[at])
	}
	for at, fragment := range read {
		if !placed[at] {
			list.Fragments = append(list.Fragments, fragment)
		}
	}
	return list, leftovers
}

func readSillyTavernScripts(
	source map[string]json.RawMessage,
) ([]block.Script, map[uuid.UUID]map[string]json.RawMessage) {
	extensions := keys.Object(source[stExtensions])
	var listed []json.RawMessage
	if !keys.Take(extensions, stScripts, &listed) {
		return nil, nil
	}
	if len(listed) == 0 {
		return nil, nil
	}
	if len(extensions) == 0 {
		delete(source, stExtensions)
	} else {
		source[stExtensions], _ = json.Marshal(extensions)
	}

	scripts := make([]block.Script, 0, len(listed))
	fields := make(map[uuid.UUID]map[string]json.RawMessage, len(listed))
	for _, raw := range listed {
		script := keys.Object(raw)
		if len(script) == 0 {
			continue
		}
		read := block.Script{ID: block.NewItemID(), Enabled: true}
		keys.Take(script, stScriptName, &read.Name)
		keys.Take(script, stScriptFind, &read.Find)
		keys.Take(script, stScriptReplace, &read.Replace)
		keys.Take(script, stScriptTrim, &read.Trim)
		keys.Take(script, stScriptOnEdit, &read.RunOnEdit)
		var minDepth, maxDepth int
		if keys.Take(script, stScriptMinDepth, &minDepth) {
			read.MinDepth = &minDepth
		}
		if keys.Take(script, stScriptMaxDepth, &maxDepth) {
			read.MaxDepth = &maxDepth
		}
		var disabled bool
		if keys.Take(script, stScriptDisabled, &disabled) {
			read.Enabled = !disabled
		}
		read.Targets = takeNumberedTargets(script)
		read.Affects = takeSillyTavernEffects(script)
		scripts = append(scripts, read)
		if len(script) > 0 {
			fields[read.ID] = script
		}
	}
	return scripts, fields
}

func takeNumberedTargets(script map[string]json.RawMessage) []block.ScriptTarget {
	var numbered []float64
	if !keys.Take(script, stScriptOver, &numbered) {
		return nil
	}
	targets := make([]block.ScriptTarget, 0, len(numbered))
	for _, number := range numbered {
		target, wording := sillyTavernScriptTargets[number]
		if !wording {
			script[stScriptOver] = keys.Must(numbered)
			return nil
		}
		targets = append(targets, target)
	}
	return targets
}

func takeSillyTavernEffects(script map[string]json.RawMessage) []block.ScriptEffect {
	var display, prompt bool
	shown := keys.Take(script, stScriptDisplay, &display)
	sent := keys.Take(script, stScriptPrompt, &prompt)
	if !shown && !sent {
		return nil
	}
	switch {
	case display && !prompt:
		return []block.ScriptEffect{block.EffectDisplay}
	case prompt && !display:
		return []block.ScriptEffect{block.EffectPrompt}
	default:
		return []block.ScriptEffect{block.EffectDisplay, block.EffectPrompt}
	}
}
