package blog

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
	blog           *Service
	accounts       *account.Service
	maxUploadBytes int64
}

func NewHandlers(service *Service, accounts *account.Service, maxUploadBytes int64) *Handlers {
	return &Handlers{blog: service, accounts: accounts, maxUploadBytes: maxUploadBytes}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/blog/categories", d.JSON, h.ListBlogCategories)
	routes.Handle(http.MethodPut, "/v1/blog/categories", d.JSON, h.OrderBlogCategories)
	routes.Handle(http.MethodPatch, "/v1/blog/categories/:id", d.JSON, h.UpdateBlogCategory)
	routes.Handle(http.MethodGet, "/v1/blog/writers", d.JSON, h.ListWriters)
	routes.Handle(http.MethodPost, "/v1/blog/writers", d.JSON, h.SwitchWriterOn)
	routes.Handle(http.MethodDelete, "/v1/blog/writers/:id", d.JSON, h.SwitchWriterOff)
	routes.Handle(http.MethodGet, "/v1/blog/workspace", d.JSON, h.GetBlogWorkspace)
	routes.Handle(http.MethodGet, "/v1/blog/posts", d.JSON, h.ListPosts)
	routes.Handle(http.MethodPost, "/v1/blog/posts", d.JSON, h.CreatePost)
	routes.Handle(http.MethodGet, "/v1/blog/posts/:id", d.JSON, h.GetPost)
	routes.Handle(http.MethodPut, "/v1/blog/posts/:id", d.JSON, h.SavePost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/media", d.Upload, h.AddPostMedia)
	routes.Handle(http.MethodGet, "/v1/blog/posts/:id/revisions", d.JSON, h.ListPostRevisions)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/revisions", d.JSON, h.CheckpointPost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/revisions/:revisionId/restore", d.JSON, h.RestorePostRevision)
	routes.Handle(http.MethodGet, "/v1/blog/posts/:id/history", d.JSON, h.ReadPostHistory)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/import", d.JSON, h.ImportPostMarkdown)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/publish", d.JSON, h.PublishPost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/unpublish", d.JSON, h.UnpublishPost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/republish", d.JSON, h.RepublishPost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/delete", d.JSON, h.DeletePost)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/recover", d.JSON, h.RecoverPost)
	routes.Handle(http.MethodDelete, "/v1/blog/posts/:id/schedule", d.JSON, h.CancelPostSchedule)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/schedule", d.JSON, h.SchedulePost)
	routes.Handle(http.MethodPut, "/v1/blog/posts/:id/schedule", d.JSON, h.ReplacePostSchedule)
	routes.Handle(http.MethodPut, "/v1/blog/posts/:id/address", d.JSON, h.CorrectPostAddress)
	routes.Handle(http.MethodPut, "/v1/blog/posts/:id/byline", d.JSON, h.CorrectPostByline)
	routes.Handle(http.MethodGet, "/v1/post-categories", d.JSON, h.ListPostCategories)
	routes.Handle(http.MethodGet, "/v1/posts", d.JSON, h.ListPublishedPosts)
	routes.Handle(http.MethodGet, "/v1/posts/:slug", d.JSON, h.GetPublishedPost)
	registerAliases(routes, h)
}

func (h *Handlers) IntegrationAccess() IntegrationAccess {
	return IntegrationAccess{
		Authority: func(c *gin.Context, action string) (accountIdentity, bool) {
			current, ok := h.blogAdmin(c, action)
			return accountIdentity{ID: current.ID, Handle: current.Handle}, ok
		},
		Editor: h.postEditor, BlogError: h.blogError, PostError: h.postError,
	}
}

type IntegrationAccess struct {
	Authority func(*gin.Context, string) (accountIdentity, bool)
	Editor    func(*gin.Context, string) (Editor, bool)
	BlogError func(*gin.Context, error)
	PostError func(*gin.Context, error)
}

func refuseBlog(c *gin.Context, status int, code BlogErrorCode, message string) {
	c.AbortWithStatusJSON(status, BlogError{Error: message, Code: code})
}

func refuseField(
	c *gin.Context,
	status int,
	code BlogErrorCode,
	message string,
	field string,
) {
	c.AbortWithStatusJSON(status, BlogError{Error: message, Code: code, Field: &field})
}

type BlogError struct {
	Code  BlogErrorCode `json:"code"`
	Error string        `json:"error"`
	Field *string       `json:"field,omitempty"`
}

type BlogErrorCode string

const (
	BlogErrorCodeAlreadyScheduled BlogErrorCode = "already_scheduled"
	BlogErrorCodeCategoryRefused  BlogErrorCode = "category_refused"
	BlogErrorCodeForbidden        BlogErrorCode = "forbidden"
	BlogErrorCodeInvalid          BlogErrorCode = "invalid"
	BlogErrorCodeNotFound         BlogErrorCode = "not_found"
	BlogErrorCodeScheduleRunning  BlogErrorCode = "schedule_running"
	BlogErrorCodeServerError      BlogErrorCode = "server_error"
	BlogErrorCodeStaleVersion     BlogErrorCode = "stale_version"
	BlogErrorCodeUnauthenticated  BlogErrorCode = "unauthenticated"
)
