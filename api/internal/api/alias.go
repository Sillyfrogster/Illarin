package api

import (
	"bytes"
	"encoding/json"
)

// MarshalAliased writes value as JSON and repeats each field under the old name it had before the rename
func MarshalAliased(value any, aliases map[string]string) ([]byte, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(encoded, &fields); err != nil {
		return nil, err
	}
	for name, alias := range aliases {
		if field, present := fields[name]; present {
			fields[alias] = field
		}
	}
	return json.Marshal(fields)
}

// UnmarshalAliased reads JSON into destination, taking each field under its old name as well as its new one
func UnmarshalAliased(data []byte, destination any, aliases map[string]string) error {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return nil
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	for name, alias := range aliases {
		field, present := fields[alias]
		if !present {
			continue
		}
		delete(fields, alias)
		if _, renamed := fields[name]; !renamed {
			fields[name] = field
		}
	}
	encoded, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	return decoder.Decode(destination)
}

// Alias lets a query value arrive under the name it had before the rename
func (q *Query) Alias(name, alias string) {
	if _, renamed := q.values[name]; renamed {
		return
	}
	if values, present := q.values[alias]; present {
		q.values[name] = values
	}
}
