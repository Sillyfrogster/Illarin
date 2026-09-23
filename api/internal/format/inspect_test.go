package format

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/google/uuid"
)

var errReadUnavailable = errors.New("read unavailable")

type recordingStore struct {
	data  []byte
	reads []byteRange
}

type failingStore struct{}

func (failingStore) ReadRange(context.Context, uuid.UUID, int64, int64) (io.ReadCloser, error) {
	return nil, errReadUnavailable
}

func TestInspectDistinguishesMalformedInputFromAStorageFailure(t *testing.T) {
	t.Parallel()
	malformed := []byte(`{"broken":`)
	_, malformedErr := Inspect(
		context.Background(), &recordingStore{data: malformed}, uuid.New(), int64(len(malformed)), "card.json",
	)
	if !errors.Is(malformedErr, ErrMalformedInput) {
		t.Fatalf("malformed error = %v, want ErrMalformedInput", malformedErr)
	}

	_, readErr := Inspect(context.Background(), failingStore{}, uuid.New(), 8, "card.json")
	if !errors.Is(readErr, errReadUnavailable) {
		t.Fatalf("read error = %v, want storage cause", readErr)
	}
	if errors.Is(readErr, ErrMalformedInput) {
		t.Fatalf("storage error was reported as malformed input: %v", readErr)
	}
}

func TestInspectStreamsAJSONRootThroughRangeReads(t *testing.T) {
	t.Parallel()
	file := []byte(`{"padding":"` + strings.Repeat("x", 128*1024) + `","spec":"chara_card_v3"}`)
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "card.json")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got.Container != JSON {
		t.Fatalf("container = %q, want JSON", got.Container)
	}
	if len(got.Payloads) != 1 {
		t.Fatalf("payload count = %d, want 1", len(got.Payloads))
	}
	if spec, ok := got.Payloads[0].String("spec"); !ok || spec != "chara_card_v3" {
		t.Errorf("spec = %q, %v; want chara_card_v3, true", spec, ok)
	}
	for _, read := range store.reads {
		if read.length > maxRangeRead {
			t.Fatalf("range read length = %d, want at most %d", read.length, maxRangeRead)
		}
	}
}

func TestJSONFilenameDoesNotTurnOpaqueBytesIntoJSON(t *testing.T) {
	t.Parallel()
	file := []byte{0x00, 0xff, 0xfe, 0x10}
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "misleading.json")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got.Container != Unknown {
		t.Fatalf("container = %q, want unknown", got.Container)
	}
}

func TestJSONFilenameAllowsWhitespaceBeyondTheSignatureRead(t *testing.T) {
	t.Parallel()
	file := []byte(strings.Repeat(" ", 32) + `{"spec":"chara_card_v3"}`)
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "card.json")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got.Container != JSON {
		t.Fatalf("container = %q, want JSON", got.Container)
	}
}

func TestInspectRejectsAWebPSignatureWithoutAnImage(t *testing.T) {
	t.Parallel()
	file := []byte("RIFF\x0c\x00\x00\x00WEBPVP8X\x00\x00\x00\x00")
	_, err := Inspect(
		context.Background(), &recordingStore{data: file}, uuid.New(), int64(len(file)), "image.webp",
	)
	if !errors.Is(err, ErrMalformedInput) {
		t.Fatalf("Inspect error = %v, want ErrMalformedInput", err)
	}
}

