package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ViewerID returns the signed-in account's id, or nil for a reader who is signed out
func ViewerID(c *gin.Context) (*uuid.UUID, bool) {
	current, err := Current(c)
	if err != nil {
		Refuse(c, http.StatusInternalServerError, "Could not read the signed-in account.")
		return nil, false
	}
	if current == nil {
		return nil, true
	}
	return &current.ID, true
}
