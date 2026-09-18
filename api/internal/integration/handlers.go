package integration

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
)

type BlogAccess = blog.IntegrationAccess

type Handlers struct {
	publications       *blog.Service
	updateDestinations *Service
	access             BlogAccess
}

func NewHandlers(publications *blog.Service, destinations *Service, access BlogAccess) *Handlers {
	return &Handlers{publications: publications, updateDestinations: destinations, access: access}
}
func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/update-destinations", d.JSON, h.ListWorkUpdateDestinationChoices)
	routes.Handle(http.MethodPut, "/v1/works/:id/update-destinations", d.JSON, h.SetWorkUpdateDestinationDefaults)
	routes.Handle(http.MethodGet, "/v1/works/:id/announcements", d.JSON, h.ListWorkUpdateAnnouncements)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations", d.JSON, h.ListWorkUpdateDestinations)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations", d.Verify, h.AddWorkUpdateDestination)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id", d.JSON, h.RemoveWorkUpdateDestination)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations/:id", d.JSON, h.GetWorkUpdateDestination)
	routes.Handle(http.MethodPatch, "/v1/account/update-destinations/:id", d.Verify, h.UpdateWorkUpdateDestination)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id/verification", d.JSON, h.DisableWorkUpdateDestination)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/verification", d.Verify, h.VerifyWorkUpdateDestination)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/secret", d.JSON, h.RotateWorkUpdateDestinationSecret)
	routes.Handle(http.MethodGet, "/v1/publication/destinations", d.JSON, h.ListPublicationDestinations)
	routes.Handle(http.MethodPost, "/v1/publication/destinations", d.JSON, h.AddPublicationDestination)
	routes.Handle(http.MethodDelete, "/v1/publication/destinations/:id", d.JSON, h.RemovePublicationDestination)
	routes.Handle(http.MethodPatch, "/v1/publication/destinations/:id", d.JSON, h.UpdatePublicationDestination)
	routes.Handle(http.MethodDelete, "/v1/publication/destinations/:id/verification", d.JSON, h.DisablePublicationDestination)
	routes.Handle(http.MethodPost, "/v1/publication/destinations/:id/verification", d.Verify, h.VerifyPublicationDestination)
	routes.Handle(http.MethodPost, "/v1/publication/destinations/:id/secret", d.JSON, h.RotatePublicationDestinationSecret)
	routes.Handle(http.MethodPost, "/v1/publication/channels", d.Verify, h.AddPublicationChannel)
	routes.Handle(http.MethodPatch, "/v1/publication/channels/:id", d.Verify, h.UpdatePublicationChannel)
	routes.Handle(http.MethodGet, "/v1/publication/deliveries", d.JSON, h.ListPublicationDeliveries)
	routes.Handle(http.MethodGet, "/v1/publication/deliveries/:id/attempts", d.JSON, h.ListPublicationDeliveryAttempts)
	routes.Handle(http.MethodPost, "/v1/publication/deliveries/:id/replay", d.JSON, h.ReplayPublicationDelivery)
	routes.Handle(http.MethodPost, "/v1/publication/deliveries/:id/repair", d.Verify, h.RepairDiscordAnnouncement)
	routes.Handle(http.MethodPut, "/v1/publication/grants/:id/destinations", d.JSON, h.SetPublicationGrantDestinations)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/destinations", d.JSON, h.ListPostDestinations)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/deliveries", d.JSON, h.ListPostDeliveries)
	registerAliases(routes, h)
}
