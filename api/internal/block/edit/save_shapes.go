package edit

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/google/uuid"
)

type SaveWorkBlockRequest struct {
	AllowedApps     *[]SaveWorkBlockRequestAllowedApps `json:"allowedApps,omitempty" tstype:"'lumiverse'[]"`
	Elements        []SaveWorkElement                  `json:"elements"`
	ExposeProtected *bool                              `json:"exposeProtected,omitempty"`
	Layout          SaveWorkBlockRequestLayout         `json:"layout"`
	Title           *string                            `json:"title" tstype:"string | null,required"`
	Width           SaveWorkBlockRequestWidth          `json:"width"`
}

type SaveWorkBlockRequestAllowedApps string

const (
	SaveWorkBlockRequestAllowedAppsLumiverse SaveWorkBlockRequestAllowedApps = "lumiverse"
)

type SaveWorkBlockRequestLayout string

const (
	SaveWorkBlockRequestLayoutDuo       SaveWorkBlockRequestLayout = "duo"
	SaveWorkBlockRequestLayoutMainAside SaveWorkBlockRequestLayout = "main-aside"
	SaveWorkBlockRequestLayoutSingle    SaveWorkBlockRequestLayout = "single"
	SaveWorkBlockRequestLayoutStack2    SaveWorkBlockRequestLayout = "stack-2"
	SaveWorkBlockRequestLayoutStack3    SaveWorkBlockRequestLayout = "stack-3"
	SaveWorkBlockRequestLayoutTrio      SaveWorkBlockRequestLayout = "trio"
)

type SaveWorkBlockRequestWidth string

const (
	SaveWorkBlockRequestWidthFull      SaveWorkBlockRequestWidth = "full"
	SaveWorkBlockRequestWidthHalf      SaveWorkBlockRequestWidth = "half"
	SaveWorkBlockRequestWidthThird     SaveWorkBlockRequestWidth = "third"
	SaveWorkBlockRequestWidthTwoThirds SaveWorkBlockRequestWidth = "two_thirds"
)

type SaveWorkElement struct {
	Content  json.RawMessage            `json:"content" tstype:"ProseContent | TextSetContent | FieldListContent | DialogueSampleContent | ImageSetContent | LinkListContent | EntryTableContent | PromptListContent | VariableSchemaContent | SettingGroupContent | ScriptListContent | ColorSetContent | StylesheetSetContent | RecordListContent"`
	Display  *SaveWorkElementDisplay    `json:"display,omitempty"`
	Id       uuid.UUID                  `json:"id"`
	ItemSize *block.WorkElementItemSize `json:"itemSize,omitempty"`
	Role     *string                    `json:"role,omitempty"`
	Slot     string                     `json:"slot"`
	Type     block.ElementType          `json:"type"`
}

type SaveWorkElementDisplay string

const (
	SaveWorkElementDisplayRich     SaveWorkElementDisplay = "rich"
	SaveWorkElementDisplayVerbatim SaveWorkElementDisplay = "verbatim"
)

type SealedExposureRefusal struct {
	Code    SealedExposureRefusalCode `json:"code" tstype:"'sealed_exposure',required"`
	Error   string                    `json:"error"`
	Prompts []string                  `json:"prompts"`
}

type SealedExposureRefusalCode string

const (
	SealedExposureRefusalCodeSealedExposure SealedExposureRefusalCode = "sealed_exposure"
)
