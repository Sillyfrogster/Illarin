package block

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type EntryTable struct {
	Entries []Entry `json:"entries"`
}

type Entry struct {
	ID            uuid.UUID      `json:"id"`
	Name          string         `json:"name,omitempty"`
	Keys          []string       `json:"keys"`
	SecondaryKeys []string       `json:"secondaryKeys,omitempty"`
	Selective     bool           `json:"selective,omitempty"`
	CaseSensitive bool           `json:"caseSensitive,omitempty"`
	Constant      bool           `json:"constant,omitempty"`
	Enabled       bool           `json:"enabled"`
	Order         int            `json:"order"`
	Position      EntryPosition  `json:"position,omitempty"`
	Recursion     EntryRecursion `json:"recursion"`
	Text          string         `json:"text"`
}

type EntryPosition string

const (
	BeforeCharacter EntryPosition = "before_character"
	AfterCharacter  EntryPosition = "after_character"
)

func (p EntryPosition) Known() bool {
	return p == "" || p == BeforeCharacter || p == AfterCharacter
}

func EntryPositions() []EntryPosition {
	return []EntryPosition{BeforeCharacter, AfterCharacter}
}

type EntryRecursion struct {
	Exclude    bool `json:"exclude,omitempty"`
	Prevent    bool `json:"prevent,omitempty"`
	DelayUntil bool `json:"delayUntil,omitempty"`
}

func (t EntryTable) Empty() bool {
	for _, entry := range t.Entries {
		if entry.Text != "" {
			return false
		}
	}
	return true
}

func decodeEntryTable(raw json.RawMessage) (Content, error) {
	var incoming struct {
		Entries *[]struct {
			ID            uuid.UUID      `json:"id,omitempty"`
			Name          string         `json:"name,omitempty"`
			Keys          *[]string      `json:"keys"`
			SecondaryKeys []string       `json:"secondaryKeys,omitempty"`
			Selective     bool           `json:"selective,omitempty"`
			CaseSensitive bool           `json:"caseSensitive,omitempty"`
			Constant      bool           `json:"constant,omitempty"`
			Enabled       *bool          `json:"enabled"`
			Order         int            `json:"order"`
			Position      EntryPosition  `json:"position,omitempty"`
			Recursion     EntryRecursion `json:"recursion"`
			Text          *string        `json:"text"`
		} `json:"entries"`
	}
	if err := decodeContentJSON(raw, &incoming); err != nil {
		return nil, err
	}
	if incoming.Entries == nil {
		return nil, fmt.Errorf("entries must be present as a list")
	}
	entries := make([]Entry, len(*incoming.Entries))
	for i, item := range *incoming.Entries {
		if item.Keys == nil || item.Text == nil || item.Enabled == nil {
			return nil, fmt.Errorf(
				"entry %d must include keys as a list, text as a string and enabled as a yes or no",
				i+1,
			)
		}
		if !item.Position.Known() {
			return nil, fmt.Errorf(
				"entry %d sits at %q. Choose before or after the character before saving",
				i+1, item.Position,
			)
		}
		entries[i] = Entry{
			ID:            itemID(item.ID),
			Name:          item.Name,
			Keys:          *item.Keys,
			SecondaryKeys: item.SecondaryKeys,
			Selective:     item.Selective,
			CaseSensitive: item.CaseSensitive,
			Constant:      item.Constant,
			Enabled:       *item.Enabled,
			Order:         item.Order,
			Position:      item.Position,
			Recursion:     item.Recursion,
			Text:          *item.Text,
		}
	}
	return EntryTable{Entries: entries}, nil
}
