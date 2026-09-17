package blog

import (
	"errors"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	postbody "github.com/Sillyfrogster/Illarin/api/internal/blog/body"
	"github.com/gin-gonic/gin"
	"mime"
	"net/http"
)

const maxImportBytes = 1 << 20

func (h *Handlers) ImportPostMarkdown(c *gin.Context) {
	id, ok := api.PathID(c, "id")
	if !ok {
		return
	}
	editor, ok := h.postEditor(c, "importing Markdown into a post")
	if !ok {
		return
	}
	request, ok := readImportRequest(c)
	if !ok {
		return
	}
	saved, notes, err := h.publications.ImportPost(c.Request.Context(), editor, id,
		PostImport{Version: request.Version, Markdown: request.Markdown})
	if err != nil {
		h.importError(c, err)
		return
	}
	c.JSON(http.StatusOK, PostImportResponse{PostResponse: h.toAPIPost(saved), Warnings: toAPINotes(notes)})
}

func readImportRequest(c *gin.Context) (ImportPostMarkdownRequest, bool) {
	var request ImportPostMarkdownRequest
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "application/json" {
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid,
			"Send JSON with the application/json content type.")
		return request, false
	}
	body := http.MaxBytesReader(c.Writer, c.Request.Body, maxImportBytes)
	if err := api.DecodeOneJSON(body, &request); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			refuseField(c, http.StatusRequestEntityTooLarge, PublicationErrorCodeInvalid,
				"This import is too large.", "markdown")
			return request, false
		}
		refusePublication(c, http.StatusBadRequest, PublicationErrorCodeInvalid, "Send one valid JSON object.")
		return request, false
	}
	return request, true
}

func (h *Handlers) importError(c *gin.Context, err error) {
	var refused postbody.Refused
	if !errors.As(err, &refused) {
		h.postError(c, err)
		return
	}
	lines := toAPINotes(refused.Notes)
	c.AbortWithStatusJSON(http.StatusBadRequest, PostImportRefusal{
		Error:    "This Markdown contains unsupported content. Review the reported lines.",
		Code:     PublicationErrorCodeInvalid,
		Field:    pointer("markdown"),
		Refusals: &lines,
	})
}

func toAPINotes(notes []postbody.Note) []PostImportNote {
	said := make([]PostImportNote, 0, len(notes))
	for _, note := range notes {
		said = append(said, PostImportNote{Line: note.Line, Message: note.Message})
	}
	return said
}

type ImportPostMarkdownRequest struct {
	Markdown string `json:"markdown"`
	Version  int    `json:"version"`
}

type PostImportResponse struct {
	PostResponse PostResponse     `json:"post"`
	Warnings     []PostImportNote `json:"warnings"`
}

type PostImportNote struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

type PostImportRefusal struct {
	Code     PublicationErrorCode `json:"code"`
	Error    string               `json:"error"`
	Field    *string              `json:"field,omitempty"`
	Refusals *[]PostImportNote    `json:"refusals,omitempty"`
}
