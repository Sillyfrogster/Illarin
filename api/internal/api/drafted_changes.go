package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

const draftedChangesVersionHeader = "X-Drafted-Changes-Version"

// The header the drafted changes version arrived under before the rename, accepted for sixty days
const workingCopyVersionHeader = "X-Working-Copy-Version"

// DraftedChangesVersion reads the drafted changes version the creator last saw
func DraftedChangesVersion(c *gin.Context) (int64, bool) {
	values := c.Request.Header.Values(draftedChangesVersionHeader)
	if len(values) == 0 {
		values = c.Request.Header.Values(workingCopyVersionHeader)
	}
	if len(values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Header parameter " + draftedChangesVersionHeader + " is required, but not found"})
		return 0, false
	}
	if len(values) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("Expected one value for %s, got %d", draftedChangesVersionHeader, len(values))})
		return 0, false
	}
	version, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		RefuseParameter(c, draftedChangesVersionHeader, err)
		return 0, false
	}
	return version, true
}
