package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) WithholdAsset(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	admin, ok := api.Admin(c, "manage withholds")
	if !ok {
		return
	}
	var request WithholdAssetRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Give a reason for withholding the asset.")
		return
	}
	err := h.assets.Withhold(c.Request.Context(), id, admin.ID, request.Reason)
	switch {
	case errors.Is(err, asset.ErrInvalidWithholdReason):
		api.Refuse(c, http.StatusBadRequest, "Give a reason for withholding the asset.")
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such asset")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not withhold the asset.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) ClearAssetWithhold(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	if _, ok := api.Admin(c, "manage withholds"); !ok {
		return
	}
	err := h.assets.ClearWithhold(c.Request.Context(), id)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such asset")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not clear the withhold.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) CreateAsset(c *gin.Context) {
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	if c.ContentType() == "application/json" {
		h.startAssetFromNothing(c, owner)
		return
	}
	h.acceptUpload(c, owner)
}

func (h *Handlers) acceptUpload(c *gin.Context, owner api.Account) {
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refuse(c, refusal{
			reason: "send the asset as form data, with a metadata part and a file part",
			cause:  err,
		})
		return
	}

	metadata, err := readMetadata(parts)
	if err != nil {
		h.refuse(c, err)
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refuse(c, err)
		return
	}
	if !metadata.Confirmed {
		h.refuse(c, refusal{reason: "confirm the catalog details before uploading"})
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()

	operation, err := h.assets.AcceptIngest(
		c.Request.Context(), ingestInput(metadata, file.FileName(), limitedFile, owner.ID),
	)
	if errors.Is(err, storage.ErrTombstoned) {
		api.Refuse(c, http.StatusUnprocessableEntity, "This file cannot be accepted.")
		return
	}
	if err != nil {
		h.refuse(c, err)
		return
	}

	location := "/v1/ingests/" + operation.ID.String()
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, toAPIIngest(operation))
}

func (h *Handlers) AddAssetRevision(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}

	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refuse(c, refusal{
			reason: "send the revision as form data, with a file part",
			cause:  err,
		})
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refuse(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()

	candidate := &asset.Candidate{Version: version}
	operation, err := h.assets.AcceptRevision(c.Request.Context(), asset.RevisionInput{
		OwnerID:  owner.ID,
		AssetID:  id,
		Filename: file.FileName(),
		File:     limitedFile,
	}, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such asset")
		return
	case errors.Is(err, storage.ErrTombstoned):
		api.Refuse(c, http.StatusUnprocessableEntity, "This file cannot be accepted.")
		return
	case err != nil:
		h.refuse(c, err)
		return
	}

	location := "/v1/ingests/" + operation.ID.String()
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, toAPIIngest(operation))
}

func (h *Handlers) GetAssetReplacement(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	operation, err := h.assets.ReviewedReplacement(c.Request.Context(), owner.ID, id)
	if errors.Is(err, asset.ErrIngestNotFound) {
		c.JSON(http.StatusOK, nil)
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not load the replacement file. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIIngest(operation))
}

func (h *Handlers) AcceptAssetRevision(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	operationID, ok := api.PathID(c, "operationId")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	var body ReplacementAcceptance
	if err := c.ShouldBindJSON(&body); err != nil {
		h.refuse(c, refusal{reason: "send the replacement decisions as JSON", cause: err})
		return
	}
	decisions := make(map[string]string, len(body.Unrepresentable))
	for role, decision := range body.Unrepresentable {
		decisions[role] = string(decision)
	}
	candidate := &asset.Candidate{Version: version}
	operation, err := h.assets.AcceptReplacement(c.Request.Context(), owner.ID, id, operationID, candidate, decisions, body.ExposeProtected != nil && *body.ExposeProtected)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	var exposure asset.ExposureRefusal
	if errors.As(err, &exposure) {
		c.JSON(http.StatusConflict, edit.SealedExposureRefusal{
			Error:   "This replacement removes prompt protection. Confirm that text in this asset and its recorded versions may become public immediately.",
			Code:    edit.SealedExposureRefusalCodeSealedExposure,
			Prompts: exposure.Prompts,
		})
		return
	}
	if errors.Is(err, asset.ErrIngestNotFound) || errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no reviewed replacement")
		return
	}
	if errors.Is(err, asset.ErrReplacementDecision) {
		api.Refuse(c, http.StatusUnprocessableEntity, err.Error())
		return
	}
	if _, why, classified := format.Explain(err); classified {
		api.Refuse(c, http.StatusUnprocessableEntity, why)
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not apply the replacement file. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIIngest(operation))
}

