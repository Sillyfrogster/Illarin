package blog

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handlers struct {
	publications   *Service
	accounts       *account.Service
	maxUploadBytes int64
}

func NewHandlers(service *Service, accounts *account.Service, maxUploadBytes int64) *Handlers {
	return &Handlers{publications: service, accounts: accounts, maxUploadBytes: maxUploadBytes}
}

func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/publication/categories", d.JSON, h.ListPublicationCategories)
	routes.Handle(http.MethodPut, "/v1/publication/categories", d.JSON, h.OrderPublicationCategories)
	routes.Handle(http.MethodPatch, "/v1/publication/categories/:id", d.JSON, h.UpdatePublicationCategory)
	routes.Handle(http.MethodGet, "/v1/publication/grants", d.JSON, h.ListPublicationGrants)
	routes.Handle(http.MethodPost, "/v1/publication/grants", d.JSON, h.CreatePublicationGrant)
	routes.Handle(http.MethodDelete, "/v1/publication/grants/:id", d.JSON, h.RevokePublicationGrant)
	routes.Handle(http.MethodPatch, "/v1/publication/grants/:id", d.JSON, h.UpdatePublicationGrant)
	routes.Handle(http.MethodGet, "/v1/publication/workspace", d.JSON, h.GetPublicationWorkspace)
	routes.Handle(http.MethodGet, "/v1/publication/posts", d.JSON, h.ListPosts)
	routes.Handle(http.MethodPost, "/v1/publication/posts", d.JSON, h.CreatePost)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id", d.JSON, h.GetPost)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id", d.JSON, h.SavePost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/media", d.Upload, h.AddPostMedia)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/revisions", d.JSON, h.ListPostRevisions)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/revisions", d.JSON, h.CheckpointPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/revisions/:revisionId/restore", d.JSON, h.RestorePostRevision)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/history", d.JSON, h.ReadPostHistory)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/import", d.JSON, h.ImportPostMarkdown)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/publish", d.JSON, h.PublishPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/withdraw", d.JSON, h.WithdrawPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/republish", d.JSON, h.RepublishPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/delete", d.JSON, h.DeletePost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/recover", d.JSON, h.RecoverPost)
	routes.Handle(http.MethodDelete, "/v1/publication/posts/:id/schedule", d.JSON, h.CancelPostSchedule)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/schedule", d.JSON, h.SchedulePost)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/schedule", d.JSON, h.ReplacePostSchedule)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/address", d.JSON, h.CorrectPostAddress)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/byline", d.JSON, h.CorrectPostByline)
	routes.Handle(http.MethodGet, "/v1/post-categories", d.JSON, h.ListPostCategories)
	routes.Handle(http.MethodGet, "/v1/post-apps", d.JSON, h.ListPostApps)
	routes.Handle(http.MethodGet, "/v1/posts", d.JSON, h.ListPublishedPosts)
	routes.Handle(http.MethodGet, "/v1/posts/:slug", d.JSON, h.GetPublishedPost)
}

func (h *Handlers) IntegrationAccess() IntegrationAccess {
	return IntegrationAccess{
		Authority: func(c *gin.Context, action string) (accountIdentity, bool) {
			current, ok := h.publicationAuthority(c, action)
			return accountIdentity{ID: current.ID, Handle: current.Handle}, ok
		},
		Editor: h.postEditor, PublicationError: h.publicationError, PostError: h.postError,
		Grant: func(c *gin.Context, id uuid.UUID) (any, error) {
			grant, err := h.publications.Grant(c.Request.Context(), id)
			if err != nil {
				h.publicationError(c, err)
				return nil, err
			}
			listed, err := h.withHolders(c, []Grant{grant})
			if err != nil {
				return nil, err
			}
			return listed[0], nil
		},
	}
}

type IntegrationAccess struct {
	Authority        func(*gin.Context, string) (accountIdentity, bool)
	Editor           func(*gin.Context, string) (Editor, bool)
	PublicationError func(*gin.Context, error)
	PostError        func(*gin.Context, error)
	Grant            func(*gin.Context, uuid.UUID) (any, error)
}

func refusePublication(c *gin.Context, status int, code PublicationErrorCode, message string) {
	c.AbortWithStatusJSON(status, PublicationError{Error: message, Code: code})
}

func refuseField(
	c *gin.Context,
	status int,
	code PublicationErrorCode,
	message string,
	field string,
) {
	c.AbortWithStatusJSON(status, PublicationError{Error: message, Code: code, Field: &field})
}

type PublicationError struct {
	Code  PublicationErrorCode `json:"code"`
	Error string               `json:"error"`
	Field *string              `json:"field,omitempty"`
}

type PublicationErrorCode string

const (
	PublicationErrorCodeAlreadyScheduled PublicationErrorCode = "already_scheduled"
	PublicationErrorCodeCategoryRefused  PublicationErrorCode = "category_refused"
	PublicationErrorCodeForbidden        PublicationErrorCode = "forbidden"
	PublicationErrorCodeInvalid          PublicationErrorCode = "invalid"
	PublicationErrorCodeNotFound         PublicationErrorCode = "not_found"
	PublicationErrorCodeScheduleRunning  PublicationErrorCode = "schedule_running"
	PublicationErrorCodeServerError      PublicationErrorCode = "server_error"
	PublicationErrorCodeStaleVersion     PublicationErrorCode = "stale_version"
	PublicationErrorCodeUnauthenticated  PublicationErrorCode = "unauthenticated"
)
