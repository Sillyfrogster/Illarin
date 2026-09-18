package character

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"path"
	"slices"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/book"
	"github.com/Sillyfrogster/Illarin/api/internal/format/keys"
)

const (
	v2SpecVersion  = "2.0"
	v3SpecVersion  = "3.0"
	dialogueStart  = "<START>"
	defaultFileURI = "ccdefault:"
)

var v3OnlyKeys = []string{
	"assets", "nickname", "group_only_greetings", "creation_date",
	"modification_date", "source", "creator_notes_multilingual",
}

func (CCv2Module) Write(_ context.Context, work format.ExportWork) (format.Artifact, error) {
	return writeCard(work, V2)
}

func (CCv3Module) Write(_ context.Context, work format.ExportWork) (format.Artifact, error) {
	return writeCard(work, V3)
}

func (CharXModule) Write(_ context.Context, work format.ExportWork) (format.Artifact, error) {
	return writeCharX(work)
}

func writeCard(work format.ExportWork, formatID string) (format.Artifact, error) {
	picture := embeddablePicture(work)
	body, entries := cardFields(work, formatID)
	if formatID != V2 {
		body["assets"] = inlineFiles(work, picture != nil)
	}
	if err := RestorePreserved(body, entries, work.Preserved); err != nil {
		return format.Artifact{}, err
	}
	if formatID == V2 {
		for _, key := range v3OnlyKeys {
			delete(body, key)
		}
	}
	card, err := marshalCard(formatID, body)
	if err != nil {
		return format.Artifact{}, err
	}
	copies := []cardCopy{{keyword: chunkName(formatID), card: card}}
	if formatID != V2 {
		fallback, err := marshalCard(V2, olderShape(body))
		if err != nil {
			return format.Artifact{}, err
		}
		copies = append(copies, cardCopy{keyword: chunkName(V2), card: fallback})
	}
	if picture == nil {
		if formatID != V2 {
			card, err = withLegacyFields(card, body)
			if err != nil {
				return format.Artifact{}, err
			}
		}
		return format.Artifact{Body: card, MediaType: "application/json", Extension: ".json"}, nil
	}
	written, err := embedCardsInPNG(picture.Data, copies)
	if err != nil {
		return format.Artifact{}, err
	}
	return format.Artifact{Body: written, MediaType: "image/png", Extension: ".png"}, nil
}

func olderShape(body map[string]json.RawMessage) map[string]json.RawMessage {
	older := make(map[string]json.RawMessage, len(body))
	for key, value := range body {
		if !slices.Contains(v3OnlyKeys, key) {
			older[key] = value
		}
	}
	return older
}

var legacyFields = []string{
	"name", "description", "personality", "scenario", "first_mes", "mes_example",
}

func withLegacyFields(card []byte, body map[string]json.RawMessage) ([]byte, error) {
	var document map[string]json.RawMessage
	if err := json.Unmarshal(card, &document); err != nil {
		return nil, fmt.Errorf("read the written card: %w", err)
	}
	for _, field := range legacyFields {
		if value, written := body[field]; written {
			document[field] = value
		}
	}
	written, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("write the card: %w", err)
	}
	return written, nil
}

func writeCharX(work format.ExportWork) (format.Artifact, error) {
	body, entries := cardFields(work, CharX)
	files, records := archivedFiles(work)
	body["assets"] = records
	if err := RestorePreserved(body, entries, work.Preserved); err != nil {
		return format.Artifact{}, err
	}
	card, err := marshalCard(V3, body)
	if err != nil {
		return format.Artifact{}, err
	}

	var output bytes.Buffer
	archive := zip.NewWriter(&output)
	if err := writeArchiveFile(archive, cardEntry, card); err != nil {
		return format.Artifact{}, err
	}
	for _, file := range slices.Concat(files, archivedMemberFiles(work.Preserved, files)) {
		if err := writeArchiveFile(archive, file.path, file.data); err != nil {
			return format.Artifact{}, err
		}
	}
	if err := archive.Close(); err != nil {
		return format.Artifact{}, fmt.Errorf("close CharX: %w", err)
	}
	return format.Artifact{
		Body: output.Bytes(), MediaType: "application/zip", Extension: ".charx",
	}, nil
}

// archivedMemberFiles puts back the archived files this card came in with.
func archivedMemberFiles(preserved []format.Remainder, written []archivedFile) []archivedFile {
	taken := map[string]bool{cardEntry: true}
	for _, file := range written {
		taken[file.path] = true
	}
	given := make([]archivedFile, 0, len(preserved))
	for _, row := range preserved {
		name, archived := ArchivedMemberName(row.Namespace)
		if !archived || taken[name] {
			continue
		}
		data, readable := ArchivedMember(row.Payload)
		if !readable {
			continue
		}
		taken[name] = true
		given = append(given, archivedFile{path: name, data: data})
	}
	return given
}

func writeArchiveFile(archive *zip.Writer, name string, data []byte) error {
	destination, err := archive.Create(name)
	if err != nil {
		return fmt.Errorf("create CharX entry %q: %w", name, err)
	}
	if _, err := destination.Write(data); err != nil {
		return fmt.Errorf("write CharX entry %q: %w", name, err)
	}
	return nil
}

