package character

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/block"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/media"
	"github.com/google/uuid"
)

func TestACharXKeepsTheFilesIllarinReadsNothingFrom(t *testing.T) {
	t.Parallel()
	modules := []byte(`{"scripts":[{"type":"regex","in":"a","out":"b"}]}`)
	parsed := resolveAndParse(t, charxWithMembers(t, plainCard, map[string][]byte{
		"lumiverse_modules.json": modules,
	}))

	kept := preservedByNamespace(parsed.Remainder)
	held, ok := kept["archive:lumiverse_modules.json"]
	if !ok {
		t.Fatalf("namespaces = %v, want the archived file kept", slices.Sorted(maps.Keys(kept)))
	}
	given, readable := ArchivedMember(held)
	if !readable || !bytes.Equal(given, modules) {
		t.Fatalf("kept %q, want %q", given, modules)
	}
}

func TestACharXGivesBackTheFilesIllarinReadsNothingFrom(t *testing.T) {
	t.Parallel()
	expressions := make(map[string][]byte)
	for index := range 22 {
		expressions[fmt.Sprintf("assets/other/image/expr_%02d.avif", index)] = incompressible(200000)
	}
	for _, test := range []struct {
		name    string
		members map[string][]byte
	}{
		{"module settings", map[string][]byte{
			"lumiverse_modules.json": []byte(`{"scripts":[{"type":"regex","in":"a","out":"b"}]}`),
		}},
		{"one extra over 1 MiB", map[string][]byte{"large.bin": incompressible(1<<20 + 1)}},
		{"AVIF expressions over 4 MiB combined", expressions},
	} {
		t.Run(test.name, func(t *testing.T) {
			parsed := resolveAndParse(t, charxWithMembers(t, plainCard, test.members))
			written := write(t, CharXModule{}, format.ExportWork{
				Type: Type, Header: format.Header{Name: "Ana"},
				Elements: parsed.Elements, Preserved: parsed.Remainder,
			})
			for name, original := range test.members {
				if held := archiveEntry(t, written.Body, name); !bytes.Equal(held, original) {
					t.Fatalf("%s changed on download: got %d bytes, want %d unchanged bytes", name, len(held), len(original))
				}
			}
		})
	}
}

func TestACardBodyNeverCarriesAnArchivedFile(t *testing.T) {
	t.Parallel()
	parsed := resolveAndParse(t, charxWithMembers(t, plainCard, map[string][]byte{
		"lumiverse_modules.json": []byte(`{"scripts":[]}`),
	}))

	for _, module := range []format.Module{CCv2Module{}, CCv3Module{}} {
		written := write(t, module, format.ExportWork{
			Type: Type, Header: format.Header{Name: "Ana"},
			Elements: parsed.Elements, Preserved: parsed.Remainder,
		})
		if strings.Contains(string(written.Body), "archive:") {
			t.Fatalf("%s carried an archived file into the card", module.ID())
		}
	}
}

func incompressible(size int) []byte {
	held := make([]byte, size)
	source := rand.NewChaCha8([32]byte{})
	source.Read(held)
	return held
}

func TestAnArchivedFileNeverMatchesAPathTheWriterProduces(t *testing.T) {
	t.Parallel()
	written := []archivedFile{{path: "assets/icon/image/main.png", data: []byte("picture")}}
	preserved := []format.Remainder{
		archived("card.json", "card"),
		archived("assets/icon/image/main.png", "not the picture"),
		archived("../escape.json", "outside"),
		archived("/absolute.json", "rooted"),
		archived("windows\\path.json", "backslashed"),
		archived("notes.txt", "keep me"),
	}

	given := archivedMemberFiles(preserved, written)
	if len(given) != 1 || given[0].path != "notes.txt" {
		t.Fatalf("wrote %v, want only notes.txt", given)
	}
}

func archived(name, body string) format.Remainder {
	payload, err := json.Marshal(archivedMember{
		Bytes: base64.StdEncoding.EncodeToString([]byte(body)),
	})
	if err != nil {
		panic(err)
	}
	return format.Remainder{
		Owner: format.OwnerWork, Namespace: MemberNamespace + name, Payload: payload,
	}
}

const plainCard = `{"spec":"chara_card_v3","spec_version":"3.0","data":{"name":"Ana"}}`

func charxWithMembers(t *testing.T, body string, members map[string][]byte) format.Inspection {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	put := func(name string, content []byte) {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatalf("create archive entry %q: %v", name, err)
		}
		if _, err := entry.Write(content); err != nil {
			t.Fatalf("write archive entry %q: %v", name, err)
		}
	}
	put("card.json", []byte(body))
	for _, name := range slices.Sorted(maps.Keys(members)) {
		put(name, members[name])
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close archive: %v", err)
	}
	return inspect(t, file.Bytes(), "card.charx")
}

func preservedByNamespace(rows []format.Remainder) map[string][]byte {
	kept := make(map[string][]byte, len(rows))
	for _, row := range rows {
		kept[row.Namespace] = row.Payload
	}
	return kept
}

func archiveEntry(t *testing.T, body []byte, name string) []byte {
	t.Helper()
	archive, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("read the written archive: %v", err)
	}
	for _, entry := range archive.File {
		if entry.Name != name {
			continue
		}
		opened, err := entry.Open()
		if err != nil {
			t.Fatalf("open %q: %v", name, err)
		}
		defer opened.Close()
		held, err := io.ReadAll(opened)
		if err != nil {
			t.Fatalf("read %q: %v", name, err)
		}
		return held
	}
	t.Fatalf("the written archive holds no %q", name)
	return nil
}

