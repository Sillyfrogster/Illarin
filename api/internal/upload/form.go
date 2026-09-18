package upload

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func readMetadata(parts *multipart.Reader) (CreateWorkRequest, error) {
	part, err := api.NextPart(parts, api.MetadataPart)
	if err != nil {
		return CreateWorkRequest{}, err
	}

	var metadata CreateWorkRequest
	if err := api.DecodeOneJSON(io.LimitReader(part, 1<<20), &metadata); err != nil {
		return CreateWorkRequest{}, api.FormRefusal{
			Reason: "the " + api.MetadataPart + " part is not valid JSON",
			Cause:  err,
		}
	}
	return metadata, nil
}

func ingestInput(
	metadata CreateWorkRequest,
	filename string,
	file io.Reader,
	ownerID uuid.UUID,
) IngestInput {
	in := IngestInput{
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
	if metadata.Visibility != nil {
		in.Visibility = work.Visibility(*metadata.Visibility)
	}
	return in
}

// RefuseFile answers a refused upload with the reason a person can act on
func RefuseFile(c *gin.Context, err error, maxUploadBytes int64) {
	if errors.Is(err, work.ErrStorageCap) {
		api.Refuse(c, http.StatusRequestEntityTooLarge, "Your account does not have enough storage left for this file.")
		return
	}
	if errors.Is(err, storage.ErrInsufficientSpace) {
		api.Refuse(c, http.StatusServiceUnavailable, "Uploads are temporarily unavailable because storage is low.")
		return
	}
	if errors.Is(err, format.ErrInvariant) {
		api.Refuse(c, http.StatusInternalServerError, "could not create the work")
		return
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		api.Refuse(c, http.StatusRequestEntityTooLarge, fmt.Sprintf(
			"That file is larger than the %s upload limit.", readableSize(maxUploadBytes),
		))
		return
	}

	var refused api.FormRefusal
	if errors.As(err, &refused) {
		api.Refuse(c, http.StatusBadRequest, refused.Error())
		return
	}
	api.Refuse(c, http.StatusBadRequest, "could not create the work")
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
