package integration

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/publication"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AccountIdentity struct {
	ID     uuid.UUID
	Handle string
}
type BlogAccess struct {
	Authority        func(*gin.Context, string) (AccountIdentity, bool)
	Editor           func(*gin.Context, string) (publication.Editor, bool)
	PublicationError func(*gin.Context, error)
	PostError        func(*gin.Context, error)
	Grant            func(*gin.Context, uuid.UUID) (any, error)
}
type Handlers struct {
	publications       *publication.Service
	updateDestinations *Service
	access             BlogAccess
}

func NewHandlers(publications *publication.Service, destinations *Service, access BlogAccess) *Handlers {
	return &Handlers{publications: publications, updateDestinations: destinations, access: access}
}
func Register(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/update-destinations", d.JSON, h.ListAssetUpdateDestinationChoices)
	routes.Handle(http.MethodPut, "/v1/assets/:id/update-destinations", d.JSON, h.SetAssetUpdateDestinationDefaults)
	routes.Handle(http.MethodGet, "/v1/assets/:id/announcements", d.JSON, h.ListAssetUpdateAnnouncements)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations", d.JSON, h.ListAssetUpdateDestinations)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations", d.Verify, h.AddAssetUpdateDestination)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id", d.JSON, h.RemoveAssetUpdateDestination)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations/:id", d.JSON, h.GetAssetUpdateDestination)
	routes.Handle(http.MethodPatch, "/v1/account/update-destinations/:id", d.Verify, h.UpdateAssetUpdateDestination)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id/verification", d.JSON, h.DisableAssetUpdateDestination)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/verification", d.Verify, h.VerifyAssetUpdateDestination)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/secret", d.JSON, h.RotateAssetUpdateDestinationSecret)
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
}
