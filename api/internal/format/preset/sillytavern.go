package preset

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

const SillyTavernID = "preset_sillytavern"

const (
	sillyTavernNamespace       = SillyTavernID
	sillyTavernPromptNamespace = SillyTavernID + "_prompt"
	sillyTavernScriptNamespace = SillyTavernID + "_script"
)

var sillyTavernPreservation = preservation{
	body: sillyTavernNamespace, extensions: stExtensions,
	reserved: []string{
		sillyTavernNamespace, sillyTavernPromptNamespace, sillyTavernScriptNamespace,
	},
}

const (
	stPrompts    = "prompts"
	stOrder      = "prompt_order"
	stExtensions = "extensions"
	stScripts    = "regex_scripts"
)

const (
	stIdentifier = "identifier"
	stName       = "name"
	stRole       = "role"
	stText       = "content"
	stMarker     = "marker"
	stPosition   = "injection_position"
	stDepth      = "injection_depth"
	stCharacter  = "character_id"
	stOrderList  = "order"
	stEnabled    = "enabled"
)

const (
	stScriptName     = "scriptName"
	stScriptFind     = "findRegex"
	stScriptReplace  = "replaceString"
	stScriptTrim     = "trimStrings"
	stScriptOver     = "placement"
	stScriptDisabled = "disabled"
	stScriptDisplay  = "markdownOnly"
	stScriptPrompt   = "promptOnly"
	stScriptOnEdit   = "runOnEdit"
	stScriptMinDepth = "minDepth"
	stScriptMaxDepth = "maxDepth"
)

const stLiveOrder = 100001

const (
	stInOrder    = 0
	stInHistory  = 1
	stMarkerOnly = true
)

var sillyTavernScriptTargets = map[float64]block.ScriptTarget{
	1: block.TargetUserInput,
	2: block.TargetModelOutput,
	3: block.TargetSlashCommand,
	5: block.TargetLorebook,
}

type SillyTavernModule struct{}

func (SillyTavernModule) ID() string { return SillyTavernID }

func (SillyTavernModule) Declaration() format.Declaration {
	named := slotsByApp[SillyTavern]
	return format.Declaration{
		ID: SillyTavernID, Label: "SillyTavern preset", Type: Type,
		Direction: format.Direction{Read: true, Write: true},
		Recognition: []format.Recognition{{
			Type:       format.RecognitionSignature,
			Containers: []format.Container{format.JSON},
			Required: map[string]format.ValueType{
				stPrompts: format.ValueArray, stOrder: format.ValueArray,
			},
		}},
		Roles: map[block.Role]format.DirectionalRoleSupport{
			block.RolePromptFragments: {
				Read: format.RoleSupport{Grade: format.SupportFull},
				Write: format.RoleSupport{
					Grade: format.SupportPartial,
					Condition: &format.ContentCondition{
						Description: "the headings over the fragments, and a fragment " +
							"placed before or after the history, because the file has neither",
						Matches: hasHeadingOrHistoryPlacement,
					},
				},
			},
			block.RolePromptVariables: {
				Read:  format.RoleSupport{Grade: format.SupportNone},
				Write: format.RoleSupport{Grade: format.SupportNone},
			},
			block.RoleSamplerSettings:    sillyTavernSettingSupport(named.samplers),
			block.RoleCompletionSettings: sillyTavernSettingSupport(named.completion),
			block.RoleAdvancedSettings:   sillyTavernSettingSupport(named.advanced),
			block.RolePromptNudges: {
				Read: format.RoleSupport{Grade: format.SupportFull},
				Write: format.RoleSupport{
					Grade:     format.SupportPartial,
					Condition: unnamedNudge(named.nudges),
				},
			},
			block.RoleRegexScripts: {
				Read:  format.RoleSupport{Grade: format.SupportFull},
				Write: format.RoleSupport{Grade: format.SupportFull},
			},
			block.RoleGallery: {
				Read:  format.RoleSupport{Grade: format.SupportNone},
				Write: format.RoleSupport{Grade: format.SupportNone},
			},
			block.RoleCreatorNotes: {
				Read:  format.RoleSupport{Grade: format.SupportNone},
				Write: format.RoleSupport{Grade: format.SupportNone},
			},
		},
		Header: nil,
		Slots:  declaredSlots(SillyTavern),
		Limits: format.ContentLimits{
			PayloadBytes: block.MaxPayloadBytes, CollectionItems: block.MaxCollectionItems,
			ItemBytes: block.MaxItemBytes,
		},
		ConsumedKeys: sillyTavernConsumedKeys(named),
		Boilerplate:  nil,
		Preservation: format.PreservationDeclaration{
			Body: sillyTavernNamespace, Container: []string{stExtensions},
		},
		TestedOrigins: []string{SillyTavernID, format.OriginIllarin},
	}
}