func TestInspectFindsTheFolderAnArchiveIsWrappedIn(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		entries  []string
		base     string
		payloads int
	}{
		"files at the top":    {entries: []string{"spindle.json", "dist/frontend.js"}, base: "", payloads: 1},
		"repository download": {entries: []string{"Tool-main/", "Tool-main/spindle.json", "Tool-main/dist/frontend.js"}, base: "Tool-main/", payloads: 1},
		"folder zipped on a Mac": {
			entries: []string{"Tool/manifest.json", "Tool/index.js", "Tool/.DS_Store", "__MACOSX/Tool/._manifest.json"},
			base:    "Tool/", payloads: 1,
		},
		"folders inside folders": {entries: []string{"downloads/Tool-main/spindle.json", "downloads/Tool-main/src/backend.ts"}, base: "downloads/Tool-main/", payloads: 1},
		"two folders":            {entries: []string{"one/spindle.json", "two/index.js"}, base: "", payloads: 0},
		"a file beside a folder": {entries: []string{"README.md", "tool/spindle.json"}, base: "", payloads: 0},
	}
	for name, test := range cases {
		t.Run(name, func(t *testing.T) {
			entries := make([]zipEntry, 0, len(test.entries))
			for _, entry := range test.entries {
				body := ""
				if path.Ext(entry) == ".json" && !strings.Contains(entry, "__MACOSX") {
					body = `{"name":"Tool"}`
				}
				entries = append(entries, zipEntry{name: entry, body: body, method: zip.Store})
			}
			file := zipEntries(t, entries...)
			got, err := Inspect(context.Background(), &recordingStore{data: file}, uuid.New(), int64(len(file)), "tool.zip")
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if got.ArchiveBase != test.base || len(got.Payloads) != test.payloads {
				t.Fatalf("base %q with %d payloads, want %q with %d", got.ArchiveBase, len(got.Payloads), test.base, test.payloads)
			}
		})
	}
}

func TestInspectTellsRootZIPEntriesApart(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		entryName string
		body      string
		payloads  int
	}{
		{name: "theme", entryName: "theme.json", body: `{"name":"Midnight"}`, payloads: 1},
		{name: "character", entryName: "card.json", body: `{"spec":"chara_card_v3"}`, payloads: 1},
		{name: "Spindle extension", entryName: "spindle.json", body: `{"identifier":"quiet"}`, payloads: 1},
		{name: "SillyTavern extension", entryName: "manifest.json", body: `{"js":"index.js"}`, payloads: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			file := zipFile(t, test.entryName, test.body)
			store := &recordingStore{data: file}

			got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "bundle.zip")
			if err != nil {
				t.Fatalf("Inspect: %v", err)
			}
			if got.Container != ZIP {
				t.Fatalf("container = %q, want ZIP", got.Container)
			}
			if len(got.ZIPEntries) != 1 || got.ZIPEntries[0].Name != test.entryName {
				t.Fatalf("entries = %+v, want one root %s", got.ZIPEntries, test.entryName)
			}
			if len(got.Payloads) != test.payloads {
				t.Fatalf("payload count = %d, want %d", len(got.Payloads), test.payloads)
			}
			if test.payloads == 1 && got.Payloads[0].Location.Name != test.entryName {
				t.Fatalf("payloads = %+v, want decoded %s", got.Payloads, test.entryName)
			}
		})
	}
}

func TestAThemeBundleExposesItsJSONAndReferencedFiles(t *testing.T) {
	t.Parallel()
	file := zipEntries(t,
		zipEntry{"theme.json", `{"format":3,"assets":[{"archivePath":"assets/host.woff2"}]}`, zip.Store},
		zipEntry{"assets/host.woff2", "font fixture", zip.Store},
	)
	inspected, err := Inspect(
		context.Background(), &recordingStore{data: file}, uuid.New(), int64(len(file)), "midnight.lumitheme",
	)
	if err != nil {
		t.Fatalf("inspect theme bundle: %v", err)
	}
	if len(inspected.Payloads) != 1 || inspected.Payloads[0].Location.Name != "theme.json" {
		t.Fatalf("payloads = %+v, want theme.json", inspected.Payloads)
	}
	opened, err := inspected.OpenZIPEntry(context.Background(), "assets/host.woff2")
	if err != nil {
		t.Fatalf("open theme work: %v", err)
	}
	defer opened.Close()
	font, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read theme work: %v", err)
	}
	if string(font) != "font fixture" {
		t.Errorf("theme work = %q, want the archived bytes", font)
	}
}

