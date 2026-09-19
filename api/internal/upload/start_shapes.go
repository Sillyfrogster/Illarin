package upload

import "github.com/Sillyfrogster/Illarin/api/internal/page"

type StartWorkRequest struct {
	App  *string `json:"app,omitempty"`
	Type string  `json:"type"`
}

type BuildChoices struct {
	Types []BuildChoice `json:"types"`
}

type BuildChoice struct {
	Type string         `json:"type"`
	Apps []page.AppName `json:"apps"`
}
