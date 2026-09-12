package http

import (
	"errors"
	"mime"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/postdoc"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
)

const maxImportBytes = 1 << 20

func (h *Handlers) ImportPostMarkdown(c *gin.Context, id types.UUID, _ ImportPostMarkdownParams) {
	editor, ok := h.postEditor(c, "importing Markdown into a post")
	if !ok {
		return
	}
	request, ok := readImportRequest(c)
	if !ok {
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

func readImportRequest(c *gin.Context) (ImportPostMarkdownRequest, bool) {
	var request ImportPostMarkdownRequest
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		refusePublication(c, http.StatusBadRequest, CodeInvalid,
			"Send JSON with the application/json content type.")
		return request, false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, maxImportBytes)
	if err := decodeOneJSON(body, &request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			refuseField(c, http.StatusRequestEntityTooLarge, CodeInvalid,
				"This import is too large.", "markdown")
			return request, false
		}
		refusePublication(c, http.StatusBadRequest, CodeInvalid, "Send one valid JSON object.")
		return request, false
	}
	return request, true
}

func (h *Handlers) importError(c *gin.Context, err error) {
	var refused postdoc.Refused
	if !errors.As(err, &refused) {
		h.postError(c, err)
		return
	}
	lines := toAPINotes(refused.Notes)
	c.AbortWithStatusJSON(http.StatusBadRequest, PostImportRefusal{
		Error:    "This Markdown contains unsupported content. Review the reported lines.",
		Code:     CodeInvalid,
		Field:    pointer("markdown"),
		Refusals: &lines,
	})
}

func toAPINotes(notes []postdoc.Note) []PostImportNote {
	said := make([]PostImportNote, 0, len(notes))
	for _, note := range notes {
		said = append(said, PostImportNote{Line: note.Line, Message: note.Message})
	}
	return said
}
