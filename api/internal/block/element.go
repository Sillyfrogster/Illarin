package block

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/google/uuid"
)

type Type string

const (
	TypeProse          Type = "prose"
	TypeTextSet        Type = "text_set"
	TypeFieldList      Type = "field_list"
	TypeDialogueSample Type = "dialogue_sample"
	TypeImageSet       Type = "image_set"
	TypeLinkList       Type = "link_list"
	TypeEntryTable     Type = "entry_table"
	TypePromptList     Type = "prompt_list"
	TypeVariableSchema Type = "variable_schema"
	TypeSettingGroup   Type = "setting_group"
	TypeScriptList     Type = "script_list"
	TypeColorSet       Type = "color_set"
	TypeStylesheetSet  Type = "stylesheet_set"
	TypeRecordList     Type = "record_list"
)

type Role string

const (
	RoleDescription             Role = "description"
	RolePersonality             Role = "personality"
	RoleScenario                Role = "scenario"
	RoleGreetings               Role = "greetings"
	RoleGroupGreetings          Role = "group_greetings"
	RoleExampleDialogue         Role = "example_dialogue"
	RoleSystemPrompt            Role = "system_prompt"
	RolePostHistoryInstructions Role = "post_history_instructions"
	RoleCreatorNotes            Role = "creator_notes"
	RoleGallery                 Role = "gallery"
	RoleExpressions             Role = "expressions"
	RoleLorebookEntries         Role = "lorebook_entries"
	RolePromptFragments         Role = "prompt_fragments"
	RolePromptVariables         Role = "prompt_variables"
	RoleSamplerSettings         Role = "sampler_settings"
	RoleCompletionSettings      Role = "completion_settings"
	RoleAdvancedSettings        Role = "advanced_settings"
	RolePromptNudges            Role = "prompt_nudges"
	RoleRegexScripts            Role = "regex_scripts"
	RoleThemeTokens             Role = "theme_tokens"
	RoleThemeControls           Role = "theme_controls"
	RoleStylesheets             Role = "stylesheets"
	RolePackItems               Role = "pack_items"
)

func Roles() []Role {
	return []Role{
		RoleDescription, RolePersonality, RoleScenario, RoleGreetings,
		RoleGroupGreetings, RoleExampleDialogue, RoleSystemPrompt,
		RolePostHistoryInstructions, RoleCreatorNotes, RoleGallery,
		RoleExpressions, RoleLorebookEntries,
		RolePromptFragments, RolePromptVariables, RoleSamplerSettings,
		RoleCompletionSettings, RoleAdvancedSettings, RolePromptNudges,
		RoleRegexScripts, RoleThemeTokens, RoleThemeControls, RoleStylesheets,
		RolePackItems,
	}
}

func (r Role) Known() bool {
	switch r {
	case RoleDescription, RolePersonality, RoleScenario, RoleGreetings,
		RoleGroupGreetings, RoleExampleDialogue, RoleSystemPrompt,
		RolePostHistoryInstructions, RoleCreatorNotes, RoleGallery,
		RoleExpressions, RoleLorebookEntries, RolePromptFragments,
		RolePromptVariables, RoleSamplerSettings, RoleCompletionSettings,
		RoleAdvancedSettings, RolePromptNudges, RoleRegexScripts,
		RoleThemeTokens, RoleThemeControls, RoleStylesheets, RolePackItems:
		return true
	default:
		return false
	}
}

type Cardinality int

const (
	Singular Cardinality = iota
	Repeatable
)

func (r Role) Cardinality() Cardinality {
	if r == RoleGallery {
		return Repeatable
	}
	return Singular
}

type Display string

const (
	DisplayRich     Display = "rich"
	DisplayVerbatim Display = "verbatim"
)

func (d Display) Known() bool {
	return d == DisplayRich || d == DisplayVerbatim
}

type ItemSize string

const (
	ItemSmall  ItemSize = "small"
	ItemMedium ItemSize = "medium"
	ItemLarge  ItemSize = "large"
)

func (s ItemSize) Known() bool {
	return s == ItemSmall || s == ItemMedium || s == ItemLarge
}

func ItemSizes() []ItemSize { return []ItemSize{ItemSmall, ItemMedium, ItemLarge} }

