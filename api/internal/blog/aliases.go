package blog

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the renames, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/publication/categories", d.JSON, h.ListBlogCategories)
	routes.Handle(http.MethodPut, "/v1/publication/categories", d.JSON, h.OrderBlogCategories)
	routes.Handle(http.MethodPatch, "/v1/publication/categories/:id", d.JSON, h.UpdateBlogCategory)
	routes.Handle(http.MethodGet, "/v1/publication/workspace", d.JSON, h.GetBlogWorkspace)
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
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/withdraw", d.JSON, h.UnpublishPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/republish", d.JSON, h.RepublishPost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/delete", d.JSON, h.DeletePost)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/recover", d.JSON, h.RecoverPost)
	routes.Handle(http.MethodDelete, "/v1/publication/posts/:id/schedule", d.JSON, h.CancelPostSchedule)
	routes.Handle(http.MethodPost, "/v1/publication/posts/:id/schedule", d.JSON, h.SchedulePost)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/schedule", d.JSON, h.ReplacePostSchedule)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/address", d.JSON, h.CorrectPostAddress)
	routes.Handle(http.MethodPut, "/v1/publication/posts/:id/byline", d.JSON, h.CorrectPostByline)
	routes.Handle(http.MethodPost, "/v1/blog/posts/:id/withdraw", d.JSON, h.UnpublishPost)
}

// The field names a post's requests answered to before the rename, kept for sixty days
var (
	postIntegrationAliases = map[string]string{
		"integrationIds": "destinationIds", "roleIntegrationIds": "roleDestinationIds",
	}
)

func (r *PublishPostRequest) UnmarshalJSON(data []byte) error {
	type plain PublishPostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *RepublishPostRequest) UnmarshalJSON(data []byte) error {
	type plain RepublishPostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *UnpublishPostRequest) UnmarshalJSON(data []byte) error {
	type plain UnpublishPostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *SchedulePostRequest) UnmarshalJSON(data []byte) error {
	type plain SchedulePostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *ReplacePostScheduleRequest) UnmarshalJSON(data []byte) error {
	type plain ReplacePostScheduleRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}
