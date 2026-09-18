package apitest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ExtensionPage struct {
	Type                  string   `json:"type"`
	Name                  string   `json:"name"`
	Identifier            *string  `json:"identifier"`
	InstalledAppVersions  []string `json:"installedAppVersions"`
	ExtensionDependencies []struct {
		Name  string `json:"name"`
		Works []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Creator string `json:"creator"`
		} `json:"assets"`
	} `json:"extensionDependencies"`
	Blocks []struct {
		ID         string `json:"id"`
		Definition string `json:"definition"`
		Elements   []struct {
			Role    string          `json:"role"`
			Pinned  bool            `json:"pinned"`
			Locked  bool            `json:"locked"`
			Content json.RawMessage `json:"content"`
		} `json:"elements"`
	} `json:"blocks"`
}

func ReadExtensionPage(t *testing.T, r http.Handler, session *http.Cookie, workID string) ExtensionPage {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	if session != nil {
		request = Authorized(request, session)
	}
	response := Send(t, r, request)
	if response.Code != http.StatusOK {
		t.Fatalf("read the extension = %d: %s", response.Code, response.Body.String())
	}
	var page ExtensionPage
	if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the extension: %v", err)
	}
	return page
}
