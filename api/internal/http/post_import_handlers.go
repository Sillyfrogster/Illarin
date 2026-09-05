package http

import (
	"errors"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

func (h *Handlers) ImportPostMarkdown(c *gin.Context, id types.UUID, _ ImportPostMarkdownParams) {
	editor, ok := h.postEditor(c, "importing Markdown into a post")
	if !ok {
		return
	}
	var request ImportPostMarkdownRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Send the import as JSON."})
		return
	}
	saved, notes, err := h.publications.ImportPost(c.Request.Context(), editor, uuid.UUID(id),
		publication.PostImport{Version: request.Version, Markdown: request.Markdown})
	if err != nil {
		h.importError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostImport{Post: h.toAPIPost(saved), Warnings: toAPINotes(notes)})
}

// importError names the lines that stopped an import, or refuses as any other
// post route would.
func (h *Handlers) importError(c *gin.Context, err error) {
	var refused postdoc.Refused
	if !errors.As(err, &refused) {
		h.postError(c, err)
		return
	}
	c.AbortWithStatusJSON(http.StatusBadRequest, PostImportRefusal{
		Error:    "This Markdown carries things a post does not.",
		Code:     CodeInvalid,
		Field:    pointer("markdown"),
		Refusals: toAPINotes(refused.Notes),
	})
}

func toAPINotes(notes []postdoc.Note) []PostImportNote {
	said := make([]PostImportNote, 0, len(notes))
	for _, note := range notes {
		said = append(said, PostImportNote{Line: note.Line, Message: note.Message})
	}
	return said
}
