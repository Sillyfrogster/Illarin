package keys

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func Take[T any](source map[string]json.RawMessage, key string, target *T) bool {
	raw, present := source[key]
	if !present || IsNull(raw) || json.Unmarshal(raw, target) != nil {
		return false
	}
	delete(source, key)
	return true
}

func IsNull(raw json.RawMessage) bool {
	return bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func WriteIfSet(target map[string]json.RawMessage, key string, set bool, value any) {
	if set {
		target[key] = Must(value)
	}
}

func MergeAbsent(target, source map[string]json.RawMessage) {
	for key, value := range source {
		if _, written := target[key]; !written {
			target[key] = value
		}
	}
}

func Object(raw json.RawMessage) map[string]json.RawMessage {
	object := make(map[string]json.RawMessage)
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &object)
	}
	return object
}

func Must(value any) json.RawMessage {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("write a JSON value: %v", err))
	}
	return encoded
}
