package integration

import (
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// registerAliases serves the paths this package had before the renames, for sixty days
func registerAliases(routes api.Routes, h *Handlers) {
	d := routes.Deadlines
	routes.Handle(http.MethodGet, "/v1/works/:id/update-destinations", d.JSON, h.ListWorkIntegrationChoices)
	routes.Handle(http.MethodPut, "/v1/works/:id/update-destinations", d.JSON, h.SetWorkIntegrationDefaults)
	routes.Handle(http.MethodGet, "/v1/works/:id/announcements", d.JSON, h.ListWorkAnnouncementAttempts)
	routes.Handle(http.MethodGet, "/v1/assets/:id/update-destinations", d.JSON, h.ListWorkIntegrationChoices)
	routes.Handle(http.MethodPut, "/v1/assets/:id/update-destinations", d.JSON, h.SetWorkIntegrationDefaults)
	routes.Handle(http.MethodGet, "/v1/assets/:id/announcements", d.JSON, h.ListWorkAnnouncementAttempts)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations", d.JSON, h.ListWorkIntegrations)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations", d.Verify, h.AddWorkIntegration)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id", d.JSON, h.RemoveWorkIntegration)
	routes.Handle(http.MethodGet, "/v1/account/update-destinations/:id", d.JSON, h.GetWorkIntegration)
	routes.Handle(http.MethodPatch, "/v1/account/update-destinations/:id", d.Verify, h.UpdateWorkIntegration)
	routes.Handle(http.MethodDelete, "/v1/account/update-destinations/:id/verification", d.JSON, h.DisableWorkIntegration)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/verification", d.Verify, h.VerifyWorkIntegration)
	routes.Handle(http.MethodPost, "/v1/account/update-destinations/:id/secret", d.JSON, h.RotateWorkIntegrationSecret)
	routes.Handle(http.MethodGet, "/v1/publication/destinations", d.JSON, h.ListBlogIntegrations)
	routes.Handle(http.MethodPost, "/v1/publication/destinations", d.JSON, h.AddBlogIntegration)
	routes.Handle(http.MethodDelete, "/v1/publication/destinations/:id", d.JSON, h.RemoveBlogIntegration)
	routes.Handle(http.MethodPatch, "/v1/publication/destinations/:id", d.JSON, h.UpdateBlogIntegration)
	routes.Handle(http.MethodDelete, "/v1/publication/destinations/:id/verification", d.JSON, h.DisableBlogIntegration)
	routes.Handle(http.MethodPost, "/v1/publication/destinations/:id/verification", d.Verify, h.VerifyBlogIntegration)
	routes.Handle(http.MethodPost, "/v1/publication/destinations/:id/secret", d.JSON, h.RotateBlogIntegrationSecret)
	routes.Handle(http.MethodPost, "/v1/publication/channels", d.Verify, h.AddBlogChannel)
	routes.Handle(http.MethodPatch, "/v1/publication/channels/:id", d.Verify, h.UpdateBlogChannel)
	routes.Handle(http.MethodGet, "/v1/publication/deliveries", d.JSON, h.ListBlogAnnouncementAttempts)
	routes.Handle(http.MethodGet, "/v1/publication/deliveries/:id/attempts", d.JSON, h.ListBlogAnnouncementTries)
	routes.Handle(http.MethodPost, "/v1/publication/deliveries/:id/replay", d.JSON, h.ReplayBlogAnnouncementAttempt)
	routes.Handle(http.MethodPost, "/v1/publication/deliveries/:id/repair", d.Verify, h.RepairDiscordAnnouncement)
	routes.Handle(http.MethodPut, "/v1/publication/grants/:id/destinations", d.JSON, h.SetBlogGrantIntegrations)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/destinations", d.JSON, h.ListPostIntegrations)
	routes.Handle(http.MethodGet, "/v1/publication/posts/:id/deliveries", d.JSON, h.ListPostAnnouncementAttempts)
}

// The field names an integration and an announcement attempt answered to before the renames, kept for sixty days
var (
	integrationTypeAliases = map[string]string{"type": "kind"}
	integrationAliases     = map[string]string{"type": "kind", "announcements": "events"}
	integrationListAliases = map[string]string{"integrations": "destinations"}
	addedAliases           = map[string]string{"integration": "destination"}
	defaultsAliases        = map[string]string{"integrationIds": "destinationIds"}
	policyAliases          = map[string]string{
		"integrationIds": "destinationIds", "defaultIntegrationIds": "defaultDestinationIds",
	}
	attemptListAliases = map[string]string{"attempts": "announcements"}
	attemptAliases     = map[string]string{
		"type": "kind", "integration": "destination", "announcementId": "eventId",
		"versionId": "updateId", "versionNumber": "updateNumber", "tries": "attempts",
	}
	blogAttemptListAliases = map[string]string{"attempts": "deliveries"}
	blogAttemptAliases     = map[string]string{
		"type": "kind", "integration": "destination", "announcementId": "eventId",
		"announcementType": "eventType", "tries": "attempts",
	}
	tryListAliases  = map[string]string{"tries": "attempts"}
	sentWorkAliases = map[string]string{"type": "kind"}
	sentAliases     = map[string]string{"work": "asset"}
)

