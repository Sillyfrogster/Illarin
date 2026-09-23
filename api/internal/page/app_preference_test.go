package page_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/Sillyfrogster/Illarin/api/internal/format/character"
	"github.com/Sillyfrogster/Illarin/api/internal/format/theme"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

type typeCount struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

type appReading struct {
	App      *string     `json:"app"`
	Types    []typeCount `json:"types"`
	AllTypes int         `json:"allTypes"`
	Items    []struct {
		Name string   `json:"name"`
		Apps []string `json:"apps"`
	} `json:"items"`
	Apps []browseOption `json:"apps"`
}

func (r appReading) names() []string {
	names := []string{}
	for _, item := range r.Items {
		names = append(names, item.Name)
	}
	return names
}

func (r appReading) types() []string {
	types := []string{}
	for _, one := range r.Types {
		types = append(types, fmt.Sprintf("%s:%d", one.Value, one.Count))
	}
	return types
}

func (r appReading) app() string {
	if r.App == nil {
		return "unset"
	}
	return *r.App
}

func readApp(t *testing.T, router http.Handler, request *http.Request) appReading {
	t.Helper()
	response := apitest.Send(t, router, request)
	if response.Code != http.StatusOK {
		t.Fatalf("browse %s status = %d: %s", request.URL, response.Code, response.Body.String())
	}
	var reading appReading
	if err := json.Unmarshal(response.Body.Bytes(), &reading); err != nil {
		t.Fatalf("decode browse response: %v", err)
	}
	return reading
}

func browseRequest(path string, cookies ...*http.Cookie) *http.Request {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	return request
}

// oneAppThemeRouter holds a card and a theme in a registry where only SillyTavern reads themes
func oneAppThemeRouter(t *testing.T) (http.Handler, *http.Cookie) {
	t.Helper()
	registry := format.NewRegistry()
	for _, module := range append(character.Modules(), theme.SillyTavernModule{}) {
		if err := registry.Register(module); err != nil {
			t.Fatalf("register %s: %v", module.ID(), err)
		}
	}
	router, session, works := harness.NewVerifiedUploadRouter(t, registry)
	card := apitest.ExampleMetadata("Ana")
	card["filename"] = "ana.json"
	apitest.UploadAndFinish(t, router, session, works, card, []byte(`{
		"spec":"chara_card_v3","spec_version":"3.0",
		"data":{"name":"Ana","description":"Keeps the archive.","first_mes":"Welcome back."}
	}`))
	palette := apitest.ExampleMetadata("Midnight violet")
	palette["filename"] = "midnight-violet.json"
	apitest.UploadAndFinish(t, router, session, works, palette, []byte(`{
		"name":"Midnight violet",
		"main_text_color":"rgba(244,241,246,1)",
		"italics_text_color":"rgba(214,203,236,1)",
		"quote_text_color":"rgba(185,179,192,1)",
		"blur_tint_color":"rgba(13,12,17,.9)",
		"chat_tint_color":"rgba(21,20,27,.94)",
		"user_mes_blur_tint_color":"rgba(35,30,45,.92)",
		"bot_mes_blur_tint_color":"rgba(25,23,32,.92)",
		"shadow_color":"rgba(0,0,0,.4)",
		"border_color":"rgba(55,52,64,1)",
		"blur_strength":8,
		"shadow_width":2,
		"font_scale":1,
		"chat_display":1,
		"chat_width":56,
		"custom_css":"body { letter-spacing: .01em; }"
	}`))
	return router, session
}

