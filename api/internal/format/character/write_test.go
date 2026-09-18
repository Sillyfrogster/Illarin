package character

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func TestEveryCharacterFormatWritesFromRolesAlone(t *testing.T) {
	t.Parallel()
	work := format.ExportWork{
		Type:   Type,
		Header: format.Header{Name: "Ana", WorkVersion: "1.2", CreditedAuthor: "Wren"},
		Elements: []block.Element{
			prose(block.RoleDescription, "Keeps the archive."),
			greetings("Hello", "You again."),
		},
	}
	for _, module := range Modules() {
		t.Run(module.ID(), func(t *testing.T) {
			artifact := write(t, module, work)
			body := writtenBody(t, artifact.Body, module.ID())
			if got := text(t, body["name"]); got != "Ana" {
				t.Errorf("name = %q, want Ana", got)
			}
			if got := text(t, body["description"]); got != "Keeps the archive." {
				t.Errorf("description = %q", got)
			}
			if got := text(t, body["first_mes"]); got != "Hello" {
				t.Errorf("first_mes = %q, want the first greeting", got)
			}
			var alternates []string
			_ = json.Unmarshal(body["alternate_greetings"], &alternates)
			if !slices.Equal(alternates, []string{"You again."}) {
				t.Errorf("alternate_greetings = %v", alternates)
			}
		})
	}
}

func TestACCv2CardCarriesNoV3OnlyKeys(t *testing.T) {
	t.Parallel()
	work := format.ExportWork{
		Type:   Type,
		Header: format.Header{Name: "Ana", Nickname: "Archivist"},
		Elements: []block.Element{
			prose(block.RoleDescription, "Quiet"),
			greetings("Hello"),
			{
				ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleGroupGreetings,
				Content: block.TextSet{Texts: []block.TextItem{
					{ID: uuid.New(), Text: "All of you made it."},
				}},
			},
		},
		Preserved: []format.Remainder{{
			Owner: format.OwnerWork, Namespace: cardNamespace,
			Payload: []byte(`{"assets":[{"type":"emotion","uri":"data:image/png;base64,AA"}],"creation_date":1717200000,"tags":["archivist"]}`),
		}},
	}

	body := writtenBody(t, write(t, CCv2Module{}, work).Body, V2)
	for _, key := range v3OnlyKeys {
		if _, present := body[key]; present {
			t.Errorf("a CCv2 card carried %q", key)
		}
	}
	if _, present := body["tags"]; !present {
		t.Error("a preserved key a v2 card has a place for did not come back")
	}
}

func TestACardIsWrittenIntoThePictureItBelongsTo(t *testing.T) {
	t.Parallel()
	picture := testPNG(t)
	withCover := format.ExportWork{
		Type:     Type,
		Header:   format.Header{Name: "Ana"},
		Elements: []block.Element{prose(block.RoleDescription, "Quiet"), greetings("Hello")},
		Cover:    &format.ExportMedia{MediaType: "image/png", Data: picture},
	}
	embedded := write(t, CCv3Module{}, withCover)
	if embedded.MediaType != "image/png" || embedded.Extension != ".png" {
		t.Fatalf("artifact = %s%s, want a PNG", embedded.MediaType, embedded.Extension)
	}
	if !bytes.HasPrefix(embedded.Body, pngSignature) {
		t.Fatal("the written file is not a PNG")
	}
	body := writtenBody(t, embedded.Body, V3)
	if !bytes.Contains(body["assets"], []byte(defaultFileURI)) {
		t.Errorf("works = %s, want the icon to point at the container", body["assets"])
	}

	withoutCover := withCover
	withoutCover.Cover = nil
	plain := write(t, CCv3Module{}, withoutCover)
	if plain.MediaType != "application/json" || plain.Extension != ".json" {
		t.Fatalf("artifact = %s%s, want a JSON document", plain.MediaType, plain.Extension)
	}
}

func TestANonPNGCoverWritesADocumentAndKeepsThePicture(t *testing.T) {
	t.Parallel()
	work := format.ExportWork{
		Type:     Type,
		Header:   format.Header{Name: "Ana"},
		Elements: []block.Element{prose(block.RoleDescription, "Quiet"), greetings("Hello")},
		Cover:    &format.ExportMedia{MediaType: "image/jpeg", Data: []byte("\xff\xd8\xff not a png")},
	}
	written := write(t, CCv3Module{}, work)
	if written.MediaType != "application/json" {
		t.Fatalf("media type = %q, want a JSON document", written.MediaType)
	}
	body := writtenBody(t, written.Body, V3)
	if !bytes.Contains(body["assets"], []byte("data:image/jpeg;base64,")) {
		t.Errorf("works = %s, want the cover inlined", body["assets"])
	}
}

