package block

import (
	"encoding/json"

	"github.com/google/uuid"
)

type WorkBlock struct {
	AllowedLayouts []WorkBlockAllowedLayouts `json:"allowedLayouts"`
	Definition     string                    `json:"definition"`
	Elements       []WorkElement             `json:"elements"`
	Hidden         bool                      `json:"hidden"`
	Hideable       bool                      `json:"hideable"`
	Id             uuid.UUID                 `json:"id"`
	IsEmpty        bool                      `json:"isEmpty"`
	Layout         WorkBlockLayout           `json:"layout"`
	Position       int                       `json:"position"`
	Required       bool                      `json:"required"`
	Title          string                    `json:"title"`
	TitleIsDefault bool                      `json:"titleIsDefault"`
	Width          WorkBlockWidth            `json:"width"`
}

type WorkBlockAllowedLayouts string

const (
	WorkBlockAllowedLayoutsDuo       WorkBlockAllowedLayouts = "duo"
	WorkBlockAllowedLayoutsMainAside WorkBlockAllowedLayouts = "main-aside"
	WorkBlockAllowedLayoutsSingle    WorkBlockAllowedLayouts = "single"
	WorkBlockAllowedLayoutsStack2    WorkBlockAllowedLayouts = "stack-2"
	WorkBlockAllowedLayoutsStack3    WorkBlockAllowedLayouts = "stack-3"
	WorkBlockAllowedLayoutsTrio      WorkBlockAllowedLayouts = "trio"
)

type WorkBlockLayout string

const (
	WorkBlockLayoutDuo       WorkBlockLayout = "duo"
	WorkBlockLayoutMainAside WorkBlockLayout = "main-aside"
	WorkBlockLayoutSingle    WorkBlockLayout = "single"
	WorkBlockLayoutStack2    WorkBlockLayout = "stack-2"
	WorkBlockLayoutStack3    WorkBlockLayout = "stack-3"
	WorkBlockLayoutTrio      WorkBlockLayout = "trio"
)

type WorkBlockWidth string

const (
	WorkBlockWidthFull      WorkBlockWidth = "full"
	WorkBlockWidthHalf      WorkBlockWidth = "half"
	WorkBlockWidthThird     WorkBlockWidth = "third"
	WorkBlockWidthTwoThirds WorkBlockWidth = "two_thirds"
)

type WorkElement struct {
	Content  json.RawMessage      `json:"content" tstype:"ProseContent | TextSetContent | FieldListContent | DialogueSampleContent | ImageSetContent | LinkListContent | EntryTableContent | PromptListContent | VariableSchemaContent | SettingGroupContent | ScriptListContent | ColorSetContent | StylesheetSetContent | RecordListContent"`
	Display  *WorkElementDisplay  `json:"display,omitempty"`
	Facts    []string             `json:"facts"`
	Id       uuid.UUID            `json:"id"`
	IsEmpty  bool                 `json:"isEmpty"`
	ItemSize *WorkElementItemSize `json:"itemSize,omitempty"`
	Label    string               `json:"label"`
	FromFile bool                 `json:"fromFile"`
	Pinned   bool                 `json:"pinned"`
	Role     *string              `json:"role,omitempty"`
	Slot     string               `json:"slot"`
	Type     ElementType          `json:"type"`
}

type WorkElementDisplay string

const (
	WorkElementDisplayRich     WorkElementDisplay = "rich"
	WorkElementDisplayVerbatim WorkElementDisplay = "verbatim"
)

type ColorSetContent struct {
	Modes []struct {
		Colors []struct {
			Id    *uuid.UUID `json:"id,omitempty"`
			Name  string     `json:"name"`
			Value string     `json:"value"`
		} `json:"colors"`
		Name *string `json:"name,omitempty"`
	} `json:"modes"`
}

