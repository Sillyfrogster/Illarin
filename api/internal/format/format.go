package format

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
)

type Media struct {
	Role        media.Role
	ImageID     uint32
	ElementRole block.Role
	Name        string
}

type Parsed struct {
	Type      string
	Format    string
	Tags      []string
	IsNSFW    *bool
	Media     []Media
	CreatedAt *time.Time
	Header    Header
	Elements  []block.Element
	Remainder []Remainder
	Protected []ProtectedPrompt
	Readme    *Readme
}

// Readme is the README a repository shows, with the archive folder the repository sits in and the README's own folder inside it.
type Readme struct {
	Text         string
	Root, Folder string
}

type ProtectedPrompt struct {
	FragmentID    uuid.UUID
	SourceKey     string
	Text          string
	ReuseExisting bool
}

type Header struct {
	Name           string
	Blurb          string
	WorkVersion    string
	CreditedAuthor string
	Nickname       string
	Identifier     string
}

const MaxBlurbRunes = 400

type Owner string

const (
	OwnerWork    Owner = "asset"
	OwnerElement Owner = "element"
	OwnerItem    Owner = "item"
)

type Remainder struct {
	Owner     Owner
	OwnerID   uuid.UUID
	Namespace string
	Payload   []byte
}

type Direction struct {
	Read  bool
	Write bool
}

type Input string

const (
	InputFile        Input = ""
	InputDatabaseRow Input = "database_row"
)

type ColumnDispositionType string

const (
	ColumnMapped    ColumnDispositionType = "mapped"
	ColumnPreserved ColumnDispositionType = "preserved"
	ColumnDropped   ColumnDispositionType = "dropped"
)

type ColumnDisposition struct {
	Table       string
	Column      string
	Disposition ColumnDispositionType
	Destination string
	Reason      string
}

func MappedColumn(table, column, destination string) ColumnDisposition {
	return ColumnDisposition{
		Table: table, Column: column, Disposition: ColumnMapped, Destination: destination,
	}
}

func PreservedColumn(table, column, destination string) ColumnDisposition {
	return ColumnDisposition{
		Table: table, Column: column, Disposition: ColumnPreserved, Destination: destination,
	}
}

func DroppedColumn(table, column, reason string) ColumnDisposition {
	return ColumnDisposition{
		Table: table, Column: column, Disposition: ColumnDropped, Reason: reason,
	}
}

type AnomalyDisposition string

const (
	AnomalyTolerated AnomalyDisposition = "tolerated"
	AnomalyFatal     AnomalyDisposition = "fatal"
)

type AnomalyDeclaration struct {
	Type        string
	Disposition AnomalyDisposition
	Reason      string
}

type RecognitionType string

const (
	RecognitionDiscriminator RecognitionType = "discriminator"
	RecognitionSignature     RecognitionType = "signature"
	RecognitionEntry         RecognitionType = "entry"
)

type ValueType string

const (
	ValueString  ValueType = "string"
	ValueNumber  ValueType = "number"
	ValueBoolean ValueType = "boolean"
	ValueObject  ValueType = "object"
	ValueArray   ValueType = "array"
)

type Recognition struct {
	Type         RecognitionType
	Containers   []Container
	Path         []string
	Entry        string
	Values       []string
	Required     map[string]ValueType
	LegacyOnly   bool
	SupersededBy []string
}

func ClaimByDeclaration(file Inspection, declaration Declaration) (Claim, bool) {
	for _, recognition := range declaration.Recognition {
		if supersededInFile(file, recognition) {
			continue
		}
		for _, payload := range file.Payloads {
			if len(recognition.Containers) > 0 &&
				!slices.Contains(recognition.Containers, payload.Locator.Container) {
				continue
			}
			switch recognition.Type {
			case RecognitionDiscriminator:
				value, ok := payloadValue(payload.Root, recognition.Path)
				if !ok || !slices.Contains(recognition.Values, value) {
					continue
				}
				return Claim{
					payloadID: payload.ID, strength: authoritative, formatID: declaration.ID,
				}, true
			case RecognitionSignature:
				if recognition.LegacyOnly {
					if spec, _ := payload.String("spec"); spec != "" {
						continue
					}
				}
				if signatureMatches(payload.Root, recognition.Required) {
					return CompatibilityClaim(payload), true
				}
			case RecognitionEntry:
				if payload.Locator.Name == file.ArchiveBase+recognition.Entry {
					return CompatibilityClaim(payload), true
				}
			}
		}
	}
	return Claim{}, false
}