func TestACharXNamesItsExpressionsForLumiverse(t *testing.T) {
	t.Parallel()
	stale := `{"version":1,"regex_scripts":[{"name":"keep"}],
		"expressions":{"enabled":false,"defaultExpression":"happy",
			"mappings":{"happy":"assets/other/image/expr_happy.png"}}}`
	for _, test := range []struct {
		name      string
		preserved []format.Remainder
		enabled   bool
		fallback  string
	}{
		{"made here", nil, true, ""},
		{"made in Lumiverse", []format.Remainder{archived(lumiverseModulesEntry, stale)}, false, "happy"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			happy, sad := uuid.New(), uuid.New()
			written := write(t, CharXModule{}, format.ExportWork{
				Type: Type, Header: format.Header{Name: "Ana"},
				Elements: []block.Element{
					greetings("Hello"),
					images(block.RoleExpressions,
						block.ImageItem{ID: uuid.New(), MediaID: happy, Name: "happy"},
						block.ImageItem{ID: uuid.New(), MediaID: sad, Name: "Twin/sad"},
					),
				},
				Images: map[uuid.UUID]format.ExportMedia{
					happy: {MediaType: "image/png", Data: []byte("happy-bytes")},
					sad:   {MediaType: "image/png", Data: []byte("sad-bytes")},
				},
				Preserved: test.preserved,
			})
			files := archiveEntries(t, written.Body)
			var modules struct {
				Version     int `json:"version"`
				Expressions struct {
					Enabled           bool              `json:"enabled"`
					DefaultExpression string            `json:"defaultExpression"`
					Mappings          map[string]string `json:"mappings"`
				} `json:"expressions"`
				ExpressionGroups struct {
					Groups map[string]map[string]string `json:"groups"`
				} `json:"expression_groups"`
				RegexScripts []json.RawMessage `json:"regex_scripts"`
			}
			if err := json.Unmarshal(files[lumiverseModulesEntry], &modules); err != nil {
				t.Fatalf("read %s: %v (entries %v)", lumiverseModulesEntry, err, slices.Sorted(maps.Keys(files)))
			}
			if got := files[modules.Expressions.Mappings["happy"]]; !bytes.Equal(got, []byte("happy-bytes")) {
				t.Errorf("happy points at %q holding %q", modules.Expressions.Mappings["happy"], got)
			}
			if got := files[modules.ExpressionGroups.Groups["Twin"]["sad"]]; !bytes.Equal(got, []byte("sad-bytes")) {
				t.Errorf("Twin/sad points at %q holding %q", modules.ExpressionGroups.Groups["Twin"]["sad"], got)
			}
			if modules.Version != 1 || modules.Expressions.Enabled != test.enabled ||
				modules.Expressions.DefaultExpression != test.fallback {
				t.Errorf("modules = %+v", modules)
			}
			if test.preserved != nil && len(modules.RegexScripts) != 1 {
				t.Errorf("regex scripts = %s, want the kept one", modules.RegexScripts)
			}
		})
	}
}

func TestACharXGivesBackEveryPictureIllarinDoesNotWriteItself(t *testing.T) {
	t.Parallel()
	card := `{"spec":"chara_card_v3","spec_version":"3.0","data":{"name":"Ana","assets":[
		{"type":"icon","uri":"embeded://assets/icon/image/main.png","name":"main","ext":"png"},
		{"type":"icon","uri":"embeded://assets/icon/image/alt.png","name":"Winter","ext":"png"},
		{"type":"user_icon","uri":"embeded://assets/user/you.png","name":"you","ext":"png"}
	]}}`
	members := map[string][]byte{
		"assets/icon/image/main.png":             testPNG(t),
		"assets/icon/image/alt.png":              testPNG(t),
		"assets/user/you.png":                    testPNG(t),
		"assets/other/image/landing_layer_a.png": testPNG(t),
		lumiverseModulesEntry: []byte(`{"version":1,
			"alternate_avatars":[{"id":"w","label":"Winter","path":"assets/icon/image/alt.png"}],
			"landing_perspective_layers":[{"id":"a","path":"assets/other/image/landing_layer_a.png","intensity":1}]}`),
	}
	parsed := resolveAndParse(t, charxWithMembers(t, card, members))
	for _, picture := range parsed.Media {
		if picture.Role != media.Avatar {
			t.Errorf("read a %s picture, want only the avatar", picture.Role)
		}
	}

	written := write(t, CharXModule{}, format.ExportWork{
		Type: Type, Header: format.Header{Name: "Ana"}, Elements: parsed.Elements,
		Cover:     &format.ExportMedia{MediaType: "image/png", Data: testPNG(t)},
		Preserved: parsed.Remainder,
	})
	files := archiveEntries(t, written.Body)
	for _, name := range []string{
		"assets/icon/image/alt.png", "assets/user/you.png", "assets/other/image/landing_layer_a.png",
	} {
		if !bytes.Equal(files[name], members[name]) {
			t.Errorf("the archive does not give back %s", name)
		}
	}
	var listed []cardFileRecord
	if err := json.Unmarshal(writtenBody(t, files[cardEntry], V3)["assets"], &listed); err != nil {
		t.Fatal(err)
	}
	uris := make([]string, 0, len(listed))
	for _, record := range listed {
		uris = append(uris, record.URI)
	}
	slices.Sort(uris)
	want := []string{
		"embeded://assets/icon/image/alt.png", "embeded://assets/icon/image/main.png",
		"embeded://assets/user/you.png",
	}
	if !slices.Equal(uris, want) {
		t.Errorf("card lists %v, want %v", uris, want)
	}
}
