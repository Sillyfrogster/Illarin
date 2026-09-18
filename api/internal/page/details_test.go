package page_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/page"
)

func startDetailsWork(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	workType string,
) apitest.StartedWork {
	t.Helper()
	body := fmt.Sprintf(`{"type":%q}`, workType)
	if workType == "preset" || workType == "theme" {
		body = fmt.Sprintf(`{"type":%q,"app":"lumiverse"}`, workType)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/works", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := apitest.Send(t, r, apitest.Authorized(request, session))
	if response.Code != http.StatusCreated {
		t.Fatalf("start a %s: status = %d, want 201: %s", workType, response.Code, response.Body.String())
	}
	var started apitest.StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode the started %s: %v", workType, err)
	}
	return started
}

func TestCreatorCanAddReplaceAndClearABlurbForEveryWorkType(t *testing.T) {
	t.Parallel()
	for _, workType := range []string{"character", "lorebook", "preset", "theme", "pack"} {
		t.Run(workType, func(t *testing.T) {
			r, session := harness.NewVerifiedRouter(t)
			started := startDetailsWork(t, r, session, workType)

			for _, blurb := range []string{
				"A short pitch for a person.",
				"A replacement pitch with a clearer promise.",
				"",
			} {
				body, err := json.Marshal(map[string]any{
					"blurb": blurb, "isNsfw": false, "name": "Catalog name",
				})
				if err != nil {
					t.Fatal(err)
				}
				response := apitest.SaveDetails(t, r, session, started.ID, string(body))
				if response.Code != http.StatusNoContent {
					t.Fatalf("save blurb %q: status = %d, want 204: %s", blurb, response.Code, response.Body.String())
				}
				if saved := apitest.FetchStartedWork(t, r, session, started.ID); saved.Blurb != blurb {
					t.Fatalf("saved blurb = %q, want %q", saved.Blurb, blurb)
				}
			}
		})
	}
}

func TestBlurbLimitCountsUnicodeCharactersAndNeverTruncates(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	accepted := strings.Repeat("界", 400)
	tooLong := accepted + "界"

	detailsBody := func(blurb string) string {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"blurb": blurb, "isNsfw": false, "name": "Catalog name",
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	if response := apitest.SaveDetails(t, r, session, started.ID, detailsBody(accepted)); response.Code != http.StatusNoContent {
		t.Fatalf("save 400-character blurb: status = %d, want 204: %s", response.Code, response.Body.String())
	}

	response := apitest.SaveDetails(t, r, session, started.ID, detailsBody(tooLong))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("save 401-character blurb: status = %d, want 400: %s", response.Code, response.Body.String())
	}
	var refusal struct {
		Error string `json:"error"`
		Field string `json:"field"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &refusal); err != nil {
		t.Fatal(err)
	}
	if refusal.Field != "blurb" || !strings.Contains(refusal.Error, "400") {
		t.Fatalf("refusal = %+v, want the blurb field and its limit", refusal)
	}
	if saved := apitest.FetchStartedWork(t, r, session, started.ID); saved.Blurb != accepted {
		t.Fatalf("saved blurb has %d characters, want the intact 400-character value", len([]rune(saved.Blurb)))
	}
}

func TestDetailsRequestMustSayWhetherToKeepOrClearTheBlurb(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	if response := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"Catalog name","blurb":"Keep this pitch","isNsfw":false}`); response.Code != http.StatusNoContent {
		t.Fatalf("save the starting blurb: %d %s", response.Code, response.Body.String())
	}

	response := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"Renamed without the new field","isNsfw":false}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("omit the blurb: status = %d, want 400: %s", response.Code, response.Body.String())
	}
	if saved := apitest.FetchStartedWork(t, r, session, started.ID); saved.Blurb != "Keep this pitch" {
		t.Fatalf("omitted blurb changed the saved value to %q", saved.Blurb)
	}
}

func TestAWithheldWorkRefusesNewDetailsWithTheFrozenCode(t *testing.T) {
	t.Parallel()
	_, r, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	workID := apitest.PublishedCharacter(t, r, session)
	if _, err := pool.Exec(context.Background(),
		`update users set role = 'admin' where username = 'verified.creator'`); err != nil {
		t.Fatalf("make the creator an admin: %v", err)
	}
	if withheld := apitest.Send(t, r, apitest.AuthorizedJSONRequest(t, http.MethodPut,
		"/v1/works/"+workID+"/withhold", `{"reason":"Report under review"}`, session)); withheld.Code != http.StatusNoContent {
		t.Fatalf("withhold status = %d, want 204: %s", withheld.Code, withheld.Body.String())
	}

	response := apitest.SaveDetails(t, r, session, workID,
		`{"name":"Renamed while withheld","blurb":"","isNsfw":false}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %s", response.Code, response.Body.String())
	}
	var refusal page.CandidateConflict
	if err := json.Unmarshal(response.Body.Bytes(), &refusal); err != nil {
		t.Fatal(err)
	}
	if refusal.Code != page.CandidateConflictCodeWorkFrozen {
		t.Fatalf("code = %q, want %q", refusal.Code, page.CandidateConflictCodeWorkFrozen)
	}
}
