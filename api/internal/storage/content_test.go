package storage

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
)

func newTestStore(t *testing.T) Store {
	t.Helper()
	store, err := NewStore(testdb.Connect(t), t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return store
}

func TestPutComputesTheBlobDigestAndSize(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)

	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("abc")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	const wantDigest = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if hex.EncodeToString(stored.Digest[:]) != wantDigest {
		t.Errorf("digest = %x, want %s", stored.Digest, wantDigest)
	}
	if stored.ByteSize != 3 {
		t.Errorf("byte size = %d, want 3", stored.ByteSize)
	}

}

func TestPutMakesTheBlobReadableByTheByteServer(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	store, err := NewStore(testdb.Connect(t), root)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("served by nginx")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	encoded := hex.EncodeToString(stored.Digest[:])
	info, err := os.Stat(filepath.Join(root, "blobs", encoded[:2], encoded))
	if err != nil {
		t.Fatalf("stat blob: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Fatalf("blob mode = %o, want 644", got)
	}
}

func TestConcurrentIdenticalWritesConvergeOnOneBlob(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)

	const writers = 8
	start := make(chan struct{})
	results := make(chan StoredBlob, writers)
	errors := make(chan error, writers)
	var ready sync.WaitGroup
	ready.Add(writers)
	for range writers {
		go func() {
			ready.Done()
			<-start
			stored, err := store.Put(context.Background(), bytes.NewReader([]byte("same bytes")))
			results <- stored
			errors <- err
		}()
	}
	ready.Wait()
	close(start)

	var first StoredBlob
	for range writers {
		if err := <-errors; err != nil {
			t.Fatalf("Put: %v", err)
		}
		stored := <-results
		if first.ID == [16]byte{} {
			first = stored
		} else if stored.ID != first.ID {
			t.Errorf("identical write returned blob %s, want %s", stored.ID, first.ID)
		}
	}

}

func TestReadRangeReturnsOnlyTheRequestedBytes(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("0123456789")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	part, err := store.ReadRange(context.Background(), stored.ID, 3, 4)
	if err != nil {
		t.Fatalf("ReadRange: %v", err)
	}
	defer part.Close()

	got, err := io.ReadAll(part)
	if err != nil {
		t.Fatalf("read range: %v", err)
	}
	if string(got) != "3456" {
		t.Errorf("range = %q, want %q", got, "3456")
	}
}

func TestReadRangeRejectsEmptyAndOutOfBoundsRequests(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("0123456789")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	for _, test := range []struct {
		name           string
		offset, length int64
	}{
		{name: "zero length", offset: 3, length: 0},
		{name: "past end", offset: 8, length: 3},
		{name: "starts at end", offset: 10, length: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := store.ReadRange(context.Background(), stored.ID, test.offset, test.length)
			if !errors.Is(err, ErrInvalidRange) {
				t.Fatalf("ReadRange error = %v, want ErrInvalidRange", err)
			}
		})
	}
}

func TestOpenReturnsTheWholeBlob(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	want := []byte{0x00, 0xff, 0x50, 0x4e, 0x47, 0x80}
	stored, err := store.Put(context.Background(), bytes.NewReader(want))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	opened, err := store.Open(context.Background(), stored.ID)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer opened.Close()
	got, err := io.ReadAll(opened)
	if err != nil {
		t.Fatalf("read blob: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("blob changed: got %x, want %x", got, want)
	}
}

func TestTheImageCacheIsDisposableWithoutTouchingSourceBlobs(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("source")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	id := ImageSizeID{SourceDigest: stored.Digest, Size: "detail", Version: 2}
	if err := store.PutImageSize(context.Background(), id, []byte("rendered")); err != nil {
		t.Fatalf("PutImageSize: %v", err)
	}

	opened, err := store.OpenImageSize(context.Background(), id)
	if err != nil {
		t.Fatalf("OpenImageSize: %v", err)
	}
	got, err := io.ReadAll(opened)
	opened.Close()
	if err != nil {
		t.Fatalf("read the image size: %v", err)
	}
	if string(got) != "rendered" {
		t.Errorf("image size = %q, want rendered", got)
	}

	if err := store.ClearImageCache(context.Background()); err != nil {
		t.Fatalf("ClearImageCache: %v", err)
	}
	if _, err := store.OpenImageSize(context.Background(), id); err == nil {
		t.Fatal("the image size still exists after clearing the cache")
	}
	source, err := store.Open(context.Background(), stored.ID)
	if err != nil {
		t.Fatalf("Open source after clearing the image cache: %v", err)
	}
	source.Close()
}

func TestImageSizeIdentityIncludesSourceSizeAndVersion(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	first, err := store.Put(context.Background(), bytes.NewReader([]byte("first source")))
	if err != nil {
		t.Fatalf("put first source: %v", err)
	}
	second, err := store.Put(context.Background(), bytes.NewReader([]byte("second source")))
	if err != nil {
		t.Fatalf("put second source: %v", err)
	}

	sizes := []struct {
		id   ImageSizeID
		want string
	}{
		{id: ImageSizeID{SourceDigest: first.Digest, Size: "grid", Version: 1}, want: "first grid v1"},
		{id: ImageSizeID{SourceDigest: first.Digest, Size: "grid", Version: 2}, want: "first grid v2"},
		{id: ImageSizeID{SourceDigest: first.Digest, Size: "detail", Version: 1}, want: "first detail v1"},
		{id: ImageSizeID{SourceDigest: second.Digest, Size: "grid", Version: 1}, want: "second grid v1"},
	}
	for _, rendered := range sizes {
		if err := store.PutImageSize(context.Background(), rendered.id,
			[]byte(rendered.want)); err != nil {
			t.Fatalf("PutSize: %v", err)
		}
	}

	for _, rendered := range sizes {
		opened, err := store.OpenImageSize(context.Background(), rendered.id)
		if err != nil {
			t.Fatalf("OpenSize: %v", err)
		}
		got, err := io.ReadAll(opened)
		opened.Close()
		if err != nil {
			t.Fatalf("read the image size: %v", err)
		}
		if string(got) != rendered.want {
			t.Errorf("image size = %q, want %q", got, rendered.want)
		}
	}
}

func TestAnImageSizeCanBeHandedToTheInternalByteServer(t *testing.T) {
	t.Parallel()
	store := newTestStore(t)
	stored, err := store.Put(context.Background(), bytes.NewReader([]byte("source")))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	id := ImageSizeID{SourceDigest: stored.Digest, Size: "grid", Version: 1}
	if err := store.PutImageSize(context.Background(), id, []byte("rendered")); err != nil {
		t.Fatalf("PutSize: %v", err)
	}

	redirect, err := store.InternalImageSizeRedirect(context.Background(), id)
	if err != nil {
		t.Fatalf("InternalImageSizeRedirect: %v", err)
	}
	if !strings.HasPrefix(redirect, "/_illarin/image-cache/") {
		t.Fatalf("redirect = %q, want internal image cache location", redirect)
	}

	if err := store.ClearImageCache(context.Background()); err != nil {
		t.Fatalf("ClearImageCache: %v", err)
	}
	_, err = store.InternalImageSizeRedirect(context.Background(), id)
	if !errors.Is(err, ErrImageSizeNotFound) {
		t.Fatalf("missing image size error = %v, want ErrImageSizeNotFound", err)
	}
}
