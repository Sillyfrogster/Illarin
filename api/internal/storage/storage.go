package storage

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/google/uuid"
)

var (
	ErrBlobNotFound      = errors.New("blob not found")
	ErrTombstoned        = errors.New("blob digest is tombstoned")
	ErrInvalidRange      = errors.New("invalid blob range")
	ErrImageSizeNotFound = errors.New("rendered not found")
	ErrInsufficientSpace = errors.New("storage reserve would be crossed")
)

type Capacity struct {
	FreeSpaceReserveBytes int64
	MaximumBlobWriteBytes int64
}

type StoredBlob struct {
	ID       uuid.UUID
	Digest   [sha256.Size]byte
	ByteSize int64
}

type ImageSizeID struct {
	SourceDigest [sha256.Size]byte
	Size         string
	Version      uint32
}

type Store interface {
	Put(ctx context.Context, r io.Reader) (StoredBlob, error)
	RecordOrphans(ctx context.Context) (int, error)
	Open(ctx context.Context, id uuid.UUID) (io.ReadCloser, error)
	ReadRange(ctx context.Context, id uuid.UUID, offset, length int64) (io.ReadCloser, error)
	InternalRedirect(ctx context.Context, id uuid.UUID) (string, error)
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteImageSizes(ctx context.Context, digest [sha256.Size]byte) error
	PutImageSize(ctx context.Context, id ImageSizeID, body []byte) error
	OpenImageSize(ctx context.Context, id ImageSizeID) (io.ReadCloser, error)
	InternalImageSizeRedirect(ctx context.Context, id ImageSizeID) (string, error)
	ClearImageCache(ctx context.Context) error
}