func marshalCard(formatID string, body map[string]json.RawMessage) ([]byte, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("write the card body: %w", err)
	}
	version := v3SpecVersion
	if formatID == V2 {
		version = v2SpecVersion
	}
	card, err := json.Marshal(map[string]json.RawMessage{
		"spec":         keys.Must(formatID),
		"spec_version": keys.Must(version),
		"data":         data,
	})
	if err != nil {
		return nil, fmt.Errorf("write the card: %w", err)
	}
	return card, nil
}

func chunkName(formatID string) string {
	if formatID == V2 {
		return "chara"
	}
	return "ccv3"
}

func cardFields(
	work format.ExportWork,
	formatID string,
) (map[string]json.RawMessage, []block.Entry) {
	greetings := textItems(work, block.RoleGreetings)
	first := ""
	if len(greetings) > 0 {
		first = greetings[0].Text
	}
	body := map[string]json.RawMessage{
		"name":                      keys.Must(work.Header.Name),
		"description":               keys.Must(work.Text(block.RoleDescription)),
		"personality":               keys.Must(work.Text(block.RolePersonality)),
		"scenario":                  keys.Must(work.Text(block.RoleScenario)),
		"first_mes":                 keys.Must(first),
		"alternate_greetings":       keys.Must(textsOf(greetings[min(1, len(greetings)):])),
		"mes_example":               keys.Must(dialogueText(work)),
		"system_prompt":             keys.Must(work.Text(block.RoleSystemPrompt)),
		"post_history_instructions": keys.Must(work.Text(block.RolePostHistoryInstructions)),
		"creator_notes":             keys.Must(work.Text(block.RoleCreatorNotes)),
		"creator":                   keys.Must(work.Header.CreditedAuthor),
		"character_version":         keys.Must(work.Header.WorkVersion),
	}
	if formatID != V2 {
		body["nickname"] = keys.Must(work.Header.Nickname)
		body["group_only_greetings"] = keys.Must(
			textsOf(textItems(work, block.RoleGroupGreetings)),
		)
	}
	entries := bookEntries(work)
	if len(entries) > 0 {
		body[bookKey] = writtenBook(entries)
	}
	return body, entries
}

func textItems(work format.ExportWork, role block.Role) []block.TextItem {
	content, ok := work.Content(role)
	if !ok {
		return nil
	}
	set, isSet := content.(block.TextSet)
	if !isSet {
		return nil
	}
	return set.Texts
}

func textsOf(items []block.TextItem) []string {
	texts := make([]string, 0, len(items))
	for _, item := range items {
		texts = append(texts, item.Text)
	}
	return texts
}

func dialogueText(work format.ExportWork) string {
	content, ok := work.Content(block.RoleExampleDialogue)
	if !ok {
		return ""
	}
	sample, isSample := content.(block.DialogueSample)
	if !isSample || len(sample.Turns) == 0 {
		return ""
	}
	lines := make([]string, 0, len(sample.Turns)+1)
	lines = append(lines, dialogueStart)
	for _, turn := range sample.Turns {
		if turn.Speaker == "" {
			lines = append(lines, turn.Text)
			continue
		}
		lines = append(lines, turn.Speaker+": "+turn.Text)
	}
	return strings.Join(lines, "\n")
}

func bookEntries(work format.ExportWork) []block.Entry {
	content, ok := work.Content(block.RoleLorebookEntries)
	if !ok {
		return nil
	}
	table, isTable := content.(block.EntryTable)
	if !isTable {
		return nil
	}
	return table.Entries
}

func writtenBook(entries []block.Entry) json.RawMessage {
	return keys.Must(map[string]any{"entries": book.Write(entries)})
}

type cardFileRecord struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
	Name string `json:"name"`
	Ext  string `json:"ext"`
}

type archivedFile struct {
	path string
	data []byte
}

func inlineFiles(work format.ExportWork, embedded bool) json.RawMessage {
	records := make([]cardFileRecord, 0)
	for _, picture := range exportedPictures(work) {
		uri := dataURI(picture.media.MediaType, picture.media.Data)
		if picture.workType == iconFileType && embedded {
			uri = defaultFileURI
		}
		records = append(records, cardFileRecord{
			Type: picture.workType, URI: uri, Name: picture.name,
			Ext: mediaExtension(picture.media.MediaType),
		})
	}
	return keys.Must(records)
}

func archivedFiles(work format.ExportWork) ([]archivedFile, json.RawMessage) {
	files := make([]archivedFile, 0)
	records := make([]cardFileRecord, 0)
	taken := make(map[string]bool)
	for index, picture := range exportedPictures(work) {
		extension := mediaExtension(picture.media.MediaType)
		entry := path.Join(
			archiveFolder(picture.workType), fmt.Sprintf("%d.%s", index+1, extension),
		)
		if picture.workType == iconFileType && index == 0 {
			entry = path.Join(archiveFolder(iconFileType), "main."+extension)
		}
		if taken[entry] {
			continue
		}
		taken[entry] = true
		files = append(files, archivedFile{path: entry, data: picture.media.Data})
		records = append(records, cardFileRecord{
			Type: picture.workType, URI: embeddedPrefix + entry,
			Name: picture.name, Ext: extension,
		})
	}
	return files, keys.Must(records)
}

