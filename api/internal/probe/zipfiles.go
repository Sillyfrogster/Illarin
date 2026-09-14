package probe

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"sync"
)

// ZIPFiles opens the files of one archive whose directory was read once.
type ZIPFiles struct {
	files map[string]*zip.File
}

// OpenZIPFiles reads the archive's directory once, and the archive in large pieces, so that many small entries open cheaply.
func (i Inspection) OpenZIPFiles(ctx context.Context) (ZIPFiles, error) {
	if i.Container != ZIP {
		return ZIPFiles{}, fmt.Errorf("open the archive: %w", ErrZIPEntryUnavailable)
	}
	reader := &pieceReader{source: &rangeReaderAt{ctx: ctx, store: i.source.store, id: i.source.id, size: i.source.size}}
	archive, err := zip.NewReader(reader, i.source.size)
	if err != nil {
		return ZIPFiles{}, fmt.Errorf("reopen the archive: %w", err)
	}
	files := make(map[string]*zip.File, len(archive.File))
	for _, entry := range archive.File {
		if !entry.FileInfo().IsDir() {
			files[entry.Name] = entry
		}
	}
	return ZIPFiles{files: files}, nil
}

// Open opens the file the archive holds under name.
func (f ZIPFiles) Open(name string) (io.ReadCloser, error) {
	entry, ok := f.files[name]
	if !ok {
		return nil, fmt.Errorf("archive entry %q: %w", name, ErrZIPEntryUnavailable)
	}
	return entry.Open()
}

// pieceReader reads a blob in aligned pieces and keeps the latest, so neighbouring small reads share one range read.
type pieceReader struct {
	source *rangeReaderAt
	mu     sync.Mutex
	offset int64
	piece  []byte
}

func (r *pieceReader) ReadAt(p []byte, offset int64) (int, error) {
	if len(p) >= maxRangeRead {
		return r.source.ReadAt(p, offset)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	read := 0
	for read < len(p) {
		at := offset + int64(read)
		if at >= r.source.size {
			return read, io.EOF
		}
		if r.piece == nil || at < r.offset || at >= r.offset+int64(len(r.piece)) {
			if err := r.load(at); err != nil {
				return read, err
			}
		}
		read += copy(p[read:], r.piece[at-r.offset:])
	}
	return read, nil
}

func (r *pieceReader) load(at int64) error {
	start := at - at%maxRangeRead
	piece := make([]byte, min(maxRangeRead, r.source.size-start))
	if _, err := r.source.ReadAt(piece, start); err != nil {
		return err
	}
	r.offset, r.piece = start, piece
	return nil
}
