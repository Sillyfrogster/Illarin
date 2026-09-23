package block

import (
	"encoding/json"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
)

// The field name an element answered to before the rename, kept for sixty days
var workElementAliases = map[string]string{"fromFile": "locked"}

func (e WorkElement) MarshalJSON() ([]byte, error) {
	type plain WorkElement
	return api.MarshalAliased(plain(e), workElementAliases)
}

// aliasPromptFragments reads the name a private fragment had before the rename, for sixty days
func aliasPromptFragments(raw json.RawMessage) json.RawMessage {
	var list map[string]json.RawMessage
	if json.Unmarshal(raw, &list) != nil {
		return raw
	}
	var fragments []map[string]json.RawMessage
	if json.Unmarshal(list["fragments"], &fragments) != nil {
		return raw
	}
	for _, fragment := range fragments {
		if old, present := fragment["protected"]; present {
			delete(fragment, "protected")
			if _, renamed := fragment["private"]; !renamed {
				fragment["private"] = old
			}
		}
	}
	encoded, err := json.Marshal(fragments)
	if err != nil {
		return raw
	}
	list["fragments"] = encoded
	aliased, err := json.Marshal(list)
	if err != nil {
		return raw
	}
	return aliased
}
