package page

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
)

func CandidateResult(c *gin.Context, candidate *work.Candidate, err error) bool {
	var conflict *work.VersionConflict
	switch {
	case errors.As(err, &conflict):
		c.JSON(http.StatusConflict, CandidateConflict{
			Code: "drafted_changes_conflict", Error: conflict.Error(), CurrentVersion: &conflict.CurrentVersion,
		})
		return true
	case errors.Is(err, work.ErrVersionRequired):
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the drafted-changes version you reviewed.", "code": "drafted_changes_version_required"})
		return true
	case errors.Is(err, work.ErrWorkFrozen):
		c.JSON(http.StatusConflict, CandidateConflict{Code: CandidateConflictCodeWorkFrozen, Error: "A taken-down work cannot be changed."})
		return true
	case err == nil && candidate.SavedVersion > 0:
		c.Header("X-Drafted-Changes-Version", strconv.FormatInt(candidate.SavedVersion, 10))
	}
	return false
}