func TestInspectEnforcesEveryArchiveResourceLimit(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		file   func(t *testing.T) []byte
		limits func() Limits
		rule   string
	}{
		{
			name: "entry count",
			rule: "it holds 2 files, and an archive may hold 1",
			file: func(t *testing.T) []byte {
				return zipEntries(t, zipEntry{"one", "1", zip.Store}, zipEntry{"two", "2", zip.Store})
			},
			limits: func() Limits {
				limits := DefaultLimits()
				limits.MaxArchiveEntries = 1
				return limits
			},
		},
		{
			name: "entry bytes",
			rule: `"large" unpacks to 5 bytes, and one file may unpack to 4 bytes`,
			file: func(t *testing.T) []byte {
				return zipEntries(t, zipEntry{"large", "12345", zip.Store})
			},
			limits: func() Limits {
				limits := DefaultLimits()
				limits.MaxEntryBytes = 4
				return limits
			},
		},
		{
			name: "total bytes",
			rule: "it unpacks to more than 5 bytes",
			file: func(t *testing.T) []byte {
				return zipEntries(t, zipEntry{"one", "123", zip.Store}, zipEntry{"two", "456", zip.Store})
			},
			limits: func() Limits {
				limits := DefaultLimits()
				limits.MaxArchiveBytes = 5
				return limits
			},
		},
		{
			name: "compression ratio",
			rule: `"compressed" unpacks to more than 2 times its packed size`,
			file: func(t *testing.T) []byte {
				return zipEntries(t, zipEntry{"compressed", strings.Repeat("0", 4096), zip.Deflate})
			},
			limits: func() Limits {
				limits := DefaultLimits()
				limits.MaxCompressionRatio = 2
				return limits
			},
		},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			file := test.file(t)
			_, err := InspectWithLimits(
				context.Background(), &recordingStore{data: file}, uuid.New(),
				int64(len(file)), "bundle.zip", test.limits(),
			)
			var violation ArchiveViolation
			if !errors.Is(err, ErrSafetyViolation) || !errors.As(err, &violation) || violation.Rule != test.rule {
				t.Fatalf("InspectWithLimits error = %v, want the safety rule %q", err, test.rule)
			}
		})
	}
}

type byteRange struct {
	offset int64
	length int64
}

func (s *recordingStore) ReadRange(_ context.Context, _ uuid.UUID, offset, length int64) (io.ReadCloser, error) {
	s.reads = append(s.reads, byteRange{offset: offset, length: length})
	return io.NopCloser(bytes.NewReader(s.data[offset : offset+length])), nil
}

func TestInspectFindsAJSONPayloadInALatePNGChunk(t *testing.T) {
	t.Parallel()
	payload := base64.StdEncoding.EncodeToString([]byte(`{"spec":"chara_card_v2","name":"Late card"}`))
	file := pngFile(
		pngChunk("IHDR", make([]byte, 13)),
		pngChunk("IDAT", make([]byte, 1024)),
		pngChunk("tEXt", append([]byte("ccv3\x00"), payload...)),
		pngChunk("IEND", nil),
	)
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "misleading.json")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if got.Container != PNG {
		t.Fatalf("container = %q, want PNG", got.Container)
	}
	if len(got.Payloads) != 1 {
		t.Fatalf("payload count = %d, want 1", len(got.Payloads))
	}
	if got.Payloads[0].Location.Name != "ccv3" {
		t.Errorf("payload location = %q, want ccv3", got.Payloads[0].Location.Name)
	}
	if got.Payloads[0].Location.Offset <= 512 {
		t.Errorf("payload offset = %d, want it beyond the old head peek", got.Payloads[0].Location.Offset)
	}
	if spec, ok := got.Payloads[0].String("spec"); !ok || spec != "chara_card_v2" {
		t.Errorf("spec = %q, %v; want chara_card_v2, true", spec, ok)
	}
	for _, read := range store.reads {
		if read.offset == 0 && read.length == int64(len(file)) {
			t.Fatal("probe loaded the whole blob instead of using bounded range reads")
		}
	}
}

func pngFile(chunks ...[]byte) []byte {
	file := append([]byte(nil), []byte("\x89PNG\r\n\x1a\n")...)
	for _, chunk := range chunks {
		file = append(file, chunk...)
	}
	return file
}

