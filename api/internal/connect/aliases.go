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

// The field names a send and the app's library answered to before the renames, kept for sixty days
var (
	deliveryFileAliases = map[string]string{"type": "kind"}
	deliveryWorkAliases = map[string]string{
		"workId": "assetId", "type": "kind", "versionNumber": "contentGeneration", "files": "artifacts",
	}
	libraryEntryAliases     = map[string]string{"workId": "assetId", "versionNumber": "contentGeneration"}
	queuedDeliveryAliases   = map[string]string{"workId": "assetId"}
	withheldNoticeAliases   = map[string]string{"workId": "assetId"}
	workInstanceAliases     = map[string]string{"installedVersion": "installedGeneration"}
	workInstanceListAliases = map[string]string{"versionNumber": "contentGeneration"}
)

func (a DeliveryFile) MarshalJSON() ([]byte, error) {
	type plain DeliveryFile
	return api.MarshalAliased(plain(a), deliveryFileAliases)
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

func (i WorkInstance) MarshalJSON() ([]byte, error) {
	type plain WorkInstance
	return api.MarshalAliased(plain(i), workInstanceAliases)
}

func (l WorkInstanceList) MarshalJSON() ([]byte, error) {
	type plain WorkInstanceList
	return api.MarshalAliased(plain(l), workInstanceListAliases)
}

func (d QueuedDelivery) MarshalJSON() ([]byte, error) {
	type plain QueuedDelivery
	return api.MarshalAliased(plain(d), queuedDeliveryAliases)
}

func (n WithheldNotice) MarshalJSON() ([]byte, error) {
	type plain WithheldNotice
	return api.MarshalAliased(plain(n), withheldNoticeAliases)
}
