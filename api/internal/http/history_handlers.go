package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

// ListAssetUpdates answers the versions an asset has recorded.
func (h *Handlers) ListAssetUpdates(c *gin.Context, id types.UUID) {
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	history, err := h.assets.VersionHistory(c.Request.Context(), uuid.UUID(id), viewerID)
	if errors.Is(err, asset.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the update history."})
		return
	}
	items := make([]RecordedVersion, 0, len(history))
	for _, recorded := range history {
		items = append(items, toAPIRecordedVersion(recorded))
	}
	c.JSON(http.StatusOK, RecordedVersionList{Items: items})
}

// CompareAssetVersions answers what changed between two recorded versions.
func (h *Handlers) CompareAssetVersions(c *gin.Context, id types.UUID, params CompareAssetVersionsParams) {
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	compared, err := h.assets.CompareVersions(
		c.Request.Context(), uuid.UUID(id), viewerID,
		versionNumber(params.From), versionNumber(params.To),
	)
	switch {
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
	case errors.Is(err, asset.ErrNoEarlierVersion):
		c.JSON(http.StatusConflict, gin.H{"error": "Nothing was recorded before that version."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not compare the versions."})
	default:
		c.JSON(http.StatusOK, toAPIComparison(compared))
	}
}

// ListProtectionMismatches answers the recorded versions an owner still has to settle.
func (h *Handlers) ListProtectionMismatches(c *gin.Context, id types.UUID) {
	owner, ok := h.signedInAccount(c, "reading an asset's sealed prompts")
	if !ok {
		return
	}
	mismatches, err := h.assets.ProtectionMismatches(c.Request.Context(), owner.ID, uuid.UUID(id))
	if errors.Is(err, asset.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No such asset."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the sealed prompts."})
		return
	}
	items := make([]ProtectionMismatch, 0, len(mismatches))
	for _, mismatch := range mismatches {
		items = append(items, ProtectionMismatch{
			Version:   toAPIRecordedVersion(mismatch.Version),
			Unmatched: toAPINamedPrompts(mismatch.Unmatched),
			Recorded:  toAPINamedPrompts(mismatch.Recorded),
		})
	}
	c.JSON(http.StatusOK, ProtectionMismatchList{Items: items})
}

// ResolvePromptCorrespondence records which recorded prompt each sealed prompt is.
func (h *Handlers) ResolvePromptCorrespondence(c *gin.Context, id types.UUID, number int) {
	owner, ok := h.verifiedAccount(c, "settling an asset's sealed prompts")
	if !ok {
		return
	}
	var request PromptCorrespondenceRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Send a match for each sealed prompt, naming the recorded prompt it stands for.",
		})
		return
	}
	answers := make([]asset.PromptCorrespondence, 0, len(request.Matches))
	for _, match := range request.Matches {
		answer := asset.PromptCorrespondence{Current: uuid.UUID(match.Current)}
		if match.Recorded != nil {
			recorded := uuid.UUID(*match.Recorded)
			answer.Recorded = &recorded
		}
		answers = append(answers, answer)
	}
	err := h.assets.ResolvePromptCorrespondence(
		c.Request.Context(), owner.ID, uuid.UUID(id), number, answers)
	switch {
	case errors.Is(err, asset.ErrUnknownPrompt):
		c.JSON(http.StatusBadRequest, gin.H{"error": "That prompt is not one of the choices."})
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not settle the sealed prompts."})
	default:
		c.Status(http.StatusNoContent)
	}
}

// versionNumber reads an absent version as the one the comparison picks itself.
func versionNumber(chosen *int) int {
	if chosen == nil {
		return 0
	}
	return *chosen
}

func toAPIRecordedVersion(recorded asset.Version) RecordedVersion {
	return RecordedVersion{
		Id: types.UUID(recorded.ID), Number: recorded.Number,
		RecordedAt: recorded.RecordedAt, VersionLabel: recorded.VersionLabel,
		Summary: recorded.Summary, Notes: recorded.Notes,
	}
}

func toAPINamedPrompts(prompts []asset.NamedPrompt) []NamedPrompt {
	out := make([]NamedPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		out = append(out, NamedPrompt{Id: types.UUID(prompt.ID), Name: prompt.Name})
	}
	return out
}

func toAPIComparison(compared asset.Comparison) VersionComparison {
	served := VersionComparison{
		From: toAPIRecordedVersion(compared.From), To: toAPIRecordedVersion(compared.To),
		Groups:          make([]VersionChangeGroup, 0, len(compared.Groups)),
		PromptsWithheld: compared.PromptsWithheld,
	}
	if compared.Unavailable != "" {
		unavailable := compared.Unavailable
		served.Unavailable = &unavailable
	}
	for _, group := range compared.Groups {
		changes := make([]VersionChange, 0, len(group.Changes))
		for _, change := range group.Changes {
			changes = append(changes, toAPIChange(change))
		}
		served.Groups = append(served.Groups, VersionChangeGroup{
			Subject: group.Subject, Label: group.Label, Changes: changes,
		})
	}
	return served
}

func toAPIChange(change asset.Change) VersionChange {
	served := VersionChange{Kind: VersionChangeKind(change.Kind), Name: change.Name}
	if change.PreviousName != "" {
		previous := change.PreviousName
		served.PreviousName = &previous
	}
	if change.Before != "" {
		before := change.Before
		served.Before = &before
	}
	if change.After != "" {
		after := change.After
		served.After = &after
	}
	if change.BeforeMedia != nil {
		before := types.UUID(*change.BeforeMedia)
		served.BeforeMedia = &before
	}
	if change.AfterMedia != nil {
		after := types.UUID(*change.AfterMedia)
		served.AfterMedia = &after
	}
	return served
}
