package version

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handlers) ListWorkVersions(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	history, err := h.versions.VersionHistory(c.Request.Context(), id, viewerID)
	if errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "No such work.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the version history.")
		return
	}
	items := make([]page.RecordedVersion, 0, len(history))
	for _, recorded := range history {
		items = append(items, page.ToRecordedVersion(recorded))
	}
	c.JSON(http.StatusOK, RecordedVersionList{Items: items})
}

func (h *Handlers) RestoreWorkVersion(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	number, ok := api.PathNumber(c, "number")
	if !ok {
		return
	}
	draftedChangesVersion, ok := api.DraftedChangesVersion(c)
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "restoring a version")
	if !ok {
		return
	}
	candidate := &work.Candidate{Version: draftedChangesVersion}
	err := h.versions.RestoreVersion(c.Request.Context(), owner.ID, id, number, candidate)
	if page.CandidateResult(c, candidate, err) {
		return
	}
	switch {
	case errors.Is(err, work.ErrInvalidBlock):
		c.JSON(http.StatusConflict, gin.H{
			"error": "This version no longer forms a valid drafted changes.",
			"code":  "invalid_recorded_version",
		})
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such version.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not restore the version.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) CorrectWorkVersionNotes(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	number, ok := api.PathNumber(c, "number")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "correcting version notes")
	if !ok {
		return
	}
	var request WorkVersionNotesRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send the corrected summary and notes.")
		return
	}
	err := h.versions.CorrectVersionNotes(c.Request.Context(), owner.ID, id, number,
		request.Summary, valueOrEmpty(request.Notes))
	switch {
	case errors.Is(err, ErrSummaryRequired):
		api.Refuse(c, http.StatusBadRequest, "Keep a summary for this version.")
	case errors.Is(err, ErrSummaryTooLong):
		api.Refuse(c, http.StatusBadRequest, "The summary or notes are too long.")
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such version.")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "This work is frozen while it is withheld.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not correct the notes.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) WithdrawWorkVersion(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	number, ok := api.PathNumber(c, "number")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "withdrawing a version")
	if !ok {
		return
	}
	var request WorkVersionWithdrawalRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a public withdrawal explanation.")
		return
	}
	err := h.versions.WithdrawVersion(c.Request.Context(), owner.ID, id, number, request.Explanation)
	switch {
	case errors.Is(err, ErrWithdrawalExplanationRequired):
		api.Refuse(c, http.StatusBadRequest, "Explain why this version was withdrawn.")
	case errors.Is(err, ErrWithdrawalExplanationTooLong):
		api.Refuse(c, http.StatusBadRequest, "Keep the explanation under 1,000 characters.")
	case errors.Is(err, ErrCurrentVersionWithdrawal):
		api.Refuse(c, http.StatusConflict, "Publish a replacement before withdrawing the current version.")
	case errors.Is(err, ErrVersionAlreadyWithdrawn):
		api.Refuse(c, http.StatusConflict, "This version is already withdrawn.")
	case errors.Is(err, work.ErrWorkFrozen):
		api.Refuse(c, http.StatusConflict, "This work is frozen while it is withheld.")
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such version.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not withdraw the version.")
	default:
		c.Status(http.StatusNoContent)
	}
}

func (h *Handlers) CompareWorkVersions(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	q := api.ReadQuery(c)
	params := CompareWorkVersionsParams{
		From: api.QueryNumber(q, "from"),
		To:   api.QueryNumber(q, "to"),
	}
	if q.Refused(c) {
		return
	}
	viewerID, ok := api.ViewerID(c)
	if !ok {
		return
	}
	preference, ok := page.ReaderNSFWPreference(c, h.accounts, nil)
	if !ok {
		return
	}
	compared, err := h.versions.CompareVersions(
		c.Request.Context(), id, viewerID,
		versionNumber(params.From), versionNumber(params.To), preference,
	)
	switch {
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such version.")
	case errors.Is(err, work.ErrNoEarlierVersion):
		api.Refuse(c, http.StatusConflict, "Nothing was recorded before that version.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not compare the versions.")
	default:
		c.JSON(http.StatusOK, toAPIComparison(compared))
	}
}