type EntryTableContent struct {
	Entries []struct {
		CaseSensitive *bool                             `json:"caseSensitive,omitempty"`
		Constant      *bool                             `json:"constant,omitempty"`
		Enabled       bool                              `json:"enabled"`
		Id            *uuid.UUID                        `json:"id,omitempty"`
		Keys          []string                          `json:"keys"`
		Name          *string                           `json:"name,omitempty"`
		Order         *int                              `json:"order,omitempty"`
		Position      *EntryTableContentEntriesPosition `json:"position,omitempty"`
		Recursion     *struct {
			DelayUntil *bool `json:"delayUntil,omitempty"`
			Exclude    *bool `json:"exclude,omitempty"`
			Prevent    *bool `json:"prevent,omitempty"`
		} `json:"recursion,omitempty"`
		SecondaryKeys *[]string `json:"secondaryKeys,omitempty"`
		Selective     *bool     `json:"selective,omitempty"`
		Text          string    `json:"text"`
	} `json:"entries"`
}

type EntryTableContentEntriesPosition string

const (
	EntryTableContentEntriesPositionAfterCharacter  EntryTableContentEntriesPosition = "after_character"
	EntryTableContentEntriesPositionBeforeCharacter EntryTableContentEntriesPosition = "before_character"
)

type PromptListContent struct {
	Fragments []struct {
		Depth     *int                                 `json:"depth,omitempty"`
		Enabled   bool                                 `json:"enabled"`
		GroupId   *uuid.UUID                           `json:"groupId,omitempty"`
		Id        *uuid.UUID                           `json:"id,omitempty"`
		Marker    *string                              `json:"marker,omitempty"`
		Name      *string                              `json:"name,omitempty"`
		Placement *PromptListContentFragmentsPlacement `json:"placement,omitempty"`
		Private   *bool                                `json:"private,omitempty"`
		Role      *PromptListContentFragmentsRole      `json:"role,omitempty"`
		Text      string                               `json:"text"`
	} `json:"fragments"`
	Groups []struct {
		Id   *uuid.UUID `json:"id,omitempty"`
		Name string     `json:"name"`
	} `json:"groups"`
}

type PromptListContentFragmentsPlacement string

const (
	PromptListContentFragmentsPlacementInHistory   PromptListContentFragmentsPlacement = "in_history"
	PromptListContentFragmentsPlacementPostHistory PromptListContentFragmentsPlacement = "post_history"
	PromptListContentFragmentsPlacementPreHistory  PromptListContentFragmentsPlacement = "pre_history"
)

type PromptListContentFragmentsRole string

const (
	PromptListContentFragmentsRoleAssistant       PromptListContentFragmentsRole = "assistant"
	PromptListContentFragmentsRoleAssistantAppend PromptListContentFragmentsRole = "assistant_append"
	PromptListContentFragmentsRoleSystem          PromptListContentFragmentsRole = "system"
	PromptListContentFragmentsRoleUser            PromptListContentFragmentsRole = "user"
	PromptListContentFragmentsRoleUserAppend      PromptListContentFragmentsRole = "user_append"
)

type RecordListContent struct {
	LoomItems []json.RawMessage `json:"loomItems,omitempty"`
	Records   []struct {
		AuthorName       string                                 `json:"authorName"`
		AvatarUrl        *uuid.UUID                             `json:"avatarUrl,omitempty"`
		GenderIdentity   RecordListContentRecordsGenderIdentity `json:"genderIdentity"`
		Id               *uuid.UUID                             `json:"id,omitempty"`
		LumiaBehavior    string                                 `json:"lumiaBehavior"`
		LumiaDefinition  string                                 `json:"lumiaDefinition"`
		LumiaName        string                                 `json:"lumiaName"`
		LumiaPersonality string                                 `json:"lumiaPersonality"`
		Version          int                                    `json:"version"`
	} `json:"records"`
	Schema RecordListContentSchema `json:"schema" tstype:"'lumia',required"`
}

type RecordListContentRecordsGenderIdentity int

const (
	RecordListContentRecordsGenderIdentityN0 RecordListContentRecordsGenderIdentity = 0
	RecordListContentRecordsGenderIdentityN1 RecordListContentRecordsGenderIdentity = 1
	RecordListContentRecordsGenderIdentityN2 RecordListContentRecordsGenderIdentity = 2
)

