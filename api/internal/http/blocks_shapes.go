package http

import (
	"encoding/json"

	"github.com/google/uuid"
)

type AssetBlock struct {
	AllowedLayouts []AssetBlockAllowedLayouts `json:"allowedLayouts"`
	Definition     string                     `json:"definition"`
	Elements       []AssetElement             `json:"elements"`
	Hidden         bool                       `json:"hidden"`
	Hideable       bool                       `json:"hideable"`
	Id             uuid.UUID                  `json:"id"`
	IsEmpty        bool                       `json:"isEmpty"`
	Layout         AssetBlockLayout           `json:"layout"`
	Position       int                        `json:"position"`
	Required       bool                       `json:"required"`
	Title          string                     `json:"title"`
	TitleIsDefault bool                       `json:"titleIsDefault"`
	Width          AssetBlockWidth            `json:"width"`
}

type AssetBlockAllowedLayouts string

const (
	AssetBlockAllowedLayoutsDuo       AssetBlockAllowedLayouts = "duo"
	AssetBlockAllowedLayoutsMainAside AssetBlockAllowedLayouts = "main-aside"
	AssetBlockAllowedLayoutsSingle    AssetBlockAllowedLayouts = "single"
	AssetBlockAllowedLayoutsStack2    AssetBlockAllowedLayouts = "stack-2"
	AssetBlockAllowedLayoutsStack3    AssetBlockAllowedLayouts = "stack-3"
	AssetBlockAllowedLayoutsTrio      AssetBlockAllowedLayouts = "trio"
)

type AssetBlockLayout string

const (
	AssetBlockLayoutDuo       AssetBlockLayout = "duo"
	AssetBlockLayoutMainAside AssetBlockLayout = "main-aside"
	AssetBlockLayoutSingle    AssetBlockLayout = "single"
	AssetBlockLayoutStack2    AssetBlockLayout = "stack-2"
	AssetBlockLayoutStack3    AssetBlockLayout = "stack-3"
	AssetBlockLayoutTrio      AssetBlockLayout = "trio"
)

type AssetBlockWidth string

const (
	AssetBlockWidthFull      AssetBlockWidth = "full"
	AssetBlockWidthHalf      AssetBlockWidth = "half"
	AssetBlockWidthThird     AssetBlockWidth = "third"
	AssetBlockWidthTwoThirds AssetBlockWidth = "two_thirds"
)

type AssetElement struct {
	Content  json.RawMessage      `json:"content"`
	Display  *AssetElementDisplay `json:"display,omitempty"`
	Facts    []string             `json:"facts"`
	Id       uuid.UUID            `json:"id"`
	IsEmpty  bool                 `json:"isEmpty"`
	ItemSize *ItemSize            `json:"itemSize,omitempty"`
	Label    string               `json:"label"`
	Locked   bool                 `json:"locked"`
	Pinned   bool                 `json:"pinned"`
	Role     *string              `json:"role,omitempty"`
	Slot     string               `json:"slot"`
	Type     ElementType          `json:"type"`
}

type AssetElementDisplay string

const (
	AssetElementDisplayRich     AssetElementDisplay = "rich"
	AssetElementDisplayVerbatim AssetElementDisplay = "verbatim"
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
	AfterCharacter  EntryTableContentEntriesPosition = "after_character"
	BeforeCharacter EntryTableContentEntriesPosition = "before_character"
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
		Protected *bool                                `json:"protected,omitempty"`
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
	InHistory   PromptListContentFragmentsPlacement = "in_history"
	PostHistory PromptListContentFragmentsPlacement = "post_history"
	PreHistory  PromptListContentFragmentsPlacement = "pre_history"
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
	Records []struct {
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
	Schema RecordListContentSchema `json:"schema"`
}

type RecordListContentRecordsGenderIdentity int

const (
	RecordListContentRecordsGenderIdentityN0 RecordListContentRecordsGenderIdentity = 0
	RecordListContentRecordsGenderIdentityN1 RecordListContentRecordsGenderIdentity = 1
	RecordListContentRecordsGenderIdentityN2 RecordListContentRecordsGenderIdentity = 2
)

type RecordListContentSchema string

const (
	Lumia RecordListContentSchema = "lumia"
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
	Display ScriptListContentScriptsAffects = "display"
	Prompt  ScriptListContentScriptsAffects = "prompt"
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
	Assets []struct {
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