type Options struct {
	Display  Display  `json:"display,omitempty"`
	ItemSize ItemSize `json:"itemSize,omitempty"`
}

type Slot string

type Element struct {
	ID      uuid.UUID
	Type    Type
	Role    Role
	Slot    Slot
	Options Options
	Content Content
}

type Content interface {
	Empty() bool
}

type Prose struct {
	Text string `json:"text"`
}

func (p Prose) Empty() bool { return p.Text == "" }

type TextSet struct {
	Texts []TextItem `json:"texts"`
}

type TextItem struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name,omitempty"`
	Text string    `json:"text"`
}

func (s TextSet) Empty() bool {
	for _, item := range s.Texts {
		if item.Text != "" {
			return false
		}
	}
	return true
}

type DialogueSample struct {
	Turns []DialogueTurn `json:"turns"`
}

type DialogueTurn struct {
	ID      uuid.UUID `json:"id"`
	Speaker string    `json:"speaker"`
	Text    string    `json:"text"`
}

func (d DialogueSample) Empty() bool { return len(d.Turns) == 0 }

type ImageSet struct {
	Images []ImageItem `json:"images"`
}

type ImageItem struct {
	ID                uuid.UUID `json:"id"`
	MediaID           uuid.UUID `json:"mediaId"`
	Name              string    `json:"name,omitempty"`
	OmitFromDownloads bool      `json:"omitFromDownloads,omitempty"`
}

func (s ImageSet) Empty() bool { return len(s.Images) == 0 }

type FieldList struct {
	Fields []FieldItem `json:"fields"`
}

type FieldItem struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name,omitempty"`
	Value string    `json:"value"`
}

func (l FieldList) Empty() bool {
	for _, field := range l.Fields {
		if field.Value != "" {
			return false
		}
	}
	return true
}

type LinkList struct {
	Links []LinkItem `json:"links"`
}

type LinkItem struct {
	ID    uuid.UUID `json:"id"`
	Label string    `json:"label,omitempty"`
	URL   string    `json:"url"`
	Note  string    `json:"note,omitempty"`
}

func (l LinkList) Empty() bool {
	for _, link := range l.Links {
		if link.URL != "" {
			return false
		}
	}
	return true
}

func (t Type) Empty() (Content, error) {
	known, ok := schemas[t]
	if !ok {
		return nil, fmt.Errorf("no element type %q", t)
	}
	return known.empty(), nil
}

func (t Type) Known() bool {
	_, ok := schemas[t]
	return ok
}