func pngChunk(workType string, data []byte) []byte {
	var chunk bytes.Buffer
	_ = binary.Write(&chunk, binary.BigEndian, uint32(len(data)))
	chunk.WriteString(workType)
	chunk.Write(data)
	_ = binary.Write(&chunk, binary.BigEndian, crc32.ChecksumIEEE(append([]byte(workType), data...)))
	return chunk.Bytes()
}

func zipFile(t *testing.T, name, body string) []byte {
	t.Helper()
	return zipEntries(t, zipEntry{name: name, body: body, method: zip.Store})
}

type zipEntry struct {
	name   string
	body   string
	method uint16
}

func zipEntries(t *testing.T, entries ...zipEntry) []byte {
	t.Helper()
	var file bytes.Buffer
	archive := zip.NewWriter(&file)
	for _, value := range entries {
		entry, err := archive.CreateHeader(&zip.FileHeader{Name: value.name, Method: value.method})
		if err != nil {
			t.Fatalf("create ZIP entry: %v", err)
		}
		if _, err := io.WriteString(entry, value.body); err != nil {
			t.Fatalf("write ZIP entry: %v", err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close ZIP: %v", err)
	}
	return file.Bytes()
}

func TestInspectOffersARasterFileAsItsOwnImage(t *testing.T) {
	t.Parallel()
	file := pngFile(
		pngChunk("IHDR", make([]byte, 13)),
		pngChunk("tEXt", append([]byte("chara\x00"), []byte(`{"spec":"chara_card_v2"}`)...)),
		pngChunk("IEND", nil),
	)
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "card.png")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got.Images) != 1 {
		t.Fatalf("image count = %d, want the file itself", len(got.Images))
	}
	if got.Images[0].Location.Container != PNG || got.Images[0].Location.Name != "" {
		t.Fatalf("image location = %+v, want the whole PNG", got.Images[0].Location)
	}

	opened, err := got.OpenImage(context.Background(), got.Images[0].ID)
	if err != nil {
		t.Fatalf("OpenImage: %v", err)
	}
	defer opened.Close()
	read, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if !bytes.Equal(read, file) {
		t.Fatal("the extracted image is not the source bytes")
	}
}

func TestInspectListsArchivedImagesAndLeavesOtherEntriesAlone(t *testing.T) {
	t.Parallel()
	picture := pngFile(pngChunk("IHDR", make([]byte, 13)), pngChunk("IEND", nil))
	file := zipEntries(t,
		zipEntry{name: "card.json", body: `{"spec":"chara_card_v3"}`, method: zip.Store},
		zipEntry{name: "assets/icon/main.png", body: string(picture), method: zip.Store},
		zipEntry{name: "assets/notes.txt", body: "not a picture", method: zip.Store},
	)
	store := &recordingStore{data: file}

	got, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "card.charx")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got.Images) != 1 {
		t.Fatalf("image count = %d, want only the PNG entry", len(got.Images))
	}
	if got.Images[0].Location.Name != "assets/icon/main.png" {
		t.Fatalf("image location = %+v, want the icon entry", got.Images[0].Location)
	}

	opened, err := got.OpenImage(context.Background(), got.Images[0].ID)
	if err != nil {
		t.Fatalf("OpenImage: %v", err)
	}
	defer opened.Close()
	read, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read image: %v", err)
	}
	if !bytes.Equal(read, picture) {
		t.Fatal("the extracted archive entry is not the stored picture")
	}
}

func TestOpenImageRefusesAnIDTheProbeNeverIssued(t *testing.T) {
	t.Parallel()
	file := []byte(`{"spec":"chara_card_v3"}`)
	got, err := Inspect(context.Background(), &recordingStore{data: file}, uuid.New(), int64(len(file)), "card.json")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if len(got.Images) != 0 {
		t.Fatalf("image count = %d, want none in a bare JSON card", len(got.Images))
	}
	if _, err := got.OpenImage(context.Background(), 0); err == nil {
		t.Fatal("OpenImage accepted an image the probe never located")
	}
}