func TestWithNoAppChosenBrowseMarksEveryWorkAndHoldsBackOneAppTypes(t *testing.T) {
	t.Parallel()
	router, _ := oneAppThemeRouter(t)

	unset := readApp(t, router, browseRequest("/v1/works"))
	if unset.app() != "unset" || !slices.Equal(unset.names(), []string{"Ana"}) {
		t.Fatalf("a reader with no app saw %v under app %s, want Ana alone", unset.names(), unset.app())
	}
	if !slices.Equal(unset.Items[0].Apps, []string{"sillytavern", "risu", "lumiverse"}) {
		t.Errorf("Ana is marked with %v, want every app that reads a card", unset.Items[0].Apps)
	}
	if !slices.Contains(unset.types(), "character:1") || slices.Contains(unset.types(), "theme:1") || unset.AllTypes != 1 {
		t.Errorf("types in view = %v of %d, want one character and no themes", unset.types(), unset.AllTypes)
	}
	for _, option := range unset.Apps {
		if option.Value == "sillytavern" && option.Count != 2 {
			t.Errorf("SillyTavern counts %d, want the theme it would show as well", option.Count)
		}
	}

	asked := readApp(t, router, browseRequest("/v1/works?type=theme"))
	if !slices.Equal(asked.names(), []string{"Midnight violet"}) {
		t.Errorf("asking for themes returned %v, want the theme", asked.names())
	}
	if !slices.Contains(asked.types(), "character:1") || !slices.Contains(asked.types(), "theme:1") || asked.AllTypes != 1 {
		t.Errorf("asking for themes offered %v of %d, want the theme beside the card", asked.types(), asked.AllTypes)
	}

	unknown := readApp(t, router, browseRequest("/v1/works?app=notepad"))
	if unknown.app() != "unset" || !slices.Equal(unknown.names(), []string{"Ana"}) {
		t.Errorf("an app Illarin does not know gave %v under %s, want it ignored", unknown.names(), unknown.app())
	}
}

func TestAChosenAppNarrowsTheFeedAndTheTypesToWhatItReads(t *testing.T) {
	t.Parallel()
	router, _ := oneAppThemeRouter(t)

	tavern := readApp(t, router, browseRequest("/v1/works?app=sillytavern"))
	if tavern.app() != "sillytavern" || !slices.Equal(tavern.names(), []string{"Midnight violet", "Ana"}) {
		t.Errorf("SillyTavern saw %v, want the theme and the card", tavern.names())
	}
	if !slices.Equal(tavern.types(), []string{"character:1", "theme:1"}) || tavern.AllTypes != 2 {
		t.Errorf("SillyTavern types = %v of %d, want a character and a theme", tavern.types(), tavern.AllTypes)
	}

	risu := readApp(t, router, browseRequest("/v1/works?app=risu"))
	if !slices.Equal(risu.names(), []string{"Ana"}) || !slices.Equal(risu.types(), []string{"character:1"}) {
		t.Errorf("RisuAI saw %v with types %v, want Ana and characters only", risu.names(), risu.types())
	}

	remembered := &http.Cookie{Name: page.AppCookie, Value: "sillytavern"}
	signedOut := readApp(t, router, browseRequest("/v1/works", remembered))
	if signedOut.app() != "sillytavern" || len(signedOut.names()) != 2 {
		t.Errorf("the browser's app gave %v under %s, want SillyTavern's feed", signedOut.names(), signedOut.app())
	}
	everything := readApp(t, router, browseRequest("/v1/works?app=any", remembered))
	if everything.app() != "any" || !slices.Equal(everything.names(), []string{"Ana"}) {
		t.Errorf("any app gave %v under %s, want the shared feed", everything.names(), everything.app())
	}
}

func TestASignedInReadersAppIsKeptOnTheAccount(t *testing.T) {
	t.Parallel()
	router, session := oneAppThemeRouter(t)

	refused := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/app-preference", `{"app":"notepad"}`, session,
	))
	if refused.Code != http.StatusBadRequest {
		t.Errorf("an unknown app saved with %d, want 400", refused.Code)
	}
	saved := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
		t, http.MethodPut, "/v1/account/app-preference", `{"app":"risu"}`, session,
	))
	if saved.Code != http.StatusNoContent {
		t.Fatalf("save app status = %d, want 204: %s", saved.Code, saved.Body.String())
	}

	remembered := &http.Cookie{Name: page.AppCookie, Value: "sillytavern"}
	reading := readApp(t, router, browseRequest("/v1/works", session, remembered))
	if reading.app() != "risu" || !slices.Equal(reading.names(), []string{"Ana"}) {
		t.Errorf("signed in, browse used %s and showed %v, want the account's RisuAI", reading.app(), reading.names())
	}

	answer := apitest.Send(t, router, browseRequest("/v1/account/preferences", session))
	var preferences struct {
		App            *string `json:"app"`
		NsfwPreference string  `json:"nsfwPreference"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &preferences); err != nil {
		t.Fatalf("decode preferences: %v", err)
	}
	if preferences.App == nil || *preferences.App != "risu" || preferences.NsfwPreference != "blurred" {
		t.Errorf("preferences = %+v, want RisuAI and blurred", preferences)
	}
}