func (d WorkIntegration) MarshalJSON() ([]byte, error) {
	type plain WorkIntegration
	return api.MarshalAliased(plain(d), integrationTypeAliases)
}

func (l WorkIntegrationList) MarshalJSON() ([]byte, error) {
	type plain WorkIntegrationList
	return api.MarshalAliased(plain(l), integrationListAliases)
}

func (a AddedWorkIntegration) MarshalJSON() ([]byte, error) {
	type plain AddedWorkIntegration
	return api.MarshalAliased(plain(a), addedAliases)
}

func (c WorkIntegrationChoice) MarshalJSON() ([]byte, error) {
	type plain WorkIntegrationChoice
	return api.MarshalAliased(plain(c), integrationTypeAliases)
}

func (c WorkIntegrationChoices) MarshalJSON() ([]byte, error) {
	type plain WorkIntegrationChoices
	return api.MarshalAliased(plain(c), integrationListAliases)
}

func (r *WorkIntegrationDefaultsRequest) UnmarshalJSON(data []byte) error {
	type plain WorkIntegrationDefaultsRequest
	return api.UnmarshalAliased(data, (*plain)(r), defaultsAliases)
}

func (a WorkAnnouncementAttempt) MarshalJSON() ([]byte, error) {
	type plain WorkAnnouncementAttempt
	return api.MarshalAliased(plain(a), attemptAliases)
}

func (l WorkAnnouncementAttemptList) MarshalJSON() ([]byte, error) {
	type plain WorkAnnouncementAttemptList
	return api.MarshalAliased(plain(l), attemptListAliases)
}

func (d BlogIntegration) MarshalJSON() ([]byte, error) {
	type plain BlogIntegration
	return api.MarshalAliased(plain(d), integrationAliases)
}

func (l BlogIntegrationList) MarshalJSON() ([]byte, error) {
	type plain BlogIntegrationList
	return api.MarshalAliased(plain(l), integrationListAliases)
}

func (l BlogIntegrationChoiceList) MarshalJSON() ([]byte, error) {
	type plain BlogIntegrationChoiceList
	return api.MarshalAliased(plain(l), integrationListAliases)
}

func (a AddedBlogIntegration) MarshalJSON() ([]byte, error) {
	type plain AddedBlogIntegration
	return api.MarshalAliased(plain(a), addedAliases)
}

func (r RotatedBlogSecret) MarshalJSON() ([]byte, error) {
	type plain RotatedBlogSecret
	return api.MarshalAliased(plain(r), addedAliases)
}

func (r *AddBlogIntegrationRequest) UnmarshalJSON(data []byte) error {
	type plain AddBlogIntegrationRequest
	return api.UnmarshalAliased(data, (*plain)(r), integrationAliases)
}

func (r *UpdateBlogIntegrationRequest) UnmarshalJSON(data []byte) error {
	type plain UpdateBlogIntegrationRequest
	return api.UnmarshalAliased(data, (*plain)(r), integrationAliases)
}

func (r *IntegrationPolicyRequest) UnmarshalJSON(data []byte) error {
	type plain IntegrationPolicyRequest
	return api.UnmarshalAliased(data, (*plain)(r), policyAliases)
}

func (d BlogAnnouncementAttempt) MarshalJSON() ([]byte, error) {
	type plain BlogAnnouncementAttempt
	return api.MarshalAliased(plain(d), blogAttemptAliases)
}

func (l BlogAnnouncementAttemptList) MarshalJSON() ([]byte, error) {
	type plain BlogAnnouncementAttemptList
	return api.MarshalAliased(plain(l), blogAttemptListAliases)
}

func (l BlogAnnouncementTryList) MarshalJSON() ([]byte, error) {
	type plain BlogAnnouncementTryList
	return api.MarshalAliased(plain(l), tryListAliases)
}

func (r *AddWorkIntegrationRequest) UnmarshalJSON(data []byte) error {
	type plain AddWorkIntegrationRequest
	return api.UnmarshalAliased(data, (*plain)(r), integrationTypeAliases)
}

func (s sent) MarshalJSON() ([]byte, error) {
	type plain sent
	return api.MarshalAliased(plain(s), sentAliases)
}

func (w sentWork) MarshalJSON() ([]byte, error) {
	type plain sentWork
	return api.MarshalAliased(plain(w), sentWorkAliases)
}