const (
	iconFileType        = "icon"
	emotionFileType     = "emotion"
	galleryFileType     = "x_gallery"
	mainIconFileName    = "main"
	fallbackPictureName = "image"
)

// archiveFolder puts each picture where the apps that read a CharX look for it.
func archiveFolder(workType string) string {
	switch workType {
	case iconFileType:
		return "assets/icon/image"
	case emotionFileType:
		return "assets/emotion/image"
	default:
		return "assets/other/image"
	}
}

type exportedPicture struct {
	workType string
	name     string
	media    format.ExportMedia
}

func exportedPictures(work format.ExportWork) []exportedPicture {
	pictures := make([]exportedPicture, 0)
	if work.Cover != nil {
		pictures = append(pictures, exportedPicture{
			workType: iconFileType, name: mainIconFileName, media: *work.Cover,
		})
	}
	for _, role := range []struct {
		role     block.Role
		workType string
	}{
		{block.RoleExpressions, emotionFileType},
		{block.RoleGallery, galleryFileType},
	} {
		for _, element := range work.Elements {
			if element.Role != role.role {
				continue
			}
			set, isSet := element.Content.(block.ImageSet)
			if !isSet {
				continue
			}
			for index, image := range set.Images {
				found, held := work.Images[image.MediaID]
				if !held {
					continue
				}
				pictures = append(pictures, exportedPicture{
					workType: role.workType,
					name:     pictureName(image.Name, index),
					media:    found,
				})
			}
		}
	}
	return pictures
}

func pictureName(name string, index int) string {
	if name != "" {
		return name
	}
	return fmt.Sprintf("%s-%d", fallbackPictureName, index+1)
}

func embeddablePicture(work format.ExportWork) *format.ExportMedia {
	if work.Cover == nil || !bytes.HasPrefix(work.Cover.Data, pngSignature) {
		return nil
	}
	return work.Cover
}

func dataURI(mediaType string, data []byte) string {
	if mediaType == "" {
		mediaType = "application/octet-stream"
	}
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func mediaExtension(mediaType string) string {
	switch mediaType {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	default:
		return "bin"
	}
}

var pngSignature = []byte("\x89PNG\r\n\x1a\n")

type cardCopy struct {
	keyword string
	card    []byte
}

func embedCardsInPNG(source []byte, copies []cardCopy) ([]byte, error) {
	if !bytes.HasPrefix(source, pngSignature) {
		return nil, errors.New("the work's picture is not a PNG")
	}
	var output bytes.Buffer
	output.Write(source[:8])
	inserted := false
	err := visitPNGChunks(source, func(chunkType string, data, raw []byte) error {
		if chunkType == "IEND" && !inserted {
			for _, copied := range copies {
				encoded := base64.StdEncoding.EncodeToString(copied.card)
				output.Write(makePNGChunk("tEXt", slices.Concat(
					[]byte(copied.keyword), []byte{0}, []byte(encoded),
				)))
			}
			inserted = true
		}
		if !isCardChunk(chunkType, data) {
			output.Write(raw)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !inserted {
		return nil, errors.New("the work's picture has no IEND chunk")
	}
	return output.Bytes(), nil
}

func isCardChunk(chunkType string, data []byte) bool {
	if chunkType != "tEXt" {
		return false
	}
	keyword, _, found := bytes.Cut(data, []byte{0})
	return found && (string(keyword) == "chara" || string(keyword) == "ccv3")
}

func makePNGChunk(chunkType string, data []byte) []byte {
	chunk := make([]byte, 12+len(data))
	binary.BigEndian.PutUint32(chunk[:4], uint32(len(data)))
	copy(chunk[4:8], chunkType)
	copy(chunk[8:], data)
	binary.BigEndian.PutUint32(chunk[8+len(data):], crc32.ChecksumIEEE(chunk[4:8+len(data)]))
	return chunk
}

func visitPNGChunks(source []byte, visit func(chunkType string, data, raw []byte) error) error {
	if !bytes.HasPrefix(source, pngSignature) {
		return errors.New("file is not a PNG")
	}
	for offset := 8; offset < len(source); {
		if offset+12 > len(source) {
			return fmt.Errorf("PNG chunk at byte %d is truncated", offset)
		}
		length := int(binary.BigEndian.Uint32(source[offset : offset+4]))
		end := offset + 12 + length
		if length < 0 || end < offset || end > len(source) {
			return fmt.Errorf("PNG chunk at byte %d exceeds the file", offset)
		}
		if err := visit(
			string(source[offset+4:offset+8]),
			source[offset+8:offset+8+length],
			source[offset:end],
		); err != nil {
			return err
		}
		offset = end
	}
	return nil
}
