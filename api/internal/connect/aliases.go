package connect

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the rename, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/assets/:id/instances", d.JSON, h.GetWorkInstances)
	routes.Handle(http.MethodPost, "/v1/assets/:id/deliveries", d.JSON, h.SendWorkToInstance)
}

// The field names a send and the app's library answered to before the rename, kept for sixty days
var (
	deliveryArtifactAliases = map[string]string{"type": "kind"}
	deliveryWorkAliases     = map[string]string{"workId": "assetId", "type": "kind"}
	libraryEntryAliases     = map[string]string{"workId": "assetId"}
	queuedDeliveryAliases   = map[string]string{"workId": "assetId"}
	withheldNoticeAliases   = map[string]string{"workId": "assetId"}
)

func (a DeliveryArtifact) MarshalJSON() ([]byte, error) {
	type plain DeliveryArtifact
	return api.MarshalAliased(plain(a), deliveryArtifactAliases)
}

func (d DeliveryWork) MarshalJSON() ([]byte, error) {
	type plain DeliveryWork
	return api.MarshalAliased(plain(d), deliveryWorkAliases)
}

func (e LibraryEntry) MarshalJSON() ([]byte, error) {
	type plain LibraryEntry
	return api.MarshalAliased(plain(e), libraryEntryAliases)
}

func (e *LibraryEntry) UnmarshalJSON(data []byte) error {
	type plain LibraryEntry
	return api.UnmarshalAliased(data, (*plain)(e), libraryEntryAliases)
}

func (d QueuedDelivery) MarshalJSON() ([]byte, error) {
	type plain QueuedDelivery
	return api.MarshalAliased(plain(d), queuedDeliveryAliases)
}

func (n WithheldNotice) MarshalJSON() ([]byte, error) {
	type plain WithheldNotice
	return api.MarshalAliased(plain(n), withheldNoticeAliases)
}