func DecodeContent(elementType Type, raw json.RawMessage) (Content, error) {
	if !elementType.Known() {
		return nil, fmt.Errorf("no element type %q", elementType)
	}
	switch elementType {
	case TypeProse:
		var incoming struct {
			Text *string `json:"text"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Text == nil {
			return nil, fmt.Errorf("text must be present as a string")
		}
		return Prose{Text: *incoming.Text}, nil
	case TypeTextSet:
		var incoming struct {
			Texts *[]struct {
				ID   uuid.UUID `json:"id,omitempty"`
				Name string    `json:"name,omitempty"`
				Text *string   `json:"text"`
			} `json:"texts"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Texts == nil {
			return nil, fmt.Errorf("texts must be present as a list")
		}
		texts := make([]TextItem, len(*incoming.Texts))
		for i, item := range *incoming.Texts {
			if item.Text == nil {
				return nil, fmt.Errorf("text %d must include text as a string", i+1)
			}
			texts[i] = TextItem{ID: itemID(item.ID), Name: item.Name, Text: *item.Text}
		}
		return TextSet{Texts: texts}, nil
	case TypeDialogueSample:
		var incoming struct {
			Turns *[]struct {
				ID      uuid.UUID `json:"id,omitempty"`
				Speaker *string   `json:"speaker"`
				Text    *string   `json:"text"`
			} `json:"turns"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Turns == nil {
			return nil, fmt.Errorf("turns must be present as a list")
		}
		turns := make([]DialogueTurn, len(*incoming.Turns))
		for i, turn := range *incoming.Turns {
			if turn.Speaker == nil || turn.Text == nil {
				return nil, fmt.Errorf("turn %d must include speaker and text as strings", i+1)
			}
			turns[i] = DialogueTurn{ID: itemID(turn.ID), Speaker: *turn.Speaker, Text: *turn.Text}
		}
		return DialogueSample{Turns: turns}, nil
	case TypeFieldList:
		var incoming struct {
			Fields *[]struct {
				ID    uuid.UUID `json:"id,omitempty"`
				Name  string    `json:"name,omitempty"`
				Value *string   `json:"value"`
			} `json:"fields"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Fields == nil {
			return nil, fmt.Errorf("fields must be present as a list")
		}
		fields := make([]FieldItem, len(*incoming.Fields))
		for i, field := range *incoming.Fields {
			if field.Value == nil {
				return nil, fmt.Errorf("field %d must include value as a string", i+1)
			}
			fields[i] = FieldItem{ID: itemID(field.ID), Name: field.Name, Value: *field.Value}
		}
		return FieldList{Fields: fields}, nil
	case TypeLinkList:
		var incoming struct {
			Links *[]LinkItem `json:"links"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Links == nil {
			return nil, fmt.Errorf("links must be present as a list")
		}
		links := *incoming.Links
		for i := range links {
			if err := checkWebAddress(links[i].URL); err != nil {
				return nil, fmt.Errorf("link %d: %w", i+1, err)
			}
			links[i].ID = itemID(links[i].ID)
		}
		return LinkList{Links: links}, nil
	case TypeEntryTable:
		return decodeEntryTable(raw)
	case TypePromptList:
		return decodePromptList(raw)
	case TypeVariableSchema:
		return decodeVariableSchema(raw)
	case TypeSettingGroup:
		return decodeSettingGroup(raw)
	case TypeScriptList:
		return decodeScriptList(raw)
	case TypeColorSet:
		return decodeColorSet(raw)
	case TypeStylesheetSet:
		return decodeStylesheetSet(raw)
	case TypeRecordList:
		return decodeRecordList(raw)
	case TypeImageSet:
		var incoming struct {
			Images *[]ImageItem `json:"images"`
		}
		if err := decodeContentJSON(raw, &incoming); err != nil {
			return nil, err
		}
		if incoming.Images == nil {
			return nil, fmt.Errorf("images must be present as a list")
		}
		images := *incoming.Images
		for i := range images {
			if images[i].MediaID == uuid.Nil {
				return nil, fmt.Errorf("image %d must include a media id", i+1)
			}
			images[i].ID = itemID(images[i].ID)
		}
		return ImageSet{Images: images}, nil
	default:
		return nil, fmt.Errorf("no element type %q", elementType)
	}
}

func checkWebAddress(address string) error {
	parsed, err := url.Parse(address)
	if err != nil {
		return fmt.Errorf("%q is not an address", address)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("a link must start with http or https")
	}
	if parsed.Host == "" {
		return fmt.Errorf("a link must name a site")
	}
	return nil
}

func decodeContentJSON(raw json.RawMessage, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("content must be one JSON value")
	}
	return nil
}

type wireElement struct {
	ID      uuid.UUID       `json:"id"`
	Type    Type            `json:"type"`
	Role    Role            `json:"role,omitempty"`
	Slot    Slot            `json:"slot"`
	Version int             `json:"version"`
	Options Options         `json:"options"`
	Content json.RawMessage `json:"content"`
}

func (e Element) MarshalJSON() ([]byte, error) {
	known, ok := schemas[e.Type]
	if !ok {
		return nil, fmt.Errorf("no element type %q", e.Type)
	}
	content, err := json.Marshal(withEmptyCollections(e.Content))
	if err != nil {
		return nil, fmt.Errorf("write %s content: %w", e.Type, err)
	}
	return json.Marshal(wireElement{
		ID: e.ID, Type: e.Type, Role: e.Role, Slot: e.Slot,
		Version: known.version(), Options: e.Options, Content: content,
	})
}

func (e Element) ContentJSON() (json.RawMessage, error) {
	body, err := json.Marshal(withEmptyCollections(e.Content))
	if err != nil {
		return nil, fmt.Errorf("write %s content: %w", e.Type, err)
	}
	return body, nil
}

func (e *Element) UnmarshalJSON(data []byte) error {
	var stored wireElement
	if err := json.Unmarshal(data, &stored); err != nil {
		return err
	}
	known, ok := schemas[stored.Type]
	if !ok {
		return fmt.Errorf("no element type %q", stored.Type)
	}
	content, err := known.read(stored.Version, stored.Content)
	if err != nil {
		return fmt.Errorf("read %s content: %w", stored.Type, err)
	}
	*e = Element{
		ID: stored.ID, Type: stored.Type, Role: stored.Role, Slot: stored.Slot,
		Options: stored.Options, Content: content,
	}
	return nil
}

var labels = map[Role]string{
	RoleDescription:     "Description",
	RolePersonality:     "Personality",
	RoleScenario:        "Scenario",
	RoleGreetings:       "Greetings",
	RoleGroupGreetings:  "Group-only greetings",
	RoleExampleDialogue: "Example dialogue",
	RoleGallery:         "Images",

	RoleSystemPrompt:            "System prompt",
	RolePostHistoryInstructions: "Post-history instructions",
	RoleCreatorNotes:            "Author’s notes",
	RoleExpressions:             "Expressions",
	RoleLorebookEntries:         "Entries",

	RolePromptFragments:    "Prompt fragments",
	RolePromptVariables:    "Variables",
	RoleSamplerSettings:    "Samplers",
	RoleCompletionSettings: "Completion",
	RoleAdvancedSettings:   "Advanced",
	RolePromptNudges:       "Nudges",
	RoleRegexScripts:       "Regex scripts",
	RoleThemeTokens:        "Palette",
	RoleThemeControls:      "Theme controls",
	RoleStylesheets:        "Stylesheets",
	RolePackItems:          "Items",
}

var typeLabels = map[Type]string{
	TypeProse:          "Text",
	TypeTextSet:        "List",
	TypeFieldList:      "Details",
	TypeDialogueSample: "Dialogue",
	TypeImageSet:       "Images",
	TypeLinkList:       "Links",
	TypeEntryTable:     "Entries",
	TypePromptList:     "Prompt fragments",
	TypeVariableSchema: "Variables",
	TypeSettingGroup:   "Settings",
	TypeScriptList:     "Regex scripts",
	TypeColorSet:       "Palette",
	TypeStylesheetSet:  "Stylesheets",
	TypeRecordList:     "Records",
}

func (e Element) Label() string {
	if label := e.Role.Label(); label != "" {
		return label
	}
	return typeLabels[e.Type]
}

var roleTypes = map[Role][]Type{
	RoleDescription:     {TypeProse},
	RolePersonality:     {TypeProse},
	RoleScenario:        {TypeProse},
	RoleGreetings:       {TypeTextSet},
	RoleGroupGreetings:  {TypeTextSet},
	RoleExampleDialogue: {TypeDialogueSample},
	RoleGallery:         {TypeImageSet},

	RoleSystemPrompt:            {TypeProse},
	RolePostHistoryInstructions: {TypeProse},
	RoleCreatorNotes:            {TypeProse},
	RoleExpressions:             {TypeImageSet},
	RoleLorebookEntries:         {TypeEntryTable},

	RolePromptFragments:    {TypePromptList},
	RolePromptVariables:    {TypeVariableSchema},
	RoleSamplerSettings:    {TypeSettingGroup},
	RoleCompletionSettings: {TypeSettingGroup},
	RoleAdvancedSettings:   {TypeSettingGroup},
	RolePromptNudges:       {TypeTextSet},
	RoleRegexScripts:       {TypeScriptList},
	RoleThemeTokens:        {TypeColorSet},
	RoleThemeControls:      {TypeSettingGroup},
	RoleStylesheets:        {TypeStylesheetSet},
	RolePackItems:          {TypeRecordList},
}

func (r Role) Label() string { return labels[r] }

func (r Role) Allows(elementType Type) bool {
	allowed, ok := roleTypes[r]
	if !ok {
		return false
	}
	for _, candidate := range allowed {
		if candidate == elementType {
			return true
		}
	}
	return false
}

func (r Role) AllowedTypes() []Type { return roleTypes[r] }