type RecordListContentSchema string

const (
	RecordListContentSchemaLumia RecordListContentSchema = "lumia"
)

type ScriptListContent struct {
	Scripts []struct {
		Affects     *[]ScriptListContentScriptsAffects `json:"affects,omitempty"`
		Description *string                            `json:"description,omitempty"`
		Enabled     bool                               `json:"enabled"`
		Find        string                             `json:"find"`
		Flags       *string                            `json:"flags,omitempty"`
		Id          *uuid.UUID                         `json:"id,omitempty"`
		MaxDepth    *int                               `json:"maxDepth,omitempty"`
		MinDepth    *int                               `json:"minDepth,omitempty"`
		Name        *string                            `json:"name,omitempty"`
		Replace     string                             `json:"replace"`
		RunOnEdit   *bool                              `json:"runOnEdit,omitempty"`
		Targets     *[]ScriptListContentScriptsTargets `json:"targets,omitempty"`
		Trim        *[]string                          `json:"trim,omitempty"`
	} `json:"scripts"`
}

type ScriptListContentScriptsAffects string

const (
	ScriptListContentScriptsAffectsDisplay ScriptListContentScriptsAffects = "display"
	ScriptListContentScriptsAffectsPrompt  ScriptListContentScriptsAffects = "prompt"
)

type ScriptListContentScriptsTargets string

const (
	ScriptListContentScriptsTargetsLorebook     ScriptListContentScriptsTargets = "lorebook"
	ScriptListContentScriptsTargetsModelOutput  ScriptListContentScriptsTargets = "model_output"
	ScriptListContentScriptsTargetsSlashCommand ScriptListContentScriptsTargets = "slash_command"
	ScriptListContentScriptsTargetsUserInput    ScriptListContentScriptsTargets = "user_input"
)

type SettingGroupContent struct {
	Settings []struct {
		Choices *[]string                       `json:"choices,omitempty"`
		Id      *uuid.UUID                      `json:"id,omitempty"`
		Label   *string                         `json:"label,omitempty"`
		Name    string                          `json:"name"`
		Type    SettingGroupContentSettingsType `json:"type"`
		Value   *TypedValue                     `json:"value,omitempty"`
	} `json:"settings"`
}

type SettingGroupContentSettingsType string

const (
	SettingGroupContentSettingsTypeBoolean    SettingGroupContentSettingsType = "boolean"
	SettingGroupContentSettingsTypeNumber     SettingGroupContentSettingsType = "number"
	SettingGroupContentSettingsTypeStringList SettingGroupContentSettingsType = "string_list"
	SettingGroupContentSettingsTypeText       SettingGroupContentSettingsType = "text"
)

type StylesheetSetContent struct {
	Works []struct {
		Data      []byte     `json:"data"`
		Id        *uuid.UUID `json:"id,omitempty"`
		MediaType *string    `json:"mediaType,omitempty"`
		Path      string     `json:"path"`
	} `json:"assets"`
	Global      string `json:"global"`
	Stylesheets []struct {
		Css     string     `json:"css"`
		Enabled bool       `json:"enabled"`
		Id      *uuid.UUID `json:"id,omitempty"`
		Name    string     `json:"name"`
	} `json:"stylesheets"`
}

type TypedValue struct {
	Boolean *bool     `json:"boolean,omitempty"`
	Number  *float32  `json:"number,omitempty"`
	Strings *[]string `json:"strings,omitempty"`
	Text    *string   `json:"text,omitempty"`
}

