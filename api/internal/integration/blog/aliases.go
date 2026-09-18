package blog

import "github.com/Sillyfrogster/Illarin/api/internal/api"

// The field name a destination choice answered to before the rename, kept for sixty days
var destinationChoiceAliases = map[string]string{"type": "kind"}

func (c PublicationDestinationChoice) MarshalJSON() ([]byte, error) {
	type plain PublicationDestinationChoice
	return api.MarshalAliased(plain(c), destinationChoiceAliases)
}