func TestCharXWritesEveryPictureAsAFileTheCardNames(t *testing.T) {
	t.Parallel()
	expressionID, galleryID := uuid.New(), uuid.New()
	work := format.ExportWork{
		Type:   Type,
		Header: format.Header{Name: "Ana"},
		Elements: []block.Element{
			prose(block.RoleDescription, "Quiet"),
			greetings("Hello"),
			images(block.RoleExpressions, block.ImageItem{
				ID: uuid.New(), MediaID: expressionID, Name: "happy",
			}),
			images(block.RoleGallery, block.ImageItem{ID: uuid.New(), MediaID: galleryID}),
		},
		Cover: &format.ExportMedia{MediaType: "image/png", Data: testPNG(t)},
		Images: map[uuid.UUID]format.ExportMedia{
			expressionID: {MediaType: "image/png", Data: []byte("happy-bytes")},
			galleryID:    {MediaType: "image/webp", Data: []byte("gallery-bytes")},
		},
	}

	written := write(t, CharXModule{}, work)
	if written.Extension != ".charx" {
		t.Fatalf("extension = %q, want .charx", written.Extension)
	}
	files := archiveEntries(t, written.Body)
	var records []cardFileRecord
	body := writtenBody(t, files["card.json"], V3)
	if err := json.Unmarshal(body["assets"], &records); err != nil {
		t.Fatalf("read the written work list: %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("work records = %+v, want the cover, the expression and the gallery image", records)
	}
	for _, record := range records {
		entry, embedded := strings.CutPrefix(record.URI, embeddedPrefix)
		if !embedded {
			t.Fatalf("work %q points at %q, want a file in the archive", record.Name, record.URI)
		}
		if _, held := files[entry]; !held {
			t.Errorf("the archive has no %q", entry)
		}
	}
	if !bytes.Equal(files["assets/emotion/image/2.png"], []byte("happy-bytes")) {
		t.Errorf("the expression file holds %q", files["assets/emotion/image/2.png"])
	}
	if !bytes.Equal(files["assets/other/image/3.webp"], []byte("gallery-bytes")) {
		t.Errorf("the gallery file holds %q", files["assets/other/image/3.webp"])
	}
}

func TestCharXPutsEachPictureWhereTheAppsThatReadOneLook(t *testing.T) {
	t.Parallel()
	expressionID, galleryID := uuid.New(), uuid.New()
	work := format.ExportWork{
		Type:   Type,
		Header: format.Header{Name: "Ana"},
		Elements: []block.Element{
			prose(block.RoleDescription, "Quiet"),
			greetings("Hello"),
			images(block.RoleExpressions, block.ImageItem{
				ID: uuid.New(), MediaID: expressionID, Name: "happy",
			}),
			images(block.RoleGallery, block.ImageItem{ID: uuid.New(), MediaID: galleryID}),
		},
		Cover: &format.ExportMedia{MediaType: "image/png", Data: testPNG(t)},
		Images: map[uuid.UUID]format.ExportMedia{
			expressionID: {MediaType: "image/png", Data: []byte("happy-bytes")},
			galleryID:    {MediaType: "image/png", Data: []byte("gallery-bytes")},
		},
	}

	files := archiveEntries(t, write(t, CharXModule{}, work).Body)
	for _, wanted := range []string{
		"assets/icon/image/main.png",
		"assets/emotion/image/2.png",
		"assets/other/image/3.png",
	} {
		if _, held := files[wanted]; !held {
			t.Errorf("the archive has no %s: %v", wanted, slices.Sorted(maps.Keys(files)))
		}
	}
}

func TestACardWrittenBackReadsAsTheSameContent(t *testing.T) {
	t.Parallel()
	source := `{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Keeps the archive.","personality":"Dry",
			"scenario":"A ledger goes missing.","first_mes":"Hello",
			"alternate_greetings":["You again."],
			"group_only_greetings":["All of you made it."],
			"mes_example":"<START>\nAna: Sit down.\nYou: I brought the ledger.",
			"creator_notes":"Built over a weekend.",
			"character_book":{"entries":[{"keys":["ledger"],"content":"A debt."}]}}
	}`
	parsed := resolveAndParse(t, jsonCard(t, source))
	written := write(t, CCv3Module{}, exportWorkOf(parsed))
	reread := resolveAndParse(t, inspect(t, written.Body, "ana.json"))

	for _, role := range []block.Role{
		block.RoleDescription, block.RolePersonality, block.RoleScenario,
		block.RoleCreatorNotes,
	} {
		before, _ := elementContent(parsed.Elements, role)
		after, _ := elementContent(reread.Elements, role)
		if before.(block.Prose).Text != after.(block.Prose).Text {
			t.Errorf("%s = %q, want %q", role, after, before)
		}
	}
	for _, role := range []block.Role{block.RoleGreetings, block.RoleGroupGreetings} {
		before, _ := elementContent(parsed.Elements, role)
		after, ok := elementContent(reread.Elements, role)
		if !ok {
			t.Fatalf("%s did not survive the write", role)
		}
		if !slices.Equal(textsOf(before.(block.TextSet).Texts), textsOf(after.(block.TextSet).Texts)) {
			t.Errorf("%s = %+v, want %+v", role, after, before)
		}
	}
	before, _ := elementContent(parsed.Elements, block.RoleExampleDialogue)
	after, _ := elementContent(reread.Elements, block.RoleExampleDialogue)
	if len(after.(block.DialogueSample).Turns) != len(before.(block.DialogueSample).Turns) {
		t.Errorf("example dialogue = %+v, want %+v", after, before)
	}
	book, ok := elementContent(reread.Elements, block.RoleLorebookEntries)
	if !ok || book.(block.EntryTable).Entries[0].Text != "A debt." {
		t.Errorf("the book = %+v, want the entry back", book)
	}
}

func write(t *testing.T, module format.Module, work format.ExportWork) format.Artifact {
	t.Helper()
	writer, ok := module.(format.Writer)
	if !ok {
		t.Fatalf("module %q declares no writer", module.ID())
	}
	artifact, err := writer.Write(context.Background(), work)
	if err != nil {
		t.Fatalf("write %s: %v", module.ID(), err)
	}
	return artifact
}

func exportWorkOf(parsed format.Parsed) format.ExportWork {
	return format.ExportWork{
		Type: parsed.Type, Header: parsed.Header,
		Elements: parsed.Elements, Preserved: parsed.Remainder,
	}
}

func writtenBody(t *testing.T, artifact []byte, formatID string) map[string]json.RawMessage {
	t.Helper()
	card := artifact
	if bytes.HasPrefix(artifact, pngSignature) {
		card = cardChunks(t, artifact)[chunkName(formatID)]
	} else if !json.Valid(artifact) {
		card = archiveEntries(t, artifact)["card.json"]
	}
	var read struct {
		Spec string                     `json:"spec"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(card, &read); err != nil {
		t.Fatalf("read the written card: %v", err)
	}
	wanted := formatID
	if formatID == CharX {
		wanted = V3
	}
	if read.Spec != wanted {
		t.Fatalf("spec = %q, want %q", read.Spec, wanted)
	}
	return read.Data
}

func archiveEntries(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open the written archive: %v", err)
	}
	entries := make(map[string][]byte, len(archive.File))
	for _, file := range archive.File {
		opened, openErr := file.Open()
		if openErr != nil {
			t.Fatalf("open %q: %v", file.Name, openErr)
		}
		contents, readErr := io.ReadAll(opened)
		_ = opened.Close()
		if readErr != nil {
			t.Fatalf("read %q: %v", file.Name, readErr)
		}
		entries[file.Name] = contents
	}
	return entries
}

func text(t *testing.T, raw json.RawMessage) string {
	t.Helper()
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("read %s as a string: %v", raw, err)
	}
	return value
}

func prose(role block.Role, body string) block.Element {
	return block.Element{
		ID: uuid.New(), Type: block.TypeProse, Role: role, Content: block.Prose{Text: body},
	}
}

func greetings(texts ...string) block.Element {
	items := make([]block.TextItem, 0, len(texts))
	for _, body := range texts {
		items = append(items, block.TextItem{ID: uuid.New(), Text: body})
	}
	return block.Element{
		ID: uuid.New(), Type: block.TypeTextSet, Role: block.RoleGreetings,
		Content: block.TextSet{Texts: items},
	}
}

func images(role block.Role, items ...block.ImageItem) block.Element {
	return block.Element{
		ID: uuid.New(), Type: block.TypeImageSet, Role: role,
		Content: block.ImageSet{Images: items},
	}
}

func TestEveryCharacterOriginWritesEveryCharacterTarget(t *testing.T) {
	t.Parallel()
	body := `"name":"Ana","description":"Keeps the archive.","first_mes":"Hello",
		"alternate_greetings":["You again."],
		"character_book":{"entries":[{"keys":["ledger"],"content":"A debt."}]}`
	origins := map[string]format.Inspection{
		V2:    jsonCard(t, `{"spec":"chara_card_v2","spec_version":"2.0","data":{`+body+`}}`),
		V3:    jsonCard(t, `{"spec":"chara_card_v3","spec_version":"3.0","data":{`+body+`}}`),
		CharX: charxCard(t, `{"spec":"chara_card_v3","spec_version":"3.0","data":{`+body+`}}`, nil),
	}
	for _, module := range Modules() {
		declaration := module.Declaration()
		for origin, file := range origins {
			if !slices.Contains(declaration.TestedOrigins, origin) {
				t.Fatalf("%s declares no tested origin for %s", module.ID(), origin)
			}
			t.Run(origin+"-to-"+module.ID(), func(t *testing.T) {
				parsed := resolveAndParse(t, file)
				written := write(t, module, exportWorkOf(parsed))
				fields := writtenBody(t, written.Body, module.ID())
				if got := text(t, fields["description"]); got != "Keeps the archive." {
					t.Errorf("description = %q", got)
				}
				if got := text(t, fields["first_mes"]); got != "Hello" {
					t.Errorf("first_mes = %q", got)
				}
				if !bytes.Contains(fields[bookKey], []byte("A debt.")) {
					t.Errorf("the book = %s, want its entry", fields[bookKey])
				}
			})
		}
	}
}

func TestAV3CardCarriesAV2CopyOfItself(t *testing.T) {
	t.Parallel()
	work := format.ExportWork{
		Type:     Type,
		Header:   format.Header{Name: "Ana", Nickname: "Archivist"},
		Elements: []block.Element{prose(block.RoleDescription, "Quiet"), greetings("Hello")},
		Cover:    &format.ExportMedia{MediaType: "image/png", Data: testPNG(t)},
	}
	for _, module := range []format.Module{CCv3Module{}, CharXModule{}} {
		if module.ID() == CharX {
			continue
		}
		t.Run(module.ID(), func(t *testing.T) {
			picture := write(t, module, work).Body
			chunks := cardChunks(t, picture)
			if len(chunks) != 2 {
				t.Fatalf("card chunks = %v, want the v3 card and its v2 copy", keysOf(chunks))
			}
			v3 := readCardText(t, chunks["ccv3"])
			v2 := readCardText(t, chunks["chara"])
			if v3["spec"] == nil || string(v3["spec"]) != `"chara_card_v3"` {
				t.Errorf("the ccv3 chunk holds spec %s", v3["spec"])
			}
			if string(v2["spec"]) != `"chara_card_v2"` {
				t.Errorf("the chara chunk holds spec %s", v2["spec"])
			}
			body := decodeObject(t, v2["data"])
			if text(t, body["description"]) != "Quiet" {
				t.Errorf("the v2 copy lost the description: %s", v2["data"])
			}
			for _, key := range v3OnlyKeys {
				if _, present := body[key]; present {
					t.Errorf("the v2 copy carries %q", key)
				}
			}
		})
	}
}

func TestAV3DocumentRepeatsTheFieldsAnOlderReaderLooksFor(t *testing.T) {
	t.Parallel()
	work := format.ExportWork{
		Type:     Type,
		Header:   format.Header{Name: "Ana"},
		Elements: []block.Element{prose(block.RoleDescription, "Quiet"), greetings("Hello")},
	}
	written := write(t, CCv3Module{}, work)
	document := decodeObject(t, written.Body)
	if text(t, document["name"]) != "Ana" || text(t, document["description"]) != "Quiet" ||
		text(t, document["first_mes"]) != "Hello" {
		t.Fatalf("the document has no legacy fields: %s", written.Body)
	}
	if text(t, decodeObject(t, document["data"])["description"]) != "Quiet" {
		t.Errorf("the spec body lost the description: %s", document["data"])
	}
}

func cardChunks(t *testing.T, picture []byte) map[string][]byte {
	t.Helper()
	found := make(map[string][]byte)
	err := visitPNGChunks(picture, func(chunkType string, data, _ []byte) error {
		if !isCardChunk(chunkType, data) {
			return nil
		}
		keyword, encoded, _ := bytes.Cut(data, []byte{0})
		decoded, decodeErr := base64.StdEncoding.DecodeString(string(encoded))
		if decodeErr != nil {
			return decodeErr
		}
		found[string(keyword)] = decoded
		return nil
	})
	if err != nil {
		t.Fatalf("read the card chunks: %v", err)
	}
	return found
}

func readCardText(t *testing.T, card []byte) map[string]json.RawMessage {
	t.Helper()
	if card == nil {
		t.Fatal("the picture has no card under that keyword")
	}
	return decodeObject(t, card)
}

func keysOf(found map[string][]byte) []string {
	names := make([]string, 0, len(found))
	for name := range found {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

func decodeObject(t *testing.T, source []byte) map[string]json.RawMessage {
	t.Helper()
	var object map[string]json.RawMessage
	if err := json.Unmarshal(source, &object); err != nil {
		t.Fatalf("read %s as an object: %v", source, err)
	}
	return object
}

func TestEachCardWritesEveryHeaderFieldItDeclares(t *testing.T) {
	t.Parallel()
	written := map[format.HeaderField]string{
		format.HeaderName:           "name",
		format.HeaderCreditedAuthor: "creator",
		format.HeaderWorkVersion:    "character_version",
		format.HeaderNickname:       "nickname",
	}
	subject := format.ExportWork{
		Type: Type,
		Header: format.Header{
			Name: "Ana", WorkVersion: "1.2", CreditedAuthor: "Wren", Nickname: "Archivist",
		},
		Elements: []block.Element{
			prose(block.RoleDescription, "Keeps the archive."),
			greetings("Hello"),
		},
	}
	for _, module := range Modules() {
		t.Run(module.ID(), func(t *testing.T) {
			declaration := module.Declaration()
			body := writtenBody(t, write(t, module, subject).Body, module.ID())
			for field, key := range written {
				value, carried := body[key]
				declared := slices.Contains(declaration.Header, field)
				if declared && (!carried || text(t, value) == "") {
					t.Errorf("%s is declared but %q is not in the file", field, key)
				}
				if !declared && carried && text(t, value) != "" {
					t.Errorf("%q is in the file but %s is not declared", key, field)
				}
			}
		})
	}
}

func TestAWriterMintsNoRecordForAGalleryImageItWasNotGiven(t *testing.T) {
	t.Parallel()
	travelling, left := uuid.New(), uuid.New()
	work := format.ExportWork{
		Type:   Type,
		Header: format.Header{Name: "Ana"},
		Elements: []block.Element{
			prose(block.RoleDescription, "Quiet"),
			greetings("Hello"),
			images(block.RoleGallery, block.ImageItem{
				ID: uuid.New(), MediaID: travelling, Name: "At the door",
			}),
		},
		Images: map[uuid.UUID]format.ExportMedia{
			travelling: {MediaType: "image/png", Data: []byte("door-bytes")},
			left:       {MediaType: "image/png", Data: []byte("stair-bytes")},
		},
	}

	for _, module := range []format.Writer{CCv3Module{}, CharXModule{}} {
		card := writtenCard(t, module, write(t, module, work))
		var records []cardFileRecord
		if err := json.Unmarshal(card["assets"], &records); err != nil {
			t.Fatalf("read %s's work list: %v", module.ID(), err)
		}
		if len(records) != 1 || records[0].Name != "At the door" {
			t.Fatalf("%s wrote %+v, want the one gallery image it was given",
				module.ID(), records)
		}
	}
}

func writtenCard(
	t *testing.T,
	module format.Writer,
	written format.Artifact,
) map[string]json.RawMessage {
	t.Helper()
	if module.ID() != CharX {
		return writtenBody(t, written.Body, V3)
	}
	return writtenBody(t, archiveEntries(t, written.Body)["card.json"], V3)
}