type VariableSchemaContent struct {
	Variables []struct {
		Default     *TypedValue `json:"default,omitempty"`
		Description *string     `json:"description,omitempty"`
		FragmentId  *uuid.UUID  `json:"fragmentId,omitempty"`
		Id          *uuid.UUID  `json:"id,omitempty"`
		Label       *string     `json:"label,omitempty"`
		Name        string      `json:"name"`
		Options     *[]struct {
			Key   *string `json:"key,omitempty"`
			Label string  `json:"label"`
			Value string  `json:"value"`
		} `json:"options,omitempty"`
		Range *struct {
			Max  *float32 `json:"max,omitempty"`
			Min  *float32 `json:"min,omitempty"`
			Step *float32 `json:"step,omitempty"`
		} `json:"range,omitempty"`
		Rows      *int                                 `json:"rows,omitempty"`
		Separator *string                              `json:"separator,omitempty"`
		Value     *TypedValue                          `json:"value,omitempty"`
		Widget    VariableSchemaContentVariablesWidget `json:"widget"`
	} `json:"variables"`
}

type VariableSchemaContentVariablesWidget string

const (
	VariableSchemaContentVariablesWidgetMultiselect VariableSchemaContentVariablesWidget = "multiselect"
	VariableSchemaContentVariablesWidgetNumber      VariableSchemaContentVariablesWidget = "number"
	VariableSchemaContentVariablesWidgetSelect      VariableSchemaContentVariablesWidget = "select"
	VariableSchemaContentVariablesWidgetSlider      VariableSchemaContentVariablesWidget = "slider"
	VariableSchemaContentVariablesWidgetSwitch      VariableSchemaContentVariablesWidget = "switch"
	VariableSchemaContentVariablesWidgetText        VariableSchemaContentVariablesWidget = "text"
	VariableSchemaContentVariablesWidgetTextarea    VariableSchemaContentVariablesWidget = "textarea"
)

type ProseContent struct {
	Text string `json:"text"`
}

type TextSetContent struct {
	Texts []struct {
		Id   *uuid.UUID `json:"id,omitempty"`
		Name *string    `json:"name,omitempty"`
		Text string     `json:"text"`
	} `json:"texts"`
}

type DialogueSampleContent struct {
	Turns []struct {
		Id      *uuid.UUID `json:"id,omitempty"`
		Speaker string     `json:"speaker"`
		Text    string     `json:"text"`
	} `json:"turns"`
}

type ImageSetContent struct {
	Images []struct {
		Id                *uuid.UUID `json:"id,omitempty"`
		MediaId           uuid.UUID  `json:"mediaId"`
		Name              *string    `json:"name,omitempty"`
		OmitFromDownloads *bool      `json:"omitFromDownloads,omitempty"`
	} `json:"images"`
}

type FieldListContent struct {
	Fields []struct {
		Id    *uuid.UUID `json:"id,omitempty"`
		Name  *string    `json:"name,omitempty"`
		Value string     `json:"value"`
	} `json:"fields"`
}

type LinkListContent struct {
	Links []struct {
		Id    *uuid.UUID `json:"id,omitempty"`
		Label *string    `json:"label,omitempty"`
		Note  *string    `json:"note,omitempty"`
		Url   string     `json:"url"`
	} `json:"links"`
}

type WorkElementItemSize string

const (
	WorkElementItemSizeLarge  WorkElementItemSize = "large"
	WorkElementItemSizeMedium WorkElementItemSize = "medium"
	WorkElementItemSizeSmall  WorkElementItemSize = "small"
)

type ElementType string

const (
	ElementTypeColorSet       ElementType = "color_set"
	ElementTypeDialogueSample ElementType = "dialogue_sample"
	ElementTypeEntryTable     ElementType = "entry_table"
	ElementTypeFieldList      ElementType = "field_list"
	ElementTypeImageSet       ElementType = "image_set"
	ElementTypeLinkList       ElementType = "link_list"
	ElementTypePromptList     ElementType = "prompt_list"
	ElementTypeProse          ElementType = "prose"
	ElementTypeRecordList     ElementType = "record_list"
	ElementTypeScriptList     ElementType = "script_list"
	ElementTypeSettingGroup   ElementType = "setting_group"
	ElementTypeStylesheetSet  ElementType = "stylesheet_set"
	ElementTypeTextSet        ElementType = "text_set"
	ElementTypeVariableSchema ElementType = "variable_schema"
)
