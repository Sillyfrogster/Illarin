package upload_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func startTheme(t *testing.T, r http.Handler, session *http.Cookie, app string) apitest.StartedWork {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/works",
		strings.NewReader(`{"type":"theme","app":"`+app+`"}`))
	request.Header.Set("Content-Type", "application/json")
	response := apitest.Send(t, r, apitest.Authorized(request, session))
	if response.Code != http.StatusCreated {
		t.Fatalf("start a theme for %s: status = %d, want 201: %s",
			app, response.Code, response.Body.String())
	}
	var started apitest.StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode the started theme: %v", err)
	}
	return started
}

func TestAThemeAsksWhichAppsNamesItUsesAndSeedsThoseNames(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)

	request := httptest.NewRequest(http.MethodPost, "/v1/works",
		strings.NewReader(`{"type":"theme"}`))
	request.Header.Set("Content-Type", "application/json")
	response := apitest.Send(t, r, apitest.Authorized(request, session))
	if response.Code != http.StatusBadRequest ||
		!strings.Contains(response.Body.String(), "SillyTavern") ||
		!strings.Contains(response.Body.String(), "Lumiverse") {
		t.Fatalf("theme without an app = %d %s, want a refusal naming both apps",
			response.Code, response.Body.String())
	}

	for _, app := range []string{"sillytavern", "lumiverse"} {
		started := startTheme(t, r, session, app)
		core := apitest.BlockNamed(t, started.Blocks, "theme_core")
		stylesheet := apitest.BlockNamed(t, started.Blocks, "stylesheet")
		if !core.Required || core.Hideable || core.Layout != "duo" || core.Width != "full" {
			t.Errorf("%s theme core = %+v, want the fixed full-width palette", app, core)
		}
		if !stylesheet.Required || !stylesheet.Hideable || stylesheet.Layout != "single" {
			t.Errorf("%s stylesheet = %+v, want a required, hideable single block", app, stylesheet)
		}

		var colours struct {
			Modes []struct {
				Colors []struct {
					Name string `json:"name"`
				} `json:"colors"`
			} `json:"modes"`
		}
		if err := json.Unmarshal(core.Elements[0].Content, &colours); err != nil {
			t.Fatalf("decode %s palette: %v", app, err)
		}
		if len(colours.Modes) == 0 || len(colours.Modes[0].Colors) == 0 {
			t.Errorf("%s palette arrived with no colour names", app)
		}
		var controls struct {
			Settings []struct {
				Name string `json:"name"`
			} `json:"settings"`
		}
		if err := json.Unmarshal(core.Elements[1].Content, &controls); err != nil {
			t.Fatalf("decode %s controls: %v", app, err)
		}
		if len(controls.Settings) == 0 {
			t.Errorf("%s theme arrived with no control names", app)
		}
	}
}
