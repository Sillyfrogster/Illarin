package format_test

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/format/lorebook"
	"github.com/Sillyfrogster/Illarin/api/internal/format/pack"
	"github.com/Sillyfrogster/Illarin/api/internal/format/preset"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/google/uuid"
)

type corpusStore struct{ data []byte }

func (s corpusStore) ReadRange(_ context.Context, _ uuid.UUID, offset, length int64) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(s.data[offset : offset+length])), nil
}

func TestLocalCorpusRunsThroughEveryModule(t *testing.T) {
	t.Parallel()
	registry := format.NewRegistry()
	for _, module := range slices.Concat(character.Modules(), lorebook.Modules(), preset.Modules(), theme.Modules(), pack.Modules()) {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %q: %v", module.ID(), err)
		}
	}
	if err := registry.ValidateDeclarations(); err != nil {
		t.Fatalf("module declarations: %v", err)
	}

	root := os.Getenv("ILLARIN_LOCAL_CORPUS")
	if root == "" {
		t.Skip("local probe corpus is not configured")
	}
	root = filepath.Clean(root)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		t.Skip("local probe corpus is not present")
	} else if err != nil {
		t.Fatal("read local probe corpus")
	}

	checked, matched := 0, 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := format.Inspect(
			context.Background(), corpusStore{data: data}, uuid.New(), int64(len(data)), "fixture.bin",
		)
		if err != nil {
			return err
		}
		resolution, resolved, err := registry.Resolve(file)
		if err != nil {
			return err
		}
		checked++
		if !resolved {
			return nil
		}
		matched++
		parsed, err := resolution.Module.Parse(context.Background(), file, resolution.Match)
		if err != nil {
			return err
		}
		declared := resolution.Module.Declaration()
		if parsed.Type != declared.Type || parsed.Format != resolution.Module.ID() {
			t.Errorf("%s parsed as type %q format %q", entry.Name(), parsed.Type, parsed.Format)
		}
		for _, sidecar := range unreadArchiveEntries(file) {
			if !slices.Contains(preservedData(parsed.Remainder), sidecar) {
				t.Errorf("%s lost the archived %s", entry.Name(), sidecar)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("local probe corpus failed: %v", err)
	}
	if checked == 0 {
		t.Fatal("local probe corpus is empty")
	}
	if matched == 0 {
		t.Fatal("no fixture in the local probe corpus resolved to a module")
	}
}

// unreadArchiveEntries names the archived files a card carries and Illarin reads nothing from.
func unreadArchiveEntries(file format.Inspection) []string {
	if file.Container != format.ZIP {
		return nil
	}
	pictures := make(map[string]bool, len(file.Images))
	for _, image := range file.Images {
		if image.Location.Container == format.ZIP {
			pictures[image.Location.Name] = true
		}
	}
	unread := make([]string, 0)
	for _, entry := range file.ZIPEntries {
		if entry.Directory || entry.Name == "card.json" || pictures[entry.Name] {
			continue
		}
		unread = append(unread, character.MemberNamespace+entry.Name)
	}
	return unread
}

func preservedData(rows []format.Remainder) []string {
	kept := make([]string, 0, len(rows))
	for _, row := range rows {
		kept = append(kept, row.Namespace)
	}
	return kept
}
