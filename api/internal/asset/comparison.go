package asset

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/protected"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrNoEarlierVersion = errors.New("nothing was recorded before that version")
	ErrAccessRequired   = errors.New("a comparison needs the reader's access rules")
)

type Version struct {
	ID           uuid.UUID
	Number       int
	RecordedAt   time.Time
	Initial      bool
	VersionLabel string
	Summary      string
	Notes        string
}

type ChangeKind string

const (
	ChangeAdded   ChangeKind = "addition"
	ChangeRemoved ChangeKind = "removal"
	ChangeEdited  ChangeKind = "change"
)

type Change struct {
	Kind         ChangeKind
	Name         string
	Note         string
	PreviousName string
	Before       string
	After        string
	BeforeMedia  *uuid.UUID
	AfterMedia   *uuid.UUID
	BeforeImage  string
	AfterImage   string
}

type ChangeGroup struct {
	Subject string
	Label   string
	Changes []Change
}

type Comparison struct {
	From            Version
	To              Version
	Groups          []ChangeGroup
	Unavailable     string
	PromptsWithheld bool
}

type VersionAccess func(Version) string

type ComparisonRequest struct {
	AssetID    uuid.UUID
	From       int
	To         int
	Access     VersionAccess
	AsOwner    bool
	Visibility ContentVisibility
}

const (
	metadataSubject     = "metadata"
	presentationSubject = "presentation"
	preservedSubject    = "preserved_data"
	picturesSubject     = "pictures"
)

func (s *Service) Compare(ctx context.Context, in ComparisonRequest) (Comparison, error) {
	if in.Access == nil {
		return Comparison{}, ErrAccessRequired
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return Comparison{}, err
	}
	defer tx.Rollback(ctx)
	from, to, err := resolveVersions(ctx, tx, in.AssetID, in.From, in.To)
	if err != nil {
		return Comparison{}, err
	}
	earlier, err := readVersion(ctx, tx, in.AssetID, from)
	if err != nil {
		return Comparison{}, err
	}
	later, err := readVersion(ctx, tx, in.AssetID, to)
	if err != nil {
		return Comparison{}, err
	}
	compared := Comparison{From: earlier.Version, To: later.Version}
	for _, version := range []recordedVersion{earlier, later} {
		if refusal := in.Access(version.Version); refusal != "" {
			compared.Unavailable = refusal
			return compared, nil
		}
	}
	for _, version := range []recordedVersion{earlier, later} {
		withheld, err := version.holdPrompts(ctx, tx, in.AssetID, in.AsOwner)
		if err != nil {
			return Comparison{}, err
		}
		compared.PromptsWithheld = compared.PromptsWithheld || withheld
	}
	compared.Groups = compareVersions(earlier, later)
	if err := s.addressPictures(ctx, tx, in, compared.Groups); err != nil {
		return Comparison{}, err
	}
	return compared, nil
}

func (s *Service) addressPictures(
	ctx context.Context,
	tx pgx.Tx,
	in ComparisonRequest,
	groups []ChangeGroup,
) error {
	var flagged *bool
	if err := tx.QueryRow(ctx,
		`select is_nsfw from assets where id = $1`, in.AssetID).Scan(&flagged); err != nil {
		return fmt.Errorf("read the asset to address its pictures: %w", err)
	}
	blurred := flagged != nil && *flagged && in.Visibility != ContentShown
	for _, group := range groups {
		for index, change := range group.Changes {
			if change.BeforeMedia != nil {
				group.Changes[index].BeforeImage = s.variantURL(*change.BeforeMedia, "thumb", blurred, false)
			}
			if change.AfterMedia != nil {
				group.Changes[index].AfterImage = s.variantURL(*change.AfterMedia, "thumb", blurred, false)
			}
		}
	}
	return nil
}

func (v recordedVersion) holdPrompts(
	ctx context.Context,
	tx pgx.Tx,
	assetID uuid.UUID,
	asOwner bool,
) (bool, error) {
	if err := protected.RestoreRecordedPrompts(v.protectedPayloads, v.blocks); err != nil {
		return false, err
	}
	if asOwner {
		return false, nil
	}
	return protected.ApplyRecordedPolicy(ctx, tx, assetID, &v.ID, v.blocks)
}

