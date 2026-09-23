package extension

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/Sillyfrogster/Illarin/api/internal/format"
)

const (
	Type = "extension"

	// MaxArchiveBytes is the largest extension archive Illarin accepts.
	MaxArchiveBytes = 32 << 20

	// MaxArchiveFiles is how many files an extension archive may hold, since built extensions carry many chunks.
	MaxArchiveFiles = 4096

	maxReadmeBytes = 1 << 20

	spindleManifest     = "spindle.json"
	sillyTavernManifest = "manifest.json"
)

// checkArchive refuses an archive over the size limit or one that holds both apps' manifests.
func checkArchive(file format.Inspection) error {
	if file.ByteSize() > MaxArchiveBytes {
		return format.LimitExceeded(fmt.Errorf(
			"an extension archive may be up to %d MB, and this one is %.1f MB",
			MaxArchiveBytes>>20, float64(file.ByteSize())/(1<<20),
		))
	}
	if hasEntry(file, spindleManifest) && hasEntry(file, sillyTavernManifest) {
		return refuse(
			"the archive has both %s and %s at its root, so it is unclear which app it is for",
			spindleManifest, sillyTavernManifest,
		)
	}
	return nil
}

// archiveManifest picks the manifest an archive is read by, preferring one where the app reads it over one found elsewhere.
func archiveManifest(file format.Inspection) (string, bool) {
	for _, manifest := range []string{spindleManifest, sillyTavernManifest} {
		if hasEntry(file, manifest) {
			return manifest, true
		}
	}
	for _, manifest := range []string{spindleManifest, sillyTavernManifest} {
		if _, found := misplacedManifest(file, manifest); found {
			return manifest, true
		}
	}
	return "", false
}

// matchArchive matches an archive for the module whose manifest it holds, even a misplaced one, so the refusal can say where it is.
func matchArchive(file format.Inspection, declaration format.Declaration, manifest string) (format.Match, bool) {
	if chosen, ok := archiveManifest(file); !ok || chosen != manifest {
		return format.Match{}, false
	}
	if match, ok := format.MatchByDeclaration(file, declaration); ok {
		return match, true
	}
	return format.WholeFileCompatibilityMatch(file), true
}

func misplacedManifest(file format.Inspection, manifest string) (string, bool) {
	for _, entry := range file.ZIPEntries {
		if !entry.Directory && path.Base(entry.Name) == manifest && !strings.HasPrefix(entry.Name, "__MACOSX/") {
			return entry.Name, true
		}
	}
	return "", false
}

func refuseMisplaced(file format.Inspection, manifest string) error {
	found, _ := misplacedManifest(file, manifest)
	return refuse(
		"%s is at %s. Put it at the top of the zip, or in the one folder that holds everything else",
		manifest, found,
	)
}

var readmeNames = []string{"readme.md", "readme.markdown", "readme"}

// readmeFolders are where GitHub looks for the README it shows, in the order it looks.
var readmeFolders = []string{".github/", "", "docs/"}

// readReadme reads the README beside the manifest, finding none where there is no README it can read as text.
func readReadme(ctx context.Context, file format.Inspection) (*format.Readme, error) {
	entry, ok := readmeEntry(file)
	if !ok || entry.UncompressedSize > maxReadmeBytes {
		return nil, nil
	}
	opened, err := file.OpenZIPEntry(ctx, entry.Name)
	if err != nil {
		return nil, format.InternalFailure(fmt.Errorf("open %s: %w", entry.Name, err))
	}
	defer opened.Close()
	body, err := io.ReadAll(io.LimitReader(opened, maxReadmeBytes+1))
	if errors.Is(err, format.ErrRangeRead) {
		return nil, format.InternalFailure(fmt.Errorf("read %s: %w", entry.Name, err))
	}
	if err != nil {
		return nil, refuse("%s cannot be read from the archive: %v", entry.Name, err)
	}
	if len(body) > maxReadmeBytes || !utf8.Valid(body) {
		return nil, nil
	}
	folder := path.Dir(strings.TrimPrefix(entry.Name, file.ArchiveBase)) + "/"
	if folder == "./" {
		folder = ""
	}
	return &format.Readme{Text: string(body), Root: file.ArchiveBase, Folder: folder}, nil
}

func readmeEntry(file format.Inspection) (format.ZIPEntry, bool) {
	for _, folder := range readmeFolders {
		for _, wanted := range readmeNames {
			for _, entry := range file.ZIPEntries {
				name, inside := strings.CutPrefix(entry.Name, file.ArchiveBase)
				if inside && !entry.Directory && strings.EqualFold(name, folder+wanted) {
					return entry, true
				}
			}
		}
	}
	return format.ZIPEntry{}, false
}

func hasEntry(file format.Inspection, name string) bool {
	return slices.ContainsFunc(file.ZIPEntries, func(entry format.ZIPEntry) bool {
		return !entry.Directory && entry.Name == file.ArchiveBase+name
	})
}

func insideArchive(name string) bool {
	return name != ".." && !strings.HasPrefix(name, "../") && !path.IsAbs(name)
}

func refuse(message string, args ...any) error {
	return format.MalformedInput(fmt.Errorf(message, args...))
}

func Modules() []format.Reader { return []format.Reader{Spindle{}, SillyTavern{}} }