func (h *Handlers) ListProtectionMismatches(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	owner, ok := api.SignedIn(c, "reading a work's sealed prompts")
	if !ok {
		return
	}
	mismatches, err := h.versions.ProtectionMismatches(c.Request.Context(), owner.ID, id)
	if errors.Is(err, work.ErrNotFound) {
		api.Refuse(c, http.StatusNotFound, "No such work.")
		return
	}
	if err != nil {
		api.Refuse(c, http.StatusInternalServerError, "Could not read the sealed prompts.")
		return
	}
	items := make([]ProtectionMismatch, 0, len(mismatches))
	for _, mismatch := range mismatches {
		items = append(items, ProtectionMismatch{
			Version:   page.ToRecordedVersion(mismatch.Version),
			Unmatched: toAPINamedPrompts(mismatch.Unmatched),
			Recorded:  toAPINamedPrompts(mismatch.Recorded),
		})
	}
	c.JSON(http.StatusOK, ProtectionMismatchList{Items: items})
}

func (h *Handlers) ResolvePromptCorrespondence(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	number, ok := api.PathNumber(c, "number")
	if !ok {
		return
	}
	owner, ok := api.Verified(c, "settling a work's sealed prompts")
	if !ok {
		return
	}
	var request PromptCorrespondenceRequest
	if err := api.DecodeOneJSON(c.Request.Body, &request); err != nil {
		api.Refuse(c, http.StatusBadRequest, "Send a match for each sealed prompt, naming the recorded prompt it stands for.")
		return
	}
	answers := make([]PromptCorrespondence, 0, len(request.Matches))
	for _, match := range request.Matches {
		answer := PromptCorrespondence{Current: match.Current}
		if match.Recorded != nil {
			recorded := uuid.UUID(*match.Recorded)
			answer.Recorded = &recorded
		}
		answers = append(answers, answer)
	}
	err := h.versions.ResolvePromptCorrespondence(
		c.Request.Context(), owner.ID, id, number, answers)
	switch {
	case errors.Is(err, ErrUnknownPrompt):
		api.Refuse(c, http.StatusBadRequest, "That prompt is not one of the choices.")
	case errors.Is(err, work.ErrNotFound):
		api.Refuse(c, http.StatusNotFound, "No such version.")
	case err != nil:
		api.Refuse(c, http.StatusInternalServerError, "Could not settle the sealed prompts.")
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

func toAPINamedPrompts(prompts []Prompt) []NamedPrompt {
	out := make([]NamedPrompt, 0, len(prompts))
	for _, prompt := range prompts {
		out = append(out, NamedPrompt{Id: prompt.ID, Name: prompt.Name})
	}
	return out
}

func toAPIComparison(compared Comparison) VersionComparison {
	served := VersionComparison{
		From: page.ToRecordedVersion(compared.From), To: page.ToRecordedVersion(compared.To),
		Groups:          ToChangeGroups(compared.Groups),
		PromptsWithheld: compared.PromptsWithheld,
	}
	if compared.Unavailable != "" {
		unavailable := compared.Unavailable
		served.Unavailable = &unavailable
	}
	return served
}

func ToChangeGroups(groups []ChangeGroup) []VersionChangeGroup {
	served := make([]VersionChangeGroup, 0, len(groups))
	for _, group := range groups {
		changes := make([]VersionChange, 0, len(group.Changes))
		for _, change := range group.Changes {
			changes = append(changes, toAPIChange(change))
		}
		served = append(served, VersionChangeGroup{
			Subject: group.Subject, Label: group.Label, Changes: changes,
		})
	}
	return served
}

func toAPIChange(change Change) VersionChange {
	served := VersionChange{Type: VersionChangeType(change.Type), Name: change.Name}
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
