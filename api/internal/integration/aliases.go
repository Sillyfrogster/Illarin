package integration

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/update-destinations", d.JSON, h.ListWorkUpdateDestinationChoices)
	routes.Handle(http.MethodPut, "/v1/assets/:id/update-destinations", d.JSON, h.SetWorkUpdateDestinationDefaults)
	routes.Handle(http.MethodGet, "/v1/assets/:id/announcements", d.JSON, h.ListWorkUpdateAnnouncements)
}

// The field names an integration answered to before the rename, kept for sixty days
var (
	destinationTypeAliases = map[string]string{"type": "kind"}
	sentWorkAliases        = map[string]string{"type": "kind"}
	sentAliases            = map[string]string{"work": "asset"}
)

func (d WorkUpdateDestination) MarshalJSON() ([]byte, error) {
	type plain WorkUpdateDestination
	return api.MarshalAliased(plain(d), destinationTypeAliases)
}

func (c WorkUpdateDestinationChoice) MarshalJSON() ([]byte, error) {
	type plain WorkUpdateDestinationChoice
	return api.MarshalAliased(plain(c), destinationTypeAliases)
}

func (a WorkUpdateAnnouncement) MarshalJSON() ([]byte, error) {
	type plain WorkUpdateAnnouncement
	return api.MarshalAliased(plain(a), destinationTypeAliases)
}

func (d PublicationDestination) MarshalJSON() ([]byte, error) {
	type plain PublicationDestination
	return api.MarshalAliased(plain(d), destinationTypeAliases)
}

func (d PostDelivery) MarshalJSON() ([]byte, error) {
	type plain PostDelivery
	return api.MarshalAliased(plain(d), destinationTypeAliases)
}

func (r *AddWorkUpdateDestinationRequest) UnmarshalJSON(data []byte) error {
	type plain AddWorkUpdateDestinationRequest
	return api.UnmarshalAliased(data, (*plain)(r), destinationTypeAliases)
}

func (s sent) MarshalJSON() ([]byte, error) {
	type plain sent
	return api.MarshalAliased(plain(s), sentAliases)
}

func (w sentWork) MarshalJSON() ([]byte, error) {
	type plain sentWork
	return api.MarshalAliased(plain(w), sentWorkAliases)
}