func supersededInFile(file Inspection, recognition Recognition) bool {
	if len(recognition.SupersededBy) == 0 {
		return false
	}
	for _, payload := range file.Payloads {
		if len(recognition.Containers) > 0 &&
			!slices.Contains(recognition.Containers, payload.Locator.Container) {
			continue
		}
		value, ok := payloadValue(payload.Root, recognition.Path)
		if ok && slices.Contains(recognition.SupersededBy, value) {
			return true
		}
	}
	return false
}

func payloadValue(root map[string]json.RawMessage, path []string) (string, bool) {
	current := root
	for i, part := range path {
		raw, ok := current[part]
		if !ok {
			return "", false
		}
		if i == len(path)-1 {
			var value string
			if json.Unmarshal(raw, &value) == nil {
				return value, true
			}
			if valueType := jsonValueType(raw); valueType == ValueNumber || valueType == ValueBoolean {
				return string(bytes.TrimSpace(raw)), true
			}
			return "", false
		}
		if json.Unmarshal(raw, &current) != nil {
			return "", false
		}
	}
	return "", false
}

func signatureMatches(root map[string]json.RawMessage, required map[string]ValueType) bool {
	for key, wanted := range required {
		raw, ok := root[key]
		if !ok || jsonValueType(raw) != wanted {
			return false
		}
	}
	return true
}

func jsonValueType(raw json.RawMessage) ValueType {
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return ""
	}
	switch value.(type) {
	case string:
		return ValueString
	case float64:
		return ValueNumber
	case bool:
		return ValueBoolean
	case []any:
		return ValueArray
	case map[string]any:
		return ValueObject
	default:
		return ""
	}
}

type SupportGrade string

const (
	SupportFull    SupportGrade = "full"
	SupportPartial SupportGrade = "partial"
	SupportNone    SupportGrade = "none"
)

type ContentCondition struct {
	Description string
	Matches     func(block.Content) bool
}

type RoleSupport struct {
	Grade       SupportGrade
	Condition   *ContentCondition
	DropWhen    *ContentCondition
	Destination string
	ShownBy     []string
}

type DirectionalRoleSupport struct {
	Read  RoleSupport
	Write RoleSupport
}

type SlotDeclaration struct {
	Name        string
	Type        ValueType
	Constraints []string
}

// MaxArchiveFiles is how many files an archive a module reads may hold, unless the module allows more.
const MaxArchiveFiles = 512

type ContentLimits struct {
	PayloadBytes    int
	CollectionItems int
	ItemBytes       int
	ArchiveFiles    int
}

type Boilerplate struct {
	Namespace string
	Path      []string
	Unchosen  []string
}

type PreservationDeclaration struct {
	Body      string
	Container []string
}

func (d Declaration) RecordsNothing(namespace string, payload []byte) bool {
	for _, entry := range d.Boilerplate {
		if entry.Namespace != namespace {
			continue
		}
		value, ok := valueAtPath(payload, entry.Path)
		if !ok || blankJSON(value) {
			return true
		}
		return slices.Contains(entry.Unchosen, scalarText(value))
	}
	return false
}

func valueAtPath(payload []byte, path []string) (json.RawMessage, bool) {
	value := json.RawMessage(payload)
	for _, part := range path {
		var object map[string]json.RawMessage
		if json.Unmarshal(value, &object) != nil {
			return nil, false
		}
		next, ok := object[part]
		if !ok {
			return nil, false
		}
		value = next
	}
	return value, len(value) > 0
}

func blankJSON(value json.RawMessage) bool {
	var decoded any
	if json.Unmarshal(value, &decoded) != nil {
		return false
	}
	switch held := decoded.(type) {
	case nil:
		return true
	case string:
		return held == ""
	case bool:
		return !held
	case float64:
		return held == 0
	case []any:
		return len(held) == 0
	case map[string]any:
		return len(held) == 0
	default:
		return false
	}
}

func scalarText(value json.RawMessage) string {
	var text string
	if json.Unmarshal(value, &text) == nil {
		return text
	}
	return string(bytes.TrimSpace(value))
}

type Declaration struct {
	ID               string
	Label            string
	Type             string
	Types            []string
	Input            Input
	Columns          []ColumnDisposition
	Anomalies        []AnomalyDeclaration
	Direction        Direction
	Recognition      []Recognition
	Roles            map[block.Role]DirectionalRoleSupport
	Header           []HeaderField
	Slots            []SlotDeclaration
	Limits           ContentLimits
	ConsumedKeys     []string
	Boilerplate      []Boilerplate
	Preservation     PreservationDeclaration
	TestedOrigins    []string
	PreservesOrigins []string
	KeepsUpload      bool
	// KeepsPrivatePrompts says an app reading this format keeps a work's private prompts out of sight.
	KeepsPrivatePrompts bool
}

