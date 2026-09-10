package http

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

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

// readMetadata reads the catalog fields, which come first because the file is
// stored as it arrives and the fields have to be known by then.
func readMetadata(parts *multipart.Reader) (CreateAssetRequest, error) {
	part, err := nextPart(parts, metadataPart)
	if err != nil {
		return CreateAssetRequest{}, err
	}

	var metadata CreateAssetRequest
	if err := decodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
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
	if err := decodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
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
	if err := decodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return AddPostMediaRequest{}, refusal{
			reason: "the " + metadataPart + " part is not valid JSON",
			cause:  err,
		}
	}
	return metadata, nil
}

func decodeOneJSON(reader io.Reader, destination any) error {
	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("more than one JSON value")
		}
		return err
	}
	return nil
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

// refusal is a reason a request cannot be accepted, worded for whoever sent
// it. The cause stays attached so the ceiling can still be recognised through
// the layers that wrapped it.
type refusal struct {
	reason string
	cause  error
}

func (r refusal) Error() string { return r.reason }
func (r refusal) Unwrap() error { return r.cause }

func (h *Handlers) refuse(c *gin.Context, err error) {
	if errors.Is(err, asset.ErrStorageCap) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": "Your account does not have enough storage left for this file.",
		})
		return
	}
	if errors.Is(err, storage.ErrInsufficientSpace) {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Uploads are temporarily unavailable because storage is low.",
		})
		return
	}
	if errors.Is(err, format.ErrInvariant) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not create the asset"})
		return
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{
			"error": fmt.Sprintf(
				"That file is larger than the %s upload limit.", readableSize(h.maxUploadBytes),
			),
		})
		return
	}

	var refused refusal
	if errors.As(err, &refused) {
		c.JSON(http.StatusBadRequest, gin.H{"error": refused.Error()})
		return
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "could not create the asset"})
}

// readableSize writes a byte ceiling the way a person says it, so a refusal names a size a creator can compare their own file against.
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
