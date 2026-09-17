package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListAssetUpdates(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	history, err := h.assets.VersionHistory(c.Request.Context(), id, viewerID)
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

func (h *Handlers) RestoreAssetVersion(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	number, ok := pathNumber(c, "number")
	if !ok {
		return
	}
	workingCopyVersion, ok := workingCopyVersion(c)
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "restoring an asset version")
	if !ok {
		return
	}
	candidate := &asset.Candidate{Version: workingCopyVersion}
	err := h.assets.RestoreVersion(c.Request.Context(), owner.ID, id, number, candidate)
	if candidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, asset.ErrInvalidBlock):
		c.JSON(http.StatusConflict, gin.H{
			"error": "This version no longer forms a valid working copy.",
			"code":  "invalid_recorded_version",
		})
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not restore the version."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) CorrectAssetVersionNotes(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	number, ok := pathNumber(c, "number")
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "correcting asset update notes")
	if !ok {
		return
	}
	var request AssetVersionNotesRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the corrected summary and notes."})
		return
	}
	err := h.assets.CorrectVersionNotes(c.Request.Context(), owner.ID, id, number,
		request.Summary, valueOrEmpty(request.Notes))
	switch {
	case errors.Is(err, asset.ErrSummaryRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keep a summary for this update."})
	case errors.Is(err, asset.ErrSummaryTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "The summary or notes are too long."})
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
	case errors.Is(err, asset.ErrAssetFrozen):
		c.JSON(http.StatusConflict, gin.H{"error": "This asset is frozen while it is withheld."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not correct the notes."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) WithdrawAssetVersion(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	number, ok := pathNumber(c, "number")
	if !ok {
		return
	}
	owner, ok := h.verifiedAccount(c, "withdrawing an asset version")
	if !ok {
		return
	}
	var request AssetVersionWithdrawalRequest
	if err := decodeOneJSON(c.Request.Body, &request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send a public withdrawal explanation."})
		return
	}
	err := h.assets.WithdrawVersion(c.Request.Context(), owner.ID, id, number, request.Explanation)
	switch {
	case errors.Is(err, asset.ErrWithdrawalExplanationRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Explain why this version was withdrawn."})
	case errors.Is(err, asset.ErrWithdrawalExplanationTooLong):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Keep the explanation under 1,000 characters."})
	case errors.Is(err, asset.ErrCurrentVersionWithdrawal):
		c.JSON(http.StatusConflict, gin.H{"error": "Publish a replacement before withdrawing the current version."})
	case errors.Is(err, asset.ErrVersionAlreadyWithdrawn):
		c.JSON(http.StatusConflict, gin.H{"error": "This version is already withdrawn."})
	case errors.Is(err, asset.ErrAssetFrozen):
		c.JSON(http.StatusConflict, gin.H{"error": "This asset is frozen while it is withheld."})
	case errors.Is(err, asset.ErrNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not withdraw the version."})
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) CompareAssetVersions(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	q := readQuery(c)
	params := CompareAssetVersionsParams{
		From: queryNumber(q, "from"),
		To:   queryNumber(q, "to"),
	}
	if q.refused(c) {
		return
	}
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	visibility, ok := h.readerVisibility(c, nil)
	if !ok {
		return
	}
	compared, err := h.assets.CompareVersions(
		c.Request.Context(), id, viewerID,
		versionNumber(params.From), versionNumber(params.To), visibility,
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

func (h *Handlers) GetRecordedVersionDownloads(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	number, ok := pathNumber(c, "number")
	if !ok {
		return
	}
	q := readQuery(c)
	params := GetRecordedVersionDownloadsParams{
		Nsfw: queryText[GetRecordedVersionDownloadsParamsNsfw](q, "nsfw"),
	}
	if q.refused(c) {
		return
	}
	viewerID, ok := h.viewerID(c)
	if !ok {
		return
	}
	var requested *string
	if params.Nsfw != nil {
		value := string(*params.Nsfw)
		requested = &value
	}
	visibility, ok := h.readerVisibility(c, requested)
	if !ok {
		return
	}
	offered, err := h.assets.RecordedDownloads(c.Request.Context(), id, viewerID, number, visibility)
	if errors.Is(err, asset.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "No such version."})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the version's downloads."})
		return
	}
	blocks, err := toAPIBlocks(offered.Kind, offered.Blocks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read the version's downloads."})
		return
	}
	c.JSON(http.StatusOK, RecordedVersionDownloads{
		Version:           toAPIRecordedVersion(offered.Version),
		Kind:              RecordedVersionDownloadsKind(offered.Kind),
		LinkedInstallOnly: offered.LinkedInstallOnly,
		Downloads:         toAPIDownloads(offered.Downloads),
		AppTargets:        toAPIAppTargets(offered.AppTargets),
		Blocks:            blocks,
		Media:             toAPIImages(offered.Media),
	})
}

func (h *Handlers) ListProtectionMismatches(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	owner, ok := h.signedInAccount(c, "reading an asset's sealed prompts")
	if !ok {
		return
	}
	mismatches, err := h.assets.ProtectionMismatches(c.Request.Context(), owner.ID, id)
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

func (h *Handlers) ResolvePromptCorrespondence(c *gin.Context) {
	id, ok := pathID(c, "id")
	if !ok {
		return
	}
	number, ok := pathNumber(c, "number")
	if !ok {
		return
	}
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
		answer := asset.PromptCorrespondence{Current: match.Current}
		if match.Recorded != nil {
			recorded := uuid.UUID(*match.Recorded)
			answer.Recorded = &recorded
		}
		answers = append(answers, answer)
	}
	err := h.assets.ResolvePromptCorrespondence(
		c.Request.Context(), owner.ID, id, number, answers)
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

func versionNumber(chosen *int) int {
	if chosen == nil {
		return 0
	}
	return *chosen
}

func toAPIRecordedVersion(recorded asset.Version) RecordedVersion {
	return RecordedVersion{
		Id: recorded.ID, Number: recorded.Number,
		RecordedAt: recorded.RecordedAt, Initial: recorded.Initial,
		VersionLabel: recorded.VersionLabel,
		Summary:      recorded.Summary, Notes: recorded.Notes,
		NotesEditedAt:         recorded.NotesEditedAt,
		WithdrawnAt:           recorded.WithdrawnAt,
		WithdrawalExplanation: textOrNil(recorded.WithdrawalExplanation),
	}
}

func toAPINamedPrompts(prompts []asset.NamedPrompt) []NamedPrompt {
	out := make([]NamedPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		out = append(out, NamedPrompt{Id: prompt.ID, Name: prompt.Name})
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
	if change.Note != "" {
		note := change.Note
		served.Note = &note
	}
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
	if change.BeforeImage != "" {
		before := change.BeforeImage
		served.BeforeImage = &before
	}
	if change.AfterImage != "" {
		after := change.AfterImage
		served.AfterImage = &after
	}
	return served
}
