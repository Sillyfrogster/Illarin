package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

const workingCopyVersionHeader = "X-Working-Copy-Version"

// workingCopyVersion reads the working copy version the creator last saw
func workingCopyVersion(c *gin.Context) (int64, bool) {
	values := c.Request.Header.Values(workingCopyVersionHeader)
	if len(values) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "Header parameter " + workingCopyVersionHeader + " is required, but not found"})
		return 0, false
	}
	if len(values) != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"msg": fmt.Sprintf("Expected one value for %s, got %d", workingCopyVersionHeader, len(values))})
		return 0, false
	}
	version, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		api.RefuseParameter(c, workingCopyVersionHeader, err)
		return 0, false
	}
	return version, true
}
