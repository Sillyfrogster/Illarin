package media

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type Renderer interface {
	Prepare(context.Context, io.Reader) (Prepared, error)
	Render(context.Context, io.Reader, string) (Derivative, error)
	ComposeLinkCard(context.Context, io.Reader, string) (Derivative, error)
	DerivativeType() string
}

type Library struct {
	store    storage.Store
	renderer Renderer
	slots    chan struct{}
	flight   singleflight.Group
}

func NewLibrary(store storage.Store, renderer Renderer, workers int) *Library {
	if workers < 1 {
		workers = 1
	}
	return &Library{store: store, renderer: renderer, slots: make(chan struct{}, workers)}
}

func (l *Library) DerivativeType() string { return l.renderer.DerivativeType() }

func (l *Library) Accept(ctx context.Context, r io.Reader) (storage.StoredBlob, Prepared, error) {
	stored, err := l.store.Put(ctx, r)
	if err != nil {
		return storage.StoredBlob{}, Prepared{}, fmt.Errorf("store media: %w", err)
	}
	prepared, err := l.Prepare(ctx, stored)
	if err != nil {
		return storage.StoredBlob{}, Prepared{}, err
	}
	return stored, prepared, nil
}

func (l *Library) Prepare(ctx context.Context, stored storage.StoredBlob) (Prepared, error) {
	source, err := l.store.Open(ctx, stored.ID)
	if err != nil {
		return Prepared{}, fmt.Errorf("open stored media: %w", err)
	}
	release, err := l.acquireSlot(ctx)
	if err != nil {
		source.Close()
		return Prepared{}, err
	}
	prepared, prepareErr := l.renderer.Prepare(ctx, source)
	release()
	closeErr := source.Close()
	if prepareErr != nil {
		return Prepared{}, prepareErr
	}
	if closeErr != nil {
		return Prepared{}, fmt.Errorf("close stored media: %w", closeErr)
	}
	for _, derivative := range prepared.Derivatives {
		id := storage.DerivativeID{
			SourceDigest: stored.Digest,
			Variant:      derivative.Variant,
			Version:      DerivativeVersion,
		}
		if err := l.store.PutDerivative(ctx, id, derivative.Bytes); err != nil {
			if errors.Is(err, storage.ErrInsufficientSpace) {
				break
			}
			return Prepared{}, fmt.Errorf("store %s media variant: %w", derivative.Variant, err)
		}
	}
	return prepared, nil
}

func (l *Library) Serve(
	ctx context.Context,
	blobID uuid.UUID,
	digest [sha256.Size]byte,
	variant string,
	version uint32,
) (string, error) {
	id := storage.DerivativeID{SourceDigest: digest, Variant: variant, Version: version}
	redirect, err := l.store.InternalDerivativeRedirect(ctx, id)
	if errors.Is(err, storage.ErrDerivativeNotFound) {
		job := l.flight.DoChan(fmt.Sprintf("%x/%s/%d", digest, variant, version), func() (any, error) {
			return nil, l.render(ctx, blobID, id)
		})
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case result := <-job:
			if result.Err != nil {
				return "", result.Err
			}
		}
		redirect, err = l.store.InternalDerivativeRedirect(ctx, id)
	}
	if err != nil {
		return "", fmt.Errorf("resolve media variant: %w", err)
	}
	return redirect, nil
}

func (l *Library) render(ctx context.Context, blobID uuid.UUID, id storage.DerivativeID) error {
	release, err := l.acquireSlot(ctx)
	if err != nil {
		return err
	}
	defer release()
	source, err := l.store.Open(ctx, blobID)
	if err != nil {
		return fmt.Errorf("open media for regeneration: %w", err)
	}
	var derivative Derivative
	var renderErr error
	if _, composed := LinkCardByName(id.Variant); composed {
		derivative, renderErr = l.renderer.ComposeLinkCard(ctx, source, id.Variant)
	} else {
		derivative, renderErr = l.renderer.Render(ctx, source, id.Variant)
	}
	closeErr := source.Close()
	if renderErr != nil {
		return fmt.Errorf("regenerate media variant: %w", renderErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close media after regeneration: %w", closeErr)
	}
	if err := l.store.PutDerivative(ctx, id, derivative.Bytes); err != nil {
		return fmt.Errorf("store regenerated media variant: %w", err)
	}
	return nil
}

func (l *Library) acquireSlot(ctx context.Context) (func(), error) {
	select {
	case l.slots <- struct{}{}:
		return func() { <-l.slots }, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