func ValidateDeclaration(d Declaration) error {
	checks := []func(Declaration) error{
		validateDeclarationShape,
		validateColumns,
		validateAnomalies,
		validateHeader,
		validateRecognition,
		validateRoleSupport,
		validateSlots,
		validateStorageContract,
	}
	for _, check := range checks {
		if err := check(d); err != nil {
			return err
		}
	}
	return nil
}

func validateDeclarationShape(d Declaration) error {
	if d.ID == "" {
		return errors.New("identity is required")
	}
	if d.Input == InputDatabaseRow {
		if d.Type != "" || len(d.Types) == 0 {
			return errors.New("a database reader needs its supported types")
		}
		if !d.Direction.Read || d.Direction.Write || len(d.Recognition) > 0 {
			return errors.New("a database reader reads rows and neither recognises nor writes files")
		}
		if len(d.Columns) == 0 {
			return errors.New("a database reader needs a disposition for every source column")
		}
		if len(d.Anomalies) == 0 {
			return errors.New("a database reader needs an ahead-of-run anomaly policy")
		}
	} else if d.Type == "" || len(d.Types) > 0 {
		return errors.New("a file module needs exactly one type")
	} else if len(d.Columns) > 0 || len(d.Anomalies) > 0 {
		return errors.New("only a database reader declares source columns and anomalies")
	}
	if d.Direction.Write && d.Label == "" {
		return errors.New("a writer needs a label for the download menu")
	}
	if !d.Direction.Read && !d.Direction.Write {
		return errors.New("at least one direction is required")
	}
	if d.KeepsUpload && (!d.Direction.Read || !d.Direction.Write || !slices.Equal(d.TestedOrigins, []string{d.ID})) {
		return errors.New("a module that keeps the upload reads and writes only its own files")
	}
	if d.Direction.Read && d.Input == InputFile && len(d.Recognition) == 0 {
		return errors.New("a reader needs declared recognition")
	}
	return nil
}

func validateColumns(d Declaration) error {
	return ValidateColumns(d.Columns)
}

func ValidateColumns(columns []ColumnDisposition) error {
	seenColumns := make(map[string]bool, len(columns))
	for _, column := range columns {
		key := column.Table + "." + column.Column
		if column.Table == "" || column.Column == "" {
			return errors.New("a source column needs its table and name")
		}
		if seenColumns[key] {
			return fmt.Errorf("source column %s is declared twice", key)
		}
		seenColumns[key] = true
		switch column.Disposition {
		case ColumnMapped, ColumnPreserved:
			if column.Destination == "" || column.Reason != "" {
				return fmt.Errorf("%s %s column needs only a destination", key, column.Disposition)
			}
		case ColumnDropped:
			if column.Reason == "" || column.Destination != "" {
				return fmt.Errorf("dropped %s column needs only a reason", key)
			}
		default:
			return fmt.Errorf("source column %s has disposition %q", key, column.Disposition)
		}
	}
	return nil
}

func validateAnomalies(d Declaration) error {
	return ValidateAnomalies(d.Anomalies)
}

func ValidateAnomalies(anomalies []AnomalyDeclaration) error {
	seenAnomalies := make(map[string]bool, len(anomalies))
	for _, anomaly := range anomalies {
		if anomaly.Type == "" || anomaly.Reason == "" {
			return errors.New("an anomaly needs a type and reason")
		}
		if seenAnomalies[anomaly.Type] {
			return fmt.Errorf("anomaly %q is declared twice", anomaly.Type)
		}
		seenAnomalies[anomaly.Type] = true
		if anomaly.Disposition != AnomalyTolerated && anomaly.Disposition != AnomalyFatal {
			return fmt.Errorf("anomaly %q has disposition %q", anomaly.Type, anomaly.Disposition)
		}
	}
	return nil
}

func validateHeader(d Declaration) error {
	for _, field := range d.Header {
		if !field.Known() {
			return fmt.Errorf("header field %q is not one a work carries", field)
		}
	}
	return nil
}