func sillyTavernConsumedKeys(named namedSlots) []string {
	consumed := []string{stPrompts, stOrder}
	for _, group := range [][]slot{named.samplers, named.completion, named.advanced} {
		for _, slot := range group {
			consumed = append(consumed, slot.name)
		}
	}
	return append(consumed, named.nudges...)
}

func sillyTavernSettingSupport(named []slot) format.DirectionalRoleSupport {
	return format.DirectionalRoleSupport{
		Read:  format.RoleSupport{Grade: format.SupportFull},
		Write: format.RoleSupport{Grade: format.SupportPartial, Condition: unnamedSetting(named)},
	}
}

func hasHeadingOrHistoryPlacement(content block.Content) bool {
	list, isList := content.(block.PromptList)
	if !isList {
		return false
	}
	for _, fragment := range list.Fragments {
		if fragment.GroupID != nil ||
			fragment.Placement == block.BeforeHistory ||
			fragment.Placement == block.AfterHistory {
			return true
		}
	}
	return false
}

func (m SillyTavernModule) Claim(file format.Inspection) (format.Claim, bool) {
	return format.ClaimByDeclaration(file, m.Declaration())
}

func (m SillyTavernModule) Parse(
	_ context.Context,
	file format.Inspection,
	claim format.Claim,
) (format.Parsed, error) {
	payload, ok := claim.Payload(file)
	if !ok {
		return format.Parsed{}, fmt.Errorf("%s payload: the claimed payload is missing", SillyTavernID)
	}
	source := maps.Clone(payload.Root)

	var prompts []json.RawMessage
	if raw, present := source[stPrompts]; !present || json.Unmarshal(raw, &prompts) != nil {
		return format.Parsed{}, format.MalformedInput(fmt.Errorf(
			"%s prompts: a preset's prompt fragments have to be a list", SillyTavernID,
		))
	}
	delete(source, stPrompts)

	live, ordered := takeLiveOrder(source)
	list, leftovers := readSillyTavernPrompts(prompts, live, ordered)
	elements := []block.Element{{
		ID: uuid.New(), Type: block.TypePromptList, Role: block.RolePromptFragments,
		Content: list,
	}}

	named := slotsByApp[SillyTavern]
	for _, group := range []struct {
		role  block.Role
		slots []slot
	}{
		{block.RoleSamplerSettings, named.samplers},
		{block.RoleCompletionSettings, named.completion},
		{block.RoleAdvancedSettings, named.advanced},
	} {
		if element, filled := settingsElement(group.role, source, group.slots); filled {
			elements = append(elements, element)
		}
	}
	if element, filled := nudgesElement(source, named.nudges); filled {
		elements = append(elements, element)
	}

	scripts, scriptFields := readSillyTavernScripts(source)
	if len(scripts) > 0 {
		elements = append(elements, block.Element{
			ID: uuid.New(), Type: block.TypeScriptList, Role: block.RoleRegexScripts,
			Content: block.ScriptList{Scripts: scripts},
		})
	}

	return format.Parsed{
		Type: Type, Format: SillyTavernID,
		Elements: elements,
		Remainder: sillyTavernPreservation.remainder(
			source, leftovers,
			scriptLeftovers(scripts, sillyTavernScriptNamespace, scriptFields),
		),
	}, nil
}
