package blog

import "github.com/Sillyfrogster/Illarin/api/internal/api"

// The field names a post's requests and a contributor's approval answered to before the rename, kept for sixty days
var (
	postIntegrationAliases = map[string]string{
		"integrationIds": "destinationIds", "roleIntegrationIds": "roleDestinationIds",
	}
	grantIntegrationAliases = map[string]string{
		"integrations": "destinations", "integrationsInherited": "destinationsInherited",
	}
	appIntegrationAliases = map[string]string{"integrations": "destinations"}
)

func (r *PublishPostRequest) UnmarshalJSON(data []byte) error {
	type plain PublishPostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *RepublishPostRequest) UnmarshalJSON(data []byte) error {
	type plain RepublishPostRequest
	return api.UnmarshalAliasedLoosely(data, (*plain)(r), postIntegrationAliases)
}

func (r *WithdrawPostRequest) UnmarshalJSON(data []byte) error {
	type plain WithdrawPostRequest
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

func (g PublicationGrant) MarshalJSON() ([]byte, error) {
	type plain PublicationGrant
	return api.MarshalAliased(plain(g), grantIntegrationAliases)
}

func (a PublicationApp) MarshalJSON() ([]byte, error) {
	type plain PublicationApp
	return api.MarshalAliased(plain(a), appIntegrationAliases)
}
