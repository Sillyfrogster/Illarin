package preset

import (
	"encoding/json"
	"slices"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
	"github.com/google/uuid"
)

const Type = "preset"

func Modules() []format.Reader { return []format.Reader{LumiverseModule{}, SillyTavernModule{}} }

func declaredSlots(app App) []format.SlotDeclaration {
	named := slotsByApp[app]
	declared := make([]format.SlotDeclaration, 0,
		len(named.samplers)+len(named.completion)+len(named.advanced)+len(named.nudges))
	for _, group := range [][]slot{named.samplers, named.completion, named.advanced} {
		for _, named := range group {
			declared = append(declared, format.SlotDeclaration{
				Name: named.name, Type: slotValueTypes[named.settingType],
			})
		}
	}
	for _, name := range named.nudges {
		declared = append(declared, format.SlotDeclaration{Name: name, Type: format.ValueString})
	}
	return declared
}

var slotValueTypes = map[block.SettingType]format.ValueType{
	block.SettingNumber:  format.ValueNumber,
	block.SettingBoolean: format.ValueBoolean,
	block.SettingText:    format.ValueString,
	block.SettingStrings: format.ValueArray,
}

func settingsElement(
	role block.Role,
	values map[string]json.RawMessage,
	named []slot,
) (block.Element, bool) {
	settings := make([]block.Setting, 0, len(named))
	for _, slot := range named {
		raw, present := values[slot.name]
		if !present {
			continue
		}
		value, readable := readValue(raw, slot.settingType)
		if !readable {
			continue
		}
		delete(values, slot.name)
		settings = append(settings, block.Setting{
			ID: block.NewItemID(), Name: slot.name, Type: slot.settingType, Value: value,
		})
	}
	if len(settings) == 0 {
		return block.Element{}, false
	}
	return block.Element{
		ID: uuid.New(), Type: block.TypeSettingGroup, Role: role,
		Content: block.SettingGroup{Settings: settings},
	}, true
}

func nudgesElement(values map[string]json.RawMessage, names []string) (block.Element, bool) {
	texts := make([]block.TextItem, 0, len(names))
	for _, name := range names {
		var text string
		if !keys.Take(values, name, &text) {
			continue
		}
		texts = append(texts, block.TextItem{ID: block.NewItemID(), Name: name, Text: text})
	}
	if len(texts) == 0 {
		return block.Element{}, false
	}
	return block.Element{
		ID: uuid.New(), Type: block.TypeTextSet, Role: block.RolePromptNudges,
		Content: block.TextSet{Texts: texts},
	}, true
}

func readValue(raw json.RawMessage, holds block.SettingType) (*block.Value, bool) {
	if keys.IsNull(raw) {
		return nil, true
	}
	switch holds {
	case block.SettingNumber:
		var number float64
		if json.Unmarshal(raw, &number) != nil {
			return nil, false
		}
		return &block.Value{Number: &number}, true
	case block.SettingBoolean:
		var yes bool
		if json.Unmarshal(raw, &yes) != nil {
			return nil, false
		}
		return &block.Value{Boolean: &yes}, true
	case block.SettingText:
		var text string
		if json.Unmarshal(raw, &text) != nil {
			return nil, false
		}
		return &block.Value{Text: &text}, true
	case block.SettingStrings:
		var strings []string
		if json.Unmarshal(raw, &strings) != nil {
			return nil, false
		}
		return &block.Value{Strings: strings}, true
	}
	return nil, false
}

func writeValue(setting block.Setting) json.RawMessage {
	if setting.Value == nil {
		return json.RawMessage("null")
	}
	switch setting.Type {
	case block.SettingNumber:
		if setting.Value.Number != nil {
			return keys.Must(*setting.Value.Number)
		}
	case block.SettingBoolean:
		if setting.Value.Boolean != nil {
			return keys.Must(*setting.Value.Boolean)
		}
	case block.SettingText:
		if setting.Value.Text != nil {
			return keys.Must(*setting.Value.Text)
		}
	case block.SettingStrings:
		return keys.Must(orEmptyStrings(setting.Value.Strings))
	}
	return json.RawMessage("null")
}

func orEmptyStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func settings(work format.ExportWork, role block.Role) []block.Setting {
	content, ok := work.Content(role)
	if !ok {
		return nil
	}
	group, isGroup := content.(block.SettingGroup)
	if !isGroup {
		return nil
	}
	return group.Settings
}

func fragments(work format.ExportWork) block.PromptList {
	content, ok := work.Content(block.RolePromptFragments)
	if !ok {
		return block.PromptList{}
	}
	list, isList := content.(block.PromptList)
	if !isList {
		return block.PromptList{}
	}
	return list
}

func variables(work format.ExportWork) []block.Variable {
	content, ok := work.Content(block.RolePromptVariables)
	if !ok {
		return nil
	}
	schema, isSchema := content.(block.VariableSchema)
	if !isSchema {
		return nil
	}
	return schema.Variables
}

func scripts(work format.ExportWork) []block.Script {
	content, ok := work.Content(block.RoleRegexScripts)
	if !ok {
		return nil
	}
	list, isList := content.(block.ScriptList)
	if !isList {
		return nil
	}
	return list.Scripts
}

func nudges(work format.ExportWork) []block.TextItem {
	content, ok := work.Content(block.RolePromptNudges)
	if !ok {
		return nil
	}
	set, isSet := content.(block.TextSet)
	if !isSet {
		return nil
	}
	return set.Texts
}

func unnamedSetting(named []slot) *format.ContentCondition {
	return &format.ContentCondition{
		Description: "a setting this app has no name for",
		Matches: func(content block.Content) bool {
			group, isGroup := content.(block.SettingGroup)
			if !isGroup {
				return false
			}
			for _, setting := range group.Settings {
				if setting.Value != nil && !slices.ContainsFunc(named, func(s slot) bool {
					return s.name == setting.Name
				}) {
					return true
				}
			}
			return false
		},
	}
}

func unnamedNudge(names []string) *format.ContentCondition {
	return &format.ContentCondition{
		Description: "a nudge this app does not send",
		Matches: func(content block.Content) bool {
			set, isSet := content.(block.TextSet)
			if !isSet {
				return false
			}
			for _, text := range set.Texts {
				if !slices.Contains(names, text.Name) {
					return true
				}
			}
			return false
		},
	}
}

func hasLooseVariable(content block.Content) bool {
	schema, isSchema := content.(block.VariableSchema)
	if !isSchema {
		return false
	}
	for _, variable := range schema.Variables {
		if variable.FragmentID == nil {
			return true
		}
	}
	return false
}
