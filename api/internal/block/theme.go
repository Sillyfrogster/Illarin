package block

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

type ColorSet struct {
	Modes []ColorMode `json:"modes"`
}

type ColorMode struct {
	Name   string  `json:"name,omitempty"`
	Colors []Color `json:"colors"`
}

type Color struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Value string    `json:"value"`
}

func (s ColorSet) Empty() bool {
	for _, mode := range s.Modes {
		for _, color := range mode.Colors {
			if color.Value != "" {
				return false
			}
		}
	}
	return true
}

type StylesheetSet struct {
	Global      string           `json:"global"`
	Stylesheets []Stylesheet     `json:"stylesheets"`
	Files       []StylesheetFile `json:"assets"`
}

type Stylesheet struct {
	ID      uuid.UUID `json:"id"`
	Name    string    `json:"name"`
	CSS     string    `json:"css"`
	Enabled bool      `json:"enabled"`
}

type StylesheetFile struct {
	ID        uuid.UUID `json:"id"`
	Path      string    `json:"path"`
	MediaType string    `json:"mediaType,omitempty"`
	Data      []byte    `json:"data"`
}

func (s StylesheetSet) Empty() bool {
	if s.Global != "" || len(s.Files) > 0 {
		return false
	}
	for _, sheet := range s.Stylesheets {
		if sheet.CSS != "" {
			return false
		}
	}
	return true
}

func decodeColorSet(raw json.RawMessage) (Content, error) {
	var incoming struct {
		Modes *[]struct {
			Name   string `json:"name,omitempty"`
			Colors *[]struct {
				ID    uuid.UUID `json:"id,omitempty"`
				Name  *string   `json:"name"`
				Value *string   `json:"value"`
			} `json:"colors"`
		} `json:"modes"`
	}
	if err := decodeContentJSON(raw, &incoming); err != nil {
		return nil, err
	}
	if incoming.Modes == nil {
		return nil, fmt.Errorf("modes must be present as a list")
	}
	modes := make([]ColorMode, len(*incoming.Modes))
	for i, mode := range *incoming.Modes {
		if mode.Colors == nil {
			return nil, fmt.Errorf("mode %d must include colours as a list", i+1)
		}
		colors := make([]Color, len(*mode.Colors))
		for j, color := range *mode.Colors {
			if color.Name == nil || color.Value == nil {
				return nil, fmt.Errorf("mode %d colour %d must include name and value as strings", i+1, j+1)
			}
			colors[j] = Color{
				ID: itemID(color.ID), Name: *color.Name, Value: *color.Value,
			}
		}
		modes[i] = ColorMode{Name: mode.Name, Colors: colors}
	}
	return ColorSet{Modes: modes}, nil
}

func decodeStylesheetSet(raw json.RawMessage) (Content, error) {
	var incoming struct {
		Global      *string           `json:"global"`
		Stylesheets *[]Stylesheet     `json:"stylesheets"`
		Files       *[]StylesheetFile `json:"assets"`
	}
	if err := decodeContentJSON(raw, &incoming); err != nil {
		return nil, err
	}
	if incoming.Global == nil || incoming.Stylesheets == nil || incoming.Files == nil {
		return nil, fmt.Errorf("global, stylesheets and files must be present")
	}
	stylesheets := *incoming.Stylesheets
	stylesheetNames := make(map[string]struct{}, len(stylesheets))
	for i := range stylesheets {
		if _, duplicate := stylesheetNames[stylesheets[i].Name]; duplicate {
			return nil, fmt.Errorf("stylesheet %d repeats the name %q", i+1, stylesheets[i].Name)
		}
		stylesheetNames[stylesheets[i].Name] = struct{}{}
		stylesheets[i].ID = itemID(stylesheets[i].ID)
	}
	files := *incoming.Files
	filePaths := make(map[string]struct{}, len(files))
	for i := range files {
		if files[i].Path == "" {
			return nil, fmt.Errorf("file %d must include a path", i+1)
		}
		if _, duplicate := filePaths[files[i].Path]; duplicate {
			return nil, fmt.Errorf("file %d repeats the path %q", i+1, files[i].Path)
		}
		filePaths[files[i].Path] = struct{}{}
		files[i].ID = itemID(files[i].ID)
	}
	return StylesheetSet{
		Global: *incoming.Global, Stylesheets: stylesheets, Files: files,
	}, nil
}
