package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func startIdentityAsset(
	t *testing.T,
	r http.Handler,
	session *http.Cookie,
	kind string,
) startedAsset {
	t.Helper()
	body := fmt.Sprintf(`{"kind":%q}`, kind)
	if kind == "preset" || kind == "theme" {
		body = fmt.Sprintf(`{"kind":%q,"app":"lumiverse"}`, kind)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/assets", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := send(t, r, authorized(request, session))
	if response.Code != http.StatusCreated {
		t.Fatalf("start a %s: status = %d, want 201: %s", kind, response.Code, response.Body.String())
	}
	var started startedAsset
	if err := json.Unmarshal(response.Body.Bytes(), &started); err != nil {
		t.Fatalf("decode the started %s: %v", kind, err)
	}
	return started
}

func TestCreatorCanAddReplaceAndClearABlurbForEveryAssetKind(t *testing.T) {
	for _, kind := range []string{"character", "lorebook", "preset", "theme", "pack"} {
		t.Run(kind, func(t *testing.T) {
			r, session := newVerifiedTestRouter(t)
			started := startIdentityAsset(t, r, session, kind)

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
				response := saveIdentity(t, r, session, started.ID, string(body))
				if response.Code != http.StatusNoContent {
					t.Fatalf("save blurb %q: status = %d, want 204: %s", blurb, response.Code, response.Body.String())
				}
				if saved := fetchStartedAsset(t, r, session, started.ID); saved.Blurb != blurb {
					t.Fatalf("saved blurb = %q, want %q", saved.Blurb, blurb)
				}
			}
		})
	}
}

func TestBlurbLimitCountsUnicodeCharactersAndNeverTruncates(t *testing.T) {
	r, session := newVerifiedTestRouter(t)
	started := startCharacter(t, r, session)
	accepted := strings.Repeat("界", 400)
	tooLong := accepted + "界"

	identityBody := func(blurb string) string {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"blurb": blurb, "isNsfw": false, "name": "Catalog name",
		})
		if err != nil {
			t.Fatal(err)
		}
		return string(body)
	}
	if response := saveIdentity(t, r, session, started.ID, identityBody(accepted)); response.Code != http.StatusNoContent {
		t.Fatalf("save 400-character blurb: status = %d, want 204: %s", response.Code, response.Body.String())
	}

	response := saveIdentity(t, r, session, started.ID, identityBody(tooLong))
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
	if saved := fetchStartedAsset(t, r, session, started.ID); saved.Blurb != accepted {
		t.Fatalf("saved blurb has %d characters, want the intact 400-character value", len([]rune(saved.Blurb)))
	}
}