func resolveVersions(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, from, to int) (int, int, error) {
	if to == 0 {
		err := tx.QueryRow(ctx, `
			select s.number from asset_snapshots s
			  join assets a on a.published_snapshot_id = s.id and a.id = s.asset_id
			 where a.id = $1
		`, assetID).Scan(&to)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrNotFound
		}
		if err != nil {
			return 0, 0, fmt.Errorf("read the published version: %w", err)
		}
	}
	if from == 0 {
		var earlier *int
		err := tx.QueryRow(ctx, `
			select max(number) from asset_snapshots where asset_id = $1 and number < $2
		`, assetID, to).Scan(&earlier)
		if err != nil {
			return 0, 0, fmt.Errorf("read the version before %d: %w", to, err)
		}
		if earlier == nil {
			return 0, 0, ErrNoEarlierVersion
		}
		from = *earlier
	}
	return from, to, nil
}

type recordedVersion struct {
	Version
	kind              string
	origin            string
	sourceRevisionID  *uuid.UUID
	metadata          versionMetadata
	blocks            []block.Block
	preserved         []versionPreserved
	protectedPayloads []byte
}

type versionMetadata struct {
	Name           string     `json:"name"`
	Blurb          string     `json:"blurb"`
	Tags           []string   `json:"tags"`
	IsNSFW         *bool      `json:"is_nsfw"`
	CreditedAuthor string     `json:"credited_author"`
	Nickname       string     `json:"nickname"`
	AssetVersion   string     `json:"asset_version"`
	Cover          *uuid.UUID `json:"cover_media_id"`
}