func validateRecognition(d Declaration) error {
	for _, recognition := range d.Recognition {
		if len(recognition.Containers) == 0 {
			return errors.New("recognition needs at least one container")
		}
		switch recognition.Type {
		case RecognitionDiscriminator:
			if len(recognition.Path) == 0 || len(recognition.Values) == 0 {
				return errors.New("a discriminator needs a location and accepted values")
			}
			for _, superseding := range recognition.SupersededBy {
				if slices.Contains(recognition.Values, superseding) {
					return fmt.Errorf("value %q both matches and supersedes the discriminator", superseding)
				}
			}
		case RecognitionSignature:
			if len(recognition.Required) == 0 {
				return errors.New("a structural signature needs required keys")
			}
			if len(recognition.SupersededBy) > 0 {
				return errors.New("only a discriminator can name what supersedes it")
			}
			for key, valueType := range recognition.Required {
				if key == "" || !valueType.known() {
					return fmt.Errorf("structural key %q has type %q", key, valueType)
				}
			}
		case RecognitionEntry:
			if recognition.Entry == "" || !slices.Equal(recognition.Containers, []Container{ZIP}) {
				return errors.New("an entry recognition needs an archive and the entry it reads")
			}
		default:
			return fmt.Errorf("unknown recognition type %q", recognition.Type)
		}
	}
	return nil
}

func validateRoleSupport(d Declaration) error {
	for role, directional := range d.Roles {
		if !role.Known() {
			return fmt.Errorf("unknown semantic role %q", role)
		}
		for direction, support := range map[string]RoleSupport{"read": directional.Read, "write": directional.Write} {
			if support.Grade != SupportFull && support.Grade != SupportPartial && support.Grade != SupportNone {
				return fmt.Errorf("%s %s support has grade %q", role, direction, support.Grade)
			}
			if support.Grade == SupportPartial &&
				(support.Condition == nil || support.Condition.Matches == nil || support.Condition.Description == "") {
				return fmt.Errorf("%s %s partial support needs a content condition", role, direction)
			}
			if support.DropWhen != nil && support.DropWhen.Matches == nil {
				return fmt.Errorf("%s %s drop condition needs a matcher", role, direction)
			}
		}
	}
	return nil
}

func validateSlots(d Declaration) error {
	seenSlots := make(map[string]bool)
	for _, slot := range d.Slots {
		if slot.Name == "" || !slot.Type.known() {
			return fmt.Errorf("slot %q has type %q", slot.Name, slot.Type)
		}
		if seenSlots[slot.Name] {
			return fmt.Errorf("slot %q is declared twice", slot.Name)
		}
		seenSlots[slot.Name] = true
	}
	return nil
}

func validateStorageContract(d Declaration) error {
	if d.Limits.PayloadBytes <= 0 || d.Limits.CollectionItems <= 0 || d.Limits.ItemBytes <= 0 {
		return errors.New("payload, collection and item limits are required")
	}
	readsArchives := slices.ContainsFunc(d.Recognition, func(recognition Recognition) bool {
		return slices.Contains(recognition.Containers, ZIP)
	})
	if readsArchives && d.Limits.ArchiveFiles <= 0 {
		return errors.New("a module that reads archives needs a limit on their files")
	}
	if d.Input == InputFile && len(d.ConsumedKeys) == 0 {
		return errors.New("consumed keys are required")
	}
	if d.Preservation.Body == "" {
		return errors.New("a namespace for the format's own leftover keys is required")
	}
	if len(d.TestedOrigins) == 0 {
		return errors.New("tested origins are required")
	}
	for _, origin := range d.PreservesOrigins {
		if !slices.Contains(d.TestedOrigins, origin) {
			return fmt.Errorf("preserved origin %q has not been tested", origin)
		}
	}
	return validateDestinations(d)
}

func validateDestinations(d Declaration) error {
	known := make([]string, 0, len(Apps()))
	for _, app := range Apps() {
		known = append(known, app.ID)
	}
	for role, support := range d.Roles {
		write := support.Write
		if len(write.ShownBy) == 0 {
			continue
		}
		if write.Destination == "" {
			return fmt.Errorf("%s names the apps that show it but no destination", role)
		}
		for _, app := range write.ShownBy {
			if !slices.Contains(known, app) {
				return fmt.Errorf("%s names %q, which is no application Illarin knows", role, app)
			}
		}
	}
	return nil
}

func (t ValueType) known() bool {
	return t == ValueString || t == ValueNumber || t == ValueBoolean ||
		t == ValueObject || t == ValueArray
}

// Raw names the uploaded file itself as a download or send format
const Raw = "raw"
