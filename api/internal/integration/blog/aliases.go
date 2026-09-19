package blog

import "github.com/Sillyfrogster/Illarin/api/internal/api"

// The field names an integration choice answered to before the renames, kept for sixty days
var integrationChoiceAliases = map[string]string{"type": "kind", "announcements": "events"}

func (c BlogIntegrationChoice) MarshalJSON() ([]byte, error) {
	type plain BlogIntegrationChoice
	return api.MarshalAliased(plain(c), integrationChoiceAliases)
}
