package block

import (
	"fmt"
	"strconv"
)

func (e Element) Facts() []string {
	switch content := e.Content.(type) {
	case EntryTable:
		return e.bookFacts(content)
	case PromptList:
		return e.fragmentFacts(content)
	case SettingGroup:
		return e.settingFacts(content)
	case VariableSchema:
		return e.itemFacts(len(content.Variables))
	case ScriptList:
		return e.itemFacts(len(content.Scripts))
	case ImageSet:
		return e.itemFacts(len(content.Images))
	case TextSet:
		return e.itemFacts(len(content.Texts))
	case DialogueSample:
		return e.itemFacts(len(content.Turns))
	case FieldList:
		return e.itemFacts(len(content.Fields))
	case LinkList:
		return e.itemFacts(len(content.Links))
	case ColorSet:
		count := 0
		for _, mode := range content.Modes {
			count += len(mode.Colors)
		}
		return e.itemFacts(count)
	case StylesheetSet:
		count := 0
		if content.Global != "" {
			count++
		}
		return e.itemFacts(count + len(content.Stylesheets))
	case RecordList:
		return e.itemFacts(len(content.Records))
	default:
		return nil
	}
}

func (e Element) bookFacts(book EntryTable) []string {
	facts := e.itemFacts(len(book.Entries))
	if len(facts) == 0 {
		return nil
	}
	on := 0
	for _, entry := range book.Entries {
		if entry.Enabled {
			on++
		}
	}
	if on == len(book.Entries) {
		return facts
	}
	return append(facts, fmt.Sprintf("%s switched on", number(on)))
}

func (e Element) fragmentFacts(list PromptList) []string {
	facts := e.itemFacts(len(list.Fragments))
	if len(facts) == 0 {
		return nil
	}
	on := 0
	for _, fragment := range list.Fragments {
		if fragment.Enabled {
			on++
		}
	}
	if on == len(list.Fragments) {
		return facts
	}
	return append(facts, fmt.Sprintf("%s switched on", number(on)))
}

func (e Element) settingFacts(group SettingGroup) []string {
	return e.itemFacts(group.Supplied())
}

func (e Element) itemFacts(count int) []string {
	if count == 0 {
		return nil
	}
	singular, plural := e.itemNoun()
	noun := plural
	if count == 1 {
		noun = singular
	}
	return []string{fmt.Sprintf("%s %s", number(count), noun)}
}

func (e Element) itemNoun() (string, string) {
	switch e.Role {
	case RoleGreetings:
		return "greeting", "greetings"
	case RoleGroupGreetings:
		return "group-only greeting", "group-only greetings"
	case RoleExpressions:
		return "expression", "expressions"
	case RoleLorebookEntries:
		return "entry", "entries"
	case RolePromptNudges:
		return "nudge", "nudges"
	case RolePackItems:
		return "item", "items"
	}
	switch e.Type {
	case TypeImageSet:
		return "image", "images"
	case TypeDialogueSample:
		return "turn", "turns"
	case TypeFieldList:
		return "detail", "details"
	case TypeLinkList:
		return "link", "links"
	case TypeEntryTable:
		return "entry", "entries"
	case TypePromptList:
		return "fragment", "fragments"
	case TypeVariableSchema:
		return "variable", "variables"
	case TypeSettingGroup:
		return "setting", "settings"
	case TypeScriptList:
		return "script", "scripts"
	case TypeColorSet:
		return "colour", "colours"
	case TypeStylesheetSet:
		return "stylesheet", "stylesheets"
	case TypeRecordList:
		return "record", "records"
	default:
		return "item", "items"
	}
}

func number(count int) string {
	digits := strconv.Itoa(count)
	for cut := len(digits) - 3; cut > 0; cut -= 3 {
		digits = digits[:cut] + "," + digits[cut:]
	}
	return digits
}
