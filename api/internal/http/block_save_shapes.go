package http

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/google/uuid"
)

type SaveAssetBlockRequest struct {
	AllowedApps     *[]SaveAssetBlockRequestAllowedApps `json:"allowedApps,omitempty" tstype:"'lumiverse'[]"`
	Elements        []SaveAssetElement                  `json:"elements"`
	ExposeProtected *bool                               `json:"exposeProtected,omitempty"`
	Layout          SaveAssetBlockRequestLayout         `json:"layout"`
	Title           *string                             `json:"title" tstype:"string | null,required"`
	Width           SaveAssetBlockRequestWidth          `json:"width"`
}

type SaveAssetBlockRequestAllowedApps string

const (
	SaveAssetBlockRequestAllowedAppsLumiverse SaveAssetBlockRequestAllowedApps = "lumiverse"
)

type SaveAssetBlockRequestLayout string

const (
	SaveAssetBlockRequestLayoutDuo       SaveAssetBlockRequestLayout = "duo"
	SaveAssetBlockRequestLayoutMainAside SaveAssetBlockRequestLayout = "main-aside"
	SaveAssetBlockRequestLayoutSingle    SaveAssetBlockRequestLayout = "single"
	SaveAssetBlockRequestLayoutStack2    SaveAssetBlockRequestLayout = "stack-2"
	SaveAssetBlockRequestLayoutStack3    SaveAssetBlockRequestLayout = "stack-3"
	SaveAssetBlockRequestLayoutTrio      SaveAssetBlockRequestLayout = "trio"
)

type SaveAssetBlockRequestWidth string

const (
	SaveAssetBlockRequestWidthFull      SaveAssetBlockRequestWidth = "full"
	SaveAssetBlockRequestWidthHalf      SaveAssetBlockRequestWidth = "half"
	SaveAssetBlockRequestWidthThird     SaveAssetBlockRequestWidth = "third"
	SaveAssetBlockRequestWidthTwoThirds SaveAssetBlockRequestWidth = "two_thirds"
)

type SaveAssetElement struct {
	Content  json.RawMessage          `json:"content" tstype:"ProseContent | TextSetContent | FieldListContent | DialogueSampleContent | ImageSetContent | LinkListContent | EntryTableContent | PromptListContent | VariableSchemaContent | SettingGroupContent | ScriptListContent | ColorSetContent | StylesheetSetContent | RecordListContent"`
	Display  *SaveAssetElementDisplay `json:"display,omitempty"`
	Id       uuid.UUID                `json:"id"`
	ItemSize *work.ItemSize           `json:"itemSize,omitempty"`
	Role     *string                  `json:"role,omitempty"`
	Slot     string                   `json:"slot"`
	Type     work.ElementType         `json:"type"`
}

type SaveAssetElementDisplay string

const (
	SaveAssetElementDisplayRich     SaveAssetElementDisplay = "rich"
	SaveAssetElementDisplayVerbatim SaveAssetElementDisplay = "verbatim"
)
