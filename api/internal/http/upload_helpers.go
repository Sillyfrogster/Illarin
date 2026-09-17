package http

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	metadataPart = "metadata"
	filePart     = "file"
)

func readMetadata(parts *multipart.Reader) (CreateAssetRequest, error) {
	part, err := nextPart(parts, metadataPart)
	if err != nil {
		return CreateAssetRequest{}, err
	}

	var metadata CreateAssetRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return CreateAssetRequest{}, refusal{
			reason: "the " + metadataPart + " part is not valid JSON",
			cause:  err,
		}
	}
	return metadata, nil
}

func readMediaMetadata(parts *multipart.Reader) (AddMediaRequest, error) {
	part, err := nextPart(parts, metadataPart)
	if err != nil {
		return AddMediaRequest{}, err
	}
	var metadata AddMediaRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return AddMediaRequest{}, refusal{
			reason: "the " + metadataPart + " part is not valid JSON",
			cause:  err,
		}
	}
	return metadata, nil
}

func readPostMediaMetadata(parts *multipart.Reader) (AddPostMediaRequest, error) {
	part, err := nextPart(parts, metadataPart)
	if err != nil {
		return AddPostMediaRequest{}, err
	}
	var metadata AddPostMediaRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return AddPostMediaRequest{}, refusal{
			reason: "the " + metadataPart + " part is not valid JSON",
			cause:  err,
		}
	}
	return metadata, nil
}

func nextPart(parts *multipart.Reader, name string) (*multipart.Part, error) {
	part, err := parts.NextPart()
	if errors.Is(err, io.EOF) {
		return nil, refusal{reason: "the " + name + " part is missing", cause: err}
	}
	if err != nil {
		return nil, refusal{reason: "the form data could not be read", cause: err}
	}
	if part.FormName() != name {
		return nil, refusal{
			reason: fmt.Sprintf("expected the %s part here, found %q", name, part.FormName()),
			cause:  nil,
		}
	}
	return part, nil
}

func ingestInput(
	metadata CreateAssetRequest,
	filename string,
	file io.Reader,
	ownerID uuid.UUID,
) asset.IngestInput {
	in := asset.IngestInput{
		OwnerID:  ownerID,
		Filename: filename,
		File:     file,
	}
	if metadata.Name != nil {
		in.Name = metadata.Name
	}
	if metadata.Blurb != nil {
		in.Blurb = metadata.Blurb
	}
	if metadata.Tags != nil {
		in.Tags = metadata.Tags
	}
	if metadata.IsNsfw != nil {
		in.IsNSFW = metadata.IsNsfw
	}
	if metadata.Discovery != nil {
		in.Discovery = asset.Discovery(*metadata.Discovery)
	}
	return in
}

type refusal struct {
	reason string
	cause  error
}

func (r refusal) Error() string { return r.reason }
func (r refusal) Unwrap() error { return r.cause }

func (h *Handlers) refuse(c *gin.Context, err error) {
	if errors.Is(err, asset.ErrStorageCap) {
		api.Refuse(c, http.StatusRequestEntityTooLarge, "Your account does not have enough storage left for this file.")
		return
	}
	if errors.Is(err, storage.ErrInsufficientSpace) {
		api.Refuse(c, http.StatusServiceUnavailable, "Uploads are temporarily unavailable because storage is low.")
		return
	}
	if errors.Is(err, format.ErrInvariant) {
		api.Refuse(c, http.StatusInternalServerError, "could not create the asset")
		return
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		api.Refuse(c, http.StatusRequestEntityTooLarge, fmt.Sprintf(
			"That file is larger than the %s upload limit.", readableSize(h.maxUploadBytes),
		))
		return
	}

	var refused refusal
	if errors.As(err, &refused) {
		api.Refuse(c, http.StatusBadRequest, refused.Error())
		return
	}
	api.Refuse(c, http.StatusBadRequest, "could not create the asset")
}

func readableSize(bytes int64) string {
	units := []struct {
		suffix string
		scale  int64
	}{{suffix: "GB", scale: 1 << 30}, {suffix: "MB", scale: 1 << 20}, {suffix: "KB", scale: 1 << 10}}
	for _, unit := range units {
		if bytes < unit.scale {
			continue
		}
		return strings.TrimSuffix(
			strconv.FormatFloat(float64(bytes)/float64(unit.scale), 'f', 1, 64), ".0",
		) + " " + unit.suffix
	}
	return fmt.Sprintf("%d bytes", bytes)
}
