package integration

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
)

type BlogAccess = blog.IntegrationAccess

type Handlers struct {
	blog         *blog.Service
	integrations *Service
	access       BlogAccess
}

func NewHandlers(posts *blog.Service, integrations *Service, access BlogAccess) *Handlers {
	return &Handlers{blog: posts, integrations: integrations, access: access}
}
func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/integrations", d.JSON, h.ListWorkIntegrationChoices)
	routes.Handle(http.MethodPut, "/v1/works/:id/integrations", d.JSON, h.SetWorkIntegrationDefaults)
	routes.Handle(http.MethodGet, "/v1/works/:id/announcement-attempts", d.JSON, h.ListWorkAnnouncementAttempts)
	routes.Handle(http.MethodGet, "/v1/account/integrations", d.JSON, h.ListWorkIntegrations)
	routes.Handle(http.MethodPost, "/v1/account/integrations", d.Verify, h.AddWorkIntegration)
	routes.Handle(http.MethodDelete, "/v1/account/integrations/:id", d.JSON, h.RemoveWorkIntegration)
	routes.Handle(http.MethodGet, "/v1/account/integrations/:id", d.JSON, h.GetWorkIntegration)
	routes.Handle(http.MethodPatch, "/v1/account/integrations/:id", d.Verify, h.UpdateWorkIntegration)
	routes.Handle(http.MethodDelete, "/v1/account/integrations/:id/verification", d.JSON, h.DisableWorkIntegration)
	routes.Handle(http.MethodPost, "/v1/account/integrations/:id/verification", d.Verify, h.VerifyWorkIntegration)
	routes.Handle(http.MethodPost, "/v1/account/integrations/:id/secret", d.JSON, h.RotateWorkIntegrationSecret)
	routes.Handle(http.MethodGet, "/v1/blog/integrations", d.JSON, h.ListBlogIntegrations)
	routes.Handle(http.MethodPost, "/v1/blog/integrations", d.JSON, h.AddBlogIntegration)
	routes.Handle(http.MethodDelete, "/v1/blog/integrations/:id", d.JSON, h.RemoveBlogIntegration)
	routes.Handle(http.MethodPatch, "/v1/blog/integrations/:id", d.JSON, h.UpdateBlogIntegration)
	routes.Handle(http.MethodDelete, "/v1/blog/integrations/:id/verification", d.JSON, h.DisableBlogIntegration)
	routes.Handle(http.MethodPost, "/v1/blog/integrations/:id/verification", d.Verify, h.VerifyBlogIntegration)
	routes.Handle(http.MethodPost, "/v1/blog/integrations/:id/secret", d.JSON, h.RotateBlogIntegrationSecret)
	routes.Handle(http.MethodPost, "/v1/blog/channels", d.Verify, h.AddBlogChannel)
	routes.Handle(http.MethodPatch, "/v1/blog/channels/:id", d.Verify, h.UpdateBlogChannel)
	routes.Handle(http.MethodGet, "/v1/blog/announcement-attempts", d.JSON, h.ListBlogAnnouncementAttempts)
	routes.Handle(http.MethodGet, "/v1/blog/announcement-attempts/:id/tries", d.JSON, h.ListBlogAnnouncementTries)
	routes.Handle(http.MethodPost, "/v1/blog/announcement-attempts/:id/replay", d.JSON, h.ReplayBlogAnnouncementAttempt)
	routes.Handle(http.MethodPost, "/v1/blog/announcement-attempts/:id/repair", d.Verify, h.RepairDiscordAnnouncement)
	routes.Handle(http.MethodGet, "/v1/blog/posts/:id/integrations", d.JSON, h.ListPostIntegrations)
	routes.Handle(http.MethodGet, "/v1/blog/posts/:id/announcement-attempts", d.JSON, h.ListPostAnnouncementAttempts)
	registerAliases(routes, h)
}