type versionPreserved struct {
	Owner     string    `json:"owner_kind"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Namespace string    `json:"namespace"`
	Payload   string    `json:"payload"`
}

type versionPayload struct {
	versionMetadata
	Kind      string             `json:"kind"`
	Origin    string             `json:"origin_format"`
	Blocks    []block.Block      `json:"blocks"`
	Preserved []versionPreserved `json:"preserved_data"`
}

func readVersion(ctx context.Context, tx pgx.Tx, assetID uuid.UUID, number int) (recordedVersion, error) {
	var recorded recordedVersion
	var stored []byte
	var sourceRevision pgtype.UUID
	err := tx.QueryRow(ctx, `
		select id, number, recorded_at, initial_recorded, version_label, summary, notes,
		       source_revision_id, payload, protected_payloads
		  from public.asset_snapshots where asset_id = $1 and number = $2
	`, assetID, number).Scan(&recorded.ID, &recorded.Number, &recorded.RecordedAt,
		&recorded.Initial, &recorded.VersionLabel, &recorded.Summary, &recorded.Notes,
		&sourceRevision, &stored, &recorded.protectedPayloads)
	if errors.Is(err, pgx.ErrNoRows) {
		return recordedVersion{}, ErrNotFound
	}
	if err != nil {
		return recordedVersion{}, fmt.Errorf("read version %d: %w", number, err)
	}
	var payload versionPayload
	if err := json.Unmarshal(stored, &payload); err != nil {
		return recordedVersion{}, fmt.Errorf("read version %d: %w", number, err)
	}
	recorded.kind = payload.Kind
	recorded.origin = payload.Origin
	recorded.sourceRevisionID = uuidOrNil(sourceRevision)
	recorded.metadata = payload.versionMetadata
	recorded.blocks = payload.Blocks
	recorded.preserved = payload.Preserved
	return recorded, nil
}

func compareVersions(earlier, later recordedVersion) []ChangeGroup {
	groups := make([]ChangeGroup, 0, 8)
	groups = addGroup(groups, metadataSubject, "Details", compareMetadata(earlier.metadata, later.metadata))
	groups = append(groups, compareContent(earlier.blocks, later.blocks)...)
	groups = addGroup(groups, presentationSubject, "Page",
		comparePresentation(later.kind, earlier.blocks, later.blocks))
	groups = addGroup(groups, preservedSubject, "Preserved data",
		comparePreserved(earlier.preserved, later.preserved))
	return groups
}

func addGroup(groups []ChangeGroup, subject, label string, changes []Change) []ChangeGroup {
	if len(changes) == 0 {
		return groups
	}
	return append(groups, ChangeGroup{Subject: subject, Label: label, Changes: changes})
}

func compareMetadata(earlier, later versionMetadata) []Change {
	changes := make([]Change, 0, 8)
	for _, field := range []struct{ name, before, after string }{
		{"Name", earlier.Name, later.Name},
		{"Blurb", earlier.Blurb, later.Blurb},
		{"Tags", strings.Join(earlier.Tags, ", "), strings.Join(later.Tags, ", ")},
		{"Adult content", adultAnswer(earlier.IsNSFW), adultAnswer(later.IsNSFW)},
		{"Credited author", earlier.CreditedAuthor, later.CreditedAuthor},
		{"Nickname", earlier.Nickname, later.Nickname},
		{"Version", earlier.AssetVersion, later.AssetVersion},
	} {
		if change, changed := textChange(field.name, field.before, field.after); changed {
			changes = append(changes, change)
		}
	}
	if !sameMedia(earlier.Cover, later.Cover) {
		changes = append(changes, Change{
			Kind: mediaChangeKind(earlier.Cover, later.Cover), Name: "Cover picture",
			BeforeMedia: earlier.Cover, AfterMedia: later.Cover,
		})
	}
	return changes
}

func adultAnswer(answered *bool) string {
	if answered == nil {
		return ""
	}
	return strconv.FormatBool(*answered)
}

func sameMedia(earlier, later *uuid.UUID) bool {
	if earlier == nil || later == nil {
		return earlier == later
	}
	return *earlier == *later
}

func mediaChangeKind(earlier, later *uuid.UUID) ChangeKind {
	switch {
	case earlier == nil:
		return ChangeAdded
	case later == nil:
		return ChangeRemoved
	default:
		return ChangeEdited
	}
}

func textChange(name, before, after string) (Change, bool) {
	switch {
	case before == after:
		return Change{}, false
	case before == "":
		return Change{Kind: ChangeAdded, Name: name, After: after}, true
	case after == "":
		return Change{Kind: ChangeRemoved, Name: name, Before: before}, true
	default:
		return Change{Kind: ChangeEdited, Name: name, Before: before, After: after}, true
	}
}

type subjectItems struct {
	label string
	items []versionItem
}

type versionItem struct {
	key   string
	name  string
	note  string
	text  string
	media uuid.UUID
	body  string
}

func (i versionItem) mediaRef() *uuid.UUID {
	if i.media == uuid.Nil {
		return nil
	}
	return &i.media
}

func compareContent(earlier, later []block.Block) []ChangeGroup {
	return compareContentKeyed(earlier, later, nil)
}

// compareContentKeyed compares content, matching items whose ids changed by a stable name.
func compareContentKeyed(earlier, later []block.Block, names map[uuid.UUID]string) []ChangeGroup {
	before := contentSubjects(earlier, names)
	after := contentSubjects(later, names)
	groups := make([]ChangeGroup, 0, len(before)+len(after))
	for _, subject := range subjectOrder(before, after) {
		label := after[subject].label
		if label == "" {
			label = before[subject].label
		}
		groups = addGroup(groups, subject, label, compareItems(before[subject].items, after[subject].items))
	}
	return groups
}

func contentSubjects(blocks []block.Block, names map[uuid.UUID]string) map[string]subjectItems {
	subjects := make(map[string]subjectItems)
	for _, holder := range blocks {
		for _, element := range holder.Elements {
			if element.Content == nil {
				continue
			}
			subject := string(element.Type)
			if element.Role != "" {
				subject = string(element.Role)
			}
			held := subjects[subject]
			held.label = element.Label()
			held.items = append(held.items, elementItems(element, names)...)
			subjects[subject] = held
		}
	}
	return subjects
}

func subjectOrder(before, after map[string]subjectItems) []string {
	present := make(map[string]bool, len(before)+len(after))
	for subject := range before {
		present[subject] = true
	}
	for subject := range after {
		present[subject] = true
	}
	ordered := make([]string, 0, len(present))
	for _, role := range block.Roles() {
		if present[string(role)] {
			ordered = append(ordered, string(role))
			delete(present, string(role))
		}
	}
	rest := make([]string, 0, len(present))
	for subject := range present {
		rest = append(rest, subject)
	}
	sort.Strings(rest)
	return append(ordered, rest...)
}

func elementItems(element block.Element, names map[uuid.UUID]string) []versionItem {
	itemKey := func(id uuid.UUID) string {
		if stable, named := names[id]; named {
			return stable
		}
		return id.String()
	}
	switch held := element.Content.(type) {
	case block.Prose:
		return []versionItem{{key: proseKey(element), text: held.Text, body: held.Text}}
	case block.TextSet:
		return listItems(names, held.Texts, func(text block.TextItem) versionItem {
			return versionItem{key: itemKey(text.ID), name: text.Name, text: text.Text}
		})
	case block.DialogueSample:
		return listItems(names, held.Turns, func(turn block.DialogueTurn) versionItem {
			return versionItem{key: itemKey(turn.ID), name: turn.Speaker, text: turn.Text}
		})
	case block.ImageSet:
		return listItems(names, held.Images, func(image block.ImageItem) versionItem {
			return versionItem{key: itemKey(image.ID), name: image.Name, media: image.MediaID}
		})
	case block.FieldList:
		return listItems(names, held.Fields, func(field block.FieldItem) versionItem {
			return versionItem{key: itemKey(field.ID), name: field.Name, text: field.Value}
		})
	case block.LinkList:
		return listItems(names, held.Links, func(link block.LinkItem) versionItem {
			return versionItem{key: itemKey(link.ID), name: link.Label, text: link.URL}
		})
	case block.EntryTable:
		return listItems(names, held.Entries, func(entry block.Entry) versionItem {
			return versionItem{key: itemKey(entry.ID), name: entry.Name, text: entry.Text}
		})
	case block.PromptList:
		return append(
			listItems(names, held.Groups, func(group block.PromptGroup) versionItem {
				return versionItem{key: itemKey(group.ID), name: group.Name}
			}),
			listItems(names, held.Fragments, func(fragment block.PromptFragment) versionItem {
				return versionItem{
					key: itemKey(fragment.ID), name: fragment.Name,
					note: sealingNote(fragment), text: fragment.Text,
				}
			})...)
	case block.VariableSchema:
		return listItems(names, held.Variables, func(variable block.Variable) versionItem {
			return versionItem{
				key: itemKey(variable.ID), name: preferredName(variable.Label, variable.Name),
				text: settingValue(variable.Value),
			}
		})
	case block.SettingGroup:
		return listItems(names, held.Settings, func(setting block.Setting) versionItem {
			return versionItem{
				key: itemKey(setting.ID), name: preferredName(setting.Label, setting.Name),
				text: settingValue(setting.Value),
			}
		})
	case block.ScriptList:
		return listItems(names, held.Scripts, func(script block.Script) versionItem {
			return versionItem{key: itemKey(script.ID), name: script.Name, text: script.Find}
		})
	case block.ColorSet:
		items := make([]versionItem, 0, len(held.Modes))
		for _, mode := range held.Modes {
			items = append(items, listItems(names, mode.Colors, func(color block.Color) versionItem {
				return versionItem{
					key: itemKey(color.ID), name: preferredName(mode.Name+" "+color.Name, color.Name),
					text: color.Value,
				}
			})...)
		}
		return items
	case block.StylesheetSet:
		items := listItems(names, held.Stylesheets, func(sheet block.Stylesheet) versionItem {
			return versionItem{key: itemKey(sheet.ID), name: sheet.Name, text: sheet.CSS}
		})
		items = append(items, listItems(names, held.Assets, func(file block.StylesheetAsset) versionItem {
			return versionItem{key: itemKey(file.ID), name: file.Path}
		})...)
		if held.Global != "" {
			items = append(items, versionItem{
				key:  element.ID.String() + " global",
				name: "Global stylesheet", text: held.Global, body: held.Global,
			})
		}
		return items
	case block.RecordList:
		return listItems(names, held.Records, func(record block.LumiaRecord) versionItem {
			return versionItem{
				key: itemKey(record.ID), name: record.LumiaName, text: record.LumiaDefinition,
			}
		})
	default:
		return nil
	}
}

// sealingNote says when a prompt's wording is kept back for linked apps.
func sealingNote(fragment block.PromptFragment) string {
	if fragment.Protected {
		return "sealed for linked apps"
	}
	return ""
}

func proseKey(element block.Element) string {
	if element.Role != "" {
		return string(element.Role)
	}
	return element.ID.String()
}

func listItems[T any](names map[uuid.UUID]string, list []T, describe func(T) versionItem) []versionItem {
	items := make([]versionItem, 0, len(list))
	for _, entry := range list {
		item := describe(entry)
		item.body = itemBody(entry, names)
		items = append(items, item)
	}
	return items
}

func itemBody(item any, names map[uuid.UUID]string) string {
	encoded, err := json.Marshal(item)
	if err != nil {
		return ""
	}
	var fields map[string]any
	if json.Unmarshal(encoded, &fields) != nil {
		return string(encoded)
	}
	delete(fields, "id")
	canonical, err := json.Marshal(steadyIDs(fields, names))
	if err != nil {
		return string(encoded)
	}
	return string(canonical)
}

// steadyIDs swaps ids a file mints afresh for the stable names they stand for.
func steadyIDs(value any, names map[uuid.UUID]string) any {
	if len(names) == 0 {
		return value
	}
	switch held := value.(type) {
	case map[string]any:
		for key, nested := range held {
			held[key] = steadyIDs(nested, names)
		}
		return held
	case []any:
		for index, nested := range held {
			held[index] = steadyIDs(nested, names)
		}
		return held
	case string:
		id, err := uuid.Parse(held)
		if err != nil {
			return held
		}
		if stable, named := names[id]; named {
			return stable
		}
		return held
	default:
		return value
	}
}

func preferredName(preferred, fallback string) string {
	if trimmed := strings.TrimSpace(preferred); trimmed != "" {
		return trimmed
	}
	return fallback
}

func settingValue(value *block.Value) string {
	switch {
	case value == nil:
		return ""
	case value.Text != nil:
		return *value.Text
	case value.Number != nil:
		return strconv.FormatFloat(*value.Number, 'g', -1, 64)
	case value.Boolean != nil:
		return strconv.FormatBool(*value.Boolean)
	default:
		return strings.Join(value.Strings, ", ")
	}
}

func compareItems(earlier, later []versionItem) []Change {
	partner := make([]int, len(later))
	for index := range partner {
		partner[index] = -1
	}
	taken := make([]bool, len(earlier))
	matchItems(earlier, later, taken, partner, func(item versionItem) string { return item.key })
	matchItems(earlier, later, taken, partner, func(item versionItem) string { return item.body })
	matchItems(earlier, later, taken, partner, func(item versionItem) string { return item.name })
	changes := make([]Change, 0, len(later))
	for index, item := range later {
		if partner[index] < 0 {
			changes = append(changes, Change{
				Kind: ChangeAdded, Name: item.name, Note: item.note,
				After: item.text, AfterMedia: item.mediaRef(),
			})
			continue
		}
		was := earlier[partner[index]]
		if was.body == item.body {
			continue
		}
		changes = append(changes, editedItem(was, item))
	}
	for index, item := range earlier {
		if !taken[index] {
			changes = append(changes, Change{
				Kind: ChangeRemoved, Name: item.name, Note: item.note,
				Before: item.text, BeforeMedia: item.mediaRef(),
			})
		}
	}
	return changes
}

func matchItems(earlier, later []versionItem, taken []bool, partner []int, key func(versionItem) string) {
	available := make(map[string][]int, len(earlier))
	for index, item := range earlier {
		if !taken[index] && key(item) != "" {
			available[key(item)] = append(available[key(item)], index)
		}
	}
	for index, item := range later {
		if partner[index] >= 0 || key(item) == "" {
			continue
		}
		waiting := available[key(item)]
		if len(waiting) == 0 {
			continue
		}
		partner[index] = waiting[0]
		taken[waiting[0]] = true
		available[key(item)] = waiting[1:]
	}
}

func editedItem(was, now versionItem) Change {
	if was.name == "" && now.name == "" && was.media == uuid.Nil && now.media == uuid.Nil {
		if change, changed := textChange("", was.text, now.text); changed {
			return change
		}
	}
	edited := Change{Kind: ChangeEdited, Name: now.name}
	if was.note != now.note {
		edited.Note = now.note
		if edited.Note == "" {
			edited.Note = "no longer sealed"
		}
	}
	if was.name != now.name {
		edited.PreviousName = was.name
	}
	if was.text != now.text {
		edited.Before, edited.After = was.text, now.text
	}
	if was.media != now.media {
		edited.BeforeMedia, edited.AfterMedia = was.mediaRef(), now.mediaRef()
	}
	return edited
}

func comparePresentation(kind string, earlier, later []block.Block) []Change {
	before := blocksByID(earlier)
	after := blocksByID(later)
	changes := make([]Change, 0, len(later))
	for _, holder := range later {
		was, kept := before[holder.ID]
		if !kept {
			changes = append(changes, Change{Kind: ChangeAdded, Name: blockName(kind, holder)})
			continue
		}
		changes = append(changes, blockChanges(kind, was, holder)...)
	}
	for _, holder := range earlier {
		if _, kept := after[holder.ID]; !kept {
			changes = append(changes, Change{Kind: ChangeRemoved, Name: blockName(kind, holder)})
		}
	}
	if !slices.Equal(sharedOrder(earlier, after), sharedOrder(later, before)) {
		changes = append(changes, Change{Kind: ChangeEdited, Name: "Page order"})
	}
	return changes
}

func blocksByID(blocks []block.Block) map[uuid.UUID]block.Block {
	held := make(map[uuid.UUID]block.Block, len(blocks))
	for _, holder := range blocks {
		held[holder.ID] = holder
	}
	return held
}

func sharedOrder(blocks []block.Block, other map[uuid.UUID]block.Block) []uuid.UUID {
	shared := make([]uuid.UUID, 0, len(blocks))
	for _, holder := range blocks {
		if _, kept := other[holder.ID]; kept {
			shared = append(shared, holder.ID)
		}
	}
	return shared
}

func blockChanges(kind string, was, now block.Block) []Change {
	name := blockName(kind, now)
	changes := make([]Change, 0, 5)
	for _, facet := range []struct{ label, before, after string }{
		{"title", blockTitle(was), blockTitle(now)},
		{"layout", string(was.Layout), string(now.Layout)},
		{"width", string(was.Width), string(now.Width)},
		{"visibility", shownWord(was.Hidden), shownWord(now.Hidden)},
	} {
		if change, changed := textChange(name+" "+facet.label, facet.before, facet.after); changed {
			changes = append(changes, change)
		}
	}
	priorElements := make(map[uuid.UUID]block.Element, len(was.Elements))
	for _, element := range was.Elements {
		priorElements[element.ID] = element
	}
	for _, element := range now.Elements {
		prior, kept := priorElements[element.ID]
		if !kept || prior.Options == element.Options {
			continue
		}
		changes = append(changes, Change{
			Kind: ChangeEdited, Name: element.Label() + " display",
			Before: optionWords(prior.Options), After: optionWords(element.Options),
		})
	}
	return changes
}

func blockTitle(holder block.Block) string {
	if holder.Title == nil {
		return ""
	}
	return *holder.Title
}

func blockName(kind string, holder block.Block) string {
	if title := blockTitle(holder); title != "" {
		return title
	}
	definitions, known := block.Catalog(kind)
	if !known {
		return string(holder.Definition)
	}
	for _, definition := range definitions {
		if definition.ID == holder.Definition {
			return definition.Title
		}
	}
	return string(holder.Definition)
}

func shownWord(hidden bool) string {
	if hidden {
		return "hidden"
	}
	return "shown"
}

func optionWords(options block.Options) string {
	words := make([]string, 0, 2)
	if options.Display != "" {
		words = append(words, string(options.Display))
	}
	if options.ItemSize != "" {
		words = append(words, string(options.ItemSize))
	}
	return strings.Join(words, " ")
}

func comparePreserved(earlier, later []versionPreserved) []Change {
	before := preservedDigests(earlier)
	after := preservedDigests(later)
	namespaces := make([]string, 0, len(before)+len(after))
	for namespace := range before {
		namespaces = append(namespaces, namespace)
	}
	for namespace := range after {
		if _, shared := before[namespace]; !shared {
			namespaces = append(namespaces, namespace)
		}
	}
	sort.Strings(namespaces)
	changes := make([]Change, 0, len(namespaces))
	for _, namespace := range namespaces {
		was, held := before[namespace]
		now, holds := after[namespace]
		switch {
		case !held:
			changes = append(changes, Change{Kind: ChangeAdded, Name: namespace})
		case !holds:
			changes = append(changes, Change{Kind: ChangeRemoved, Name: namespace})
		case was != now:
			changes = append(changes, Change{Kind: ChangeEdited, Name: namespace})
		}
	}
	return changes
}

func preservedDigests(preserved []versionPreserved) map[string]string {
	keyed := make(map[string][]string)
	for _, item := range preserved {
		keyed[item.Namespace] = append(keyed[item.Namespace],
			item.Owner+"\x00"+item.OwnerID.String()+"\x00"+item.Payload)
	}
	digests := make(map[string]string, len(keyed))
	for namespace, entries := range keyed {
		sort.Strings(entries)
		sum := sha256.Sum256([]byte(strings.Join(entries, "\x00")))
		digests[namespace] = hex.EncodeToString(sum[:])
	}
	return digests
}
