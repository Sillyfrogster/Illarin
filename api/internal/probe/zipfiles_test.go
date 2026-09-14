package probe

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestOpenZIPFilesReadsManyEntriesWithoutRereadingTheArchive(t *testing.T) {
	entries := make([]zipEntry, 0, 120)
	for index := range 120 {
		entries = append(entries, zipEntry{
			name:   fmt.Sprintf("src/module%03d.ts", index),
			body:   strings.Repeat(fmt.Sprintf("export const value%d = %d\n", index, index), 30),
			method: zip.Deflate,
		})
	}
	file := zipEntries(t, entries...)
	store := &recordingStore{data: file}
	inspected, err := Inspect(context.Background(), store, uuid.New(), int64(len(file)), "extension.zip")
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	store.reads = nil

	files, err := inspected.OpenZIPFiles(context.Background())
	if err != nil {
		t.Fatalf("OpenZIPFiles: %v", err)
	}
	for _, entry := range entries {
		opened, err := files.Open(entry.name)
		if err != nil {
			t.Fatalf("open %s: %v", entry.name, err)
		}
		body, err := io.ReadAll(opened)
		_ = opened.Close()
		if err != nil || string(body) != entry.body {
			t.Fatalf("%s read %d bytes and %v, want its %d bytes", entry.name, len(body), err, len(entry.body))
		}
	}
	if len(store.reads) >= len(entries) {
		t.Errorf("reading %d entries took %d reads of the store, want fewer than one each", len(entries), len(store.reads))
	}
	for _, read := range store.reads {
		if read.length > maxRangeRead {
			t.Fatalf("range read length = %d, want at most %d", read.length, maxRangeRead)
		}
	}
	if _, err := files.Open("src/missing.ts"); !errors.Is(err, ErrZIPEntryUnavailable) {
		t.Errorf("opening a missing entry = %v, want ErrZIPEntryUnavailable", err)
	}
}