func (h *Handlers) CancelAssetRevision(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	operationID, ok := api.PathID(c, "operationId")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	err := h.assets.CancelReplacement(c.Request.Context(), owner.ID, id, operationID)
	if errors.Is(err, asset.ErrIngestNotFound) {
		api.Refuse(c, http.StatusNotFound, "no reviewed replacement")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not discard the replacement file. Try again.")
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handlers) AddMedia(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.WorkingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	parts, err := c.Request.MultipartReader()
	if err != nil {
		h.refuse(c, refusal{
			reason: "send the image as form data, with a metadata part and a file part",
			cause:  err,
		})
		return
	}
	metadata, err := readMediaMetadata(parts)
	if err != nil {
		h.refuse(c, err)
		return
	}
	file, err := nextPart(parts, filePart)
	if err != nil {
		h.refuse(c, err)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()
	candidate := &asset.Candidate{Version: version}
	added, err := h.assets.AddMedia(c.Request.Context(), asset.AddMediaInput{
		OwnerID: owner.ID,
		AssetID: id,
		Role:    asset.MediaRole(metadata.Role),
		File:    limitedFile,
	}, candidate)
	if work.CandidateResult(c, candidate, err) {
		return
	}
	if errors.Is(err, asset.ErrMediaNotFound) || errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such asset")
		return
	}
	if err != nil {
		h.refuse(c, err)
		return
	}
	c.JSON(http.StatusCreated, toAPIMedia(added))
}

func (h *Handlers) ListMedia(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	found, err := h.assets.ListMedia(c.Request.Context(), id, viewerID)
	if errors.Is(err, asset.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such asset")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "could not list the images")
		return
	}
	items := make([]Media, 0, len(found))
	for _, item := range found {
		items = append(items, toAPIMedia(item))
	}
	c.JSON(http.StatusOK, MediaList{Items: items})
}

func (h *Handlers) GetIngest(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	operation, err := h.assets.GetIngest(c.Request.Context(), owner.ID, id)
	if errors.Is(err, asset.ErrIngestNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such ingest operation")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not load import status. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIIngest(operation))
}

func toAPIIngest(operation asset.IngestOperation) gin.H {
	response := gin.H{
		"id":     operation.ID,
		"status": operation.Status,
		"url":    "/v1/ingests/" + operation.ID.String(),
		"asset":  ingestAsset(operation.Asset),
	}
	if operation.Failure != nil {
		response["failure"] = gin.H{
			"reason":  operation.Failure.Reason,
			"message": operation.Failure.Message,
		}
	}
	if operation.Preview != nil {
		response["preview"] = gin.H{
			"format": operation.Preview.Format, "groups": version.ToChangeGroups(operation.Preview.Groups),
			"conflicts":       nonNilStrings(operation.Preview.Conflicts),
			"unrepresentable": nonNilStrings(operation.Preview.Unrepresentable),
			"missingWording":  nonNilStrings(operation.Preview.MissingWording),
			"seals":           operation.Preview.Seals,
		}
	}
	return response
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func ingestAsset(a *asset.Asset) *Asset {
	if a == nil {
		return nil
	}
	converted := toAPI(*a)
	return &converted
}

func toAPI(a asset.Asset) Asset {
	return Asset{
		Id: a.ID, Kind: a.Kind, Format: a.Format,
		Name: a.Name, Blurb: a.Blurb, Tags: a.Tags, IsNsfw: a.IsNSFW,
		Discovery: AssetDiscovery(a.Discovery), CreatedAt: a.CreatedAt,
	}
}
