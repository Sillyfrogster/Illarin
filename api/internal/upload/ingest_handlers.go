package upload

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/block/edit"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/private"
	"github.com/Sillyfrogster/Illarin/api/internal/storage"
	"github.com/Sillyfrogster/Illarin/api/internal/version"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func (h *Handlers) CreateWork(c *gin.Context) {
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	if c.ContentType() == "application/json" {
		h.startWorkFromNothing(c, owner)
		return
	}
	h.acceptUpload(c, owner)
}

func (h *Handlers) acceptUpload(c *gin.Context, owner api.Account) {
	parts, err := c.Request.MultipartReader()
	if err != nil {
		RefuseFile(c, api.FormRefusal{
			Reason: "send the work as form data, with a metadata part and a file part",
			Cause:  err,
		}, h.maxUploadBytes)
		return
	}

	metadata, err := readMetadata(parts)
	if err != nil {
		RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	file, err := api.NextPart(parts, api.FilePart)
	if err != nil {
		RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	if !metadata.Confirmed {
		RefuseFile(c, api.FormRefusal{Reason: "confirm the details before uploading"}, h.maxUploadBytes)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()

	operation, err := h.uploads.AcceptIngest(
		c.Request.Context(), ingestInput(metadata, file.FileName(), limitedFile, owner.ID),
	)
	if errors.Is(err, storage.ErrTombstoned) {
		api.Refuse(c, http.StatusUnprocessableEntity, "This file cannot be accepted.")
		return
	}
	if err != nil {
		RefuseFile(c, err, h.maxUploadBytes)
		return
	}

	location := "/v1/ingests/" + operation.ID.String()
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, toAPIIngest(operation))
}

func (h *Handlers) AddWorkOriginalFile(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}

	parts, err := c.Request.MultipartReader()
	if err != nil {
		RefuseFile(c, api.FormRefusal{
			Reason: "send the original file as form data, with a file part",
			Cause:  err,
		}, h.maxUploadBytes)
		return
	}
	file, err := api.NextPart(parts, api.FilePart)
	if err != nil {
		RefuseFile(c, err, h.maxUploadBytes)
		return
	}
	limitedFile := http.MaxBytesReader(c.Writer, file, h.maxUploadBytes)
	defer limitedFile.Close()

	candidate := &work.Candidate{Version: version}
	operation, err := h.uploads.AcceptOriginalFile(c.Request.Context(), OriginalFileInput{
		OwnerID:  owner.ID,
		WorkID:   id,
		Filename: file.FileName(),
		File:     limitedFile,
	}, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "no such work")
		return
	case errors.Is(err, storage.ErrTombstoned):
		api.Refuse(c, http.StatusUnprocessableEntity, "This file cannot be accepted.")
		return
	case err != nil:
		RefuseFile(c, err, h.maxUploadBytes)
		return
	}

	location := "/v1/ingests/" + operation.ID.String()
	c.Header("Location", location)
	c.JSON(http.StatusAccepted, toAPIIngest(operation))
}

func (h *Handlers) GetWorkReplacement(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	operation, err := h.uploads.ReviewedReplacement(c.Request.Context(), owner.ID, id)
	if errors.Is(err, ErrIngestNotFound) {
		c.JSON(http.StatusOK, nil)
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not load the replacement file. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIIngest(operation))
}

func (h *Handlers) AcceptWorkOriginalFile(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	operationID, ok := api.PathID(c, "operationId")
	if !ok {
		return
	}
	version, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "uploading")
	if !ok {
		return
	}
	var body ReplacementAcceptance
	if err := c.ShouldBindJSON(&body); err != nil {
		RefuseFile(c, api.FormRefusal{Reason: "send the replacement decisions as JSON", Cause: err}, h.maxUploadBytes)
		return
	}
	decisions := make(map[string]string, len(body.Unrepresentable))
	for role, decision := range body.Unrepresentable {
		decisions[role] = string(decision)
	}
	candidate := &work.Candidate{Version: version}
	operation, err := h.uploads.AcceptReplacement(c.Request.Context(), owner.ID, id, operationID, candidate, decisions, body.ExposeProtected != nil && *body.ExposeProtected)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	var exposure private.ExposureRefusal
	if errors.As(err, &exposure) {
		c.JSON(http.StatusConflict, edit.SealedExposureRefusal{
			Error:   "This replacement removes prompt protection. Confirm that text in this work and its recorded versions may become public immediately.",
			Code:    edit.SealedExposureRefusalCodeSealedExposure,
			Prompts: exposure.Prompts,
		})
		return
	}
	if errors.Is(err, ErrIngestNotFound) || errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "no reviewed replacement")
		return
	}
	if errors.Is(err, ErrReplacementDecision) {
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

func (h *Handlers) CancelWorkOriginalFile(c *gin.Context) {
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
	err := h.uploads.CancelReplacement(c.Request.Context(), owner.ID, id, operationID)
	if errors.Is(err, ErrIngestNotFound) {
		api.Refuse(c, http.StatusNotFound, "no reviewed replacement")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not discard the replacement file. Try again.")
		return
	}
	c.Status(http.StatusNoContent)
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
	operation, err := h.uploads.GetIngest(c.Request.Context(), owner.ID, id)
	if errors.Is(err, ErrIngestNotFound) {
		api.Refuse(c, http.StatusNotFound, "no such ingest operation")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not load import status. Try again.")
		return
	}
	c.JSON(http.StatusOK, toAPIIngest(operation))
}

func toAPIIngest(operation Operation) gin.H {
	response := gin.H{
		"id":     operation.ID,
		"status": operation.Status,
		"url":    "/v1/ingests/" + operation.ID.String(),
		"work":   ingestWork(operation.Work),
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
	return aliasIngestKeys(response)
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func ingestWork(a *work.Work) *Work {
	if a == nil {
		return nil
	}
	converted := toAPI(*a)
	return &converted
}

func toAPI(a work.Work) Work {
	return Work{
		Id: a.ID, Type: a.Type, Format: a.Format,
		Name: a.Name, Blurb: a.Blurb, Tags: a.Tags, IsNsfw: a.IsNSFW,
		Visibility: WorkVisibility(a.Visibility), CreatedAt: a.CreatedAt,
	}
}
