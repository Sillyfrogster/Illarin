package migration

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"time"
)

type FileBackup struct {
	path string
}

type backupEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
	Body    io.Reader
}

func OpenFileBackup(archive string) (*FileBackup, error) {
	if _, err := os.Stat(archive); err != nil {
		return nil, fmt.Errorf("open the v1 file backup: %w", err)
	}
	return &FileBackup{path: archive}, nil
}

func (backup *FileBackup) each(visit func(backupEntry) error) error {
	file, err := os.Open(backup.path)
	if err != nil {
		return fmt.Errorf("open the v1 file backup: %w", err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("read the v1 file backup as gzip: %w", err)
	}
	defer compressed.Close()

	reader := tar.NewReader(compressed)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read the v1 file backup: %w", err)
		}
		if !header.FileInfo().Mode().IsRegular() {
			continue
		}
		if err := visit(backupEntry{
			Name: header.Name, Size: header.Size, ModTime: header.ModTime, Body: reader,
		}); err != nil {
			return err
		}
	}
}

type backupIndex struct {
	paths map[string]int
	stems map[string]int
}

func newBackupIndex() *backupIndex {
	return &backupIndex{paths: make(map[string]int), stems: make(map[string]int)}
}

func (index *backupIndex) want(stored string, at int) {
	if cleaned := cleanBackupPath(stored); cleaned != "" {
		index.paths[cleaned] = at
	}
}

func (index *backupIndex) wantStem(identifier string, at int) {
	if identifier != "" {
		index.stems[identifier] = at
	}
}

func (index *backupIndex) find(name string) (int, bool) {
	cleaned := cleanBackupPath(name)
	if cleaned == "" {
		return 0, false
	}
	parts := strings.Split(cleaned, "/")
	for i := range parts {
		if at, found := index.paths[strings.Join(parts[i:], "/")]; found {
			return at, true
		}
	}
	base := path.Base(cleaned)
	if at, found := index.stems[strings.TrimSuffix(base, path.Ext(base))]; found {
		return at, true
	}
	return 0, false
}

func cleanBackupPath(value string) string {
	cleaned := path.Clean(strings.TrimPrefix(strings.ReplaceAll(value, "\\", "/"), "./"))
	if cleaned == "." || cleaned == "/" {
		return ""
	}
	return strings.TrimPrefix(cleaned, "/")
}
