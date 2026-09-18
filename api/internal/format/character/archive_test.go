package character

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
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
	modules := []byte(`{"scripts":[{"type":"regex","in":"a","out":"b"}]}`)
	parsed := resolveAndParse(t, charxWithMembers(t, plainCard, map[string][]byte{
		"lumiverse_modules.json": modules,
	}))

	written := write(t, CharXModule{}, format.ExportWork{
		Type: Type, Header: format.Header{Name: "Ana"},
		Elements: parsed.Elements, Preserved: parsed.Remainder,
	})
	if held := archiveEntry(t, written.Body, "lumiverse_modules.json"); !bytes.Equal(held, modules) {
		t.Fatalf("gave back %q, want %q", held, modules)
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

func TestAnArchivedFileTooLargeToKeepTurnsTheCardAway(t *testing.T) {
	t.Parallel()
	file := charxWithMembers(t, plainCard, map[string][]byte{
		"huge.bin": incompressible(maxArchiveMemberBytes + 1),
	})
	module := CharXModule{}
	claim, held := module.Claim(file)
	if !held {
		t.Fatal("the module claimed nothing")
	}

	_, err := module.Parse(context.Background(), file, claim)
	reason, classified := format.FailureOf(err)
	if !classified || reason != format.FailureLimitExceeded {
		t.Fatalf("err = %v, want a refusal naming the limit", err)
	}
	if !strings.Contains(err.Error(), "huge.bin") {
		t.Fatalf("err = %v, want it to name the archived file", err)
	}
}

func incompressible(size int) []byte {
	held := make([]byte, size)
	source := rand.NewChaCha8([32]byte{})
	source.Read(held)
	return held
}

func TestAnArchivedFileNeverClaimsAPathTheWriterProduces(t *testing.T) {
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
