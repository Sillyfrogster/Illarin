package connect_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
)

var bothPermissions = []string{apitest.ReceivePermission, "library:sync"}

// approveRequestFor starts a request asking for both permissions and approves it with the given body plus its proof
func approveRequestFor(t *testing.T, r *gin.Engine, session *http.Cookie, body map[string]any) (*httptest.ResponseRecorder, string) {
	t.Helper()
	started, _ := apitest.StartConnection(t, r, apitest.ConnectionStartBody("Paper Lantern", "desk", bothPermissions))
	_, pending := apitest.ReviewConnectionRequest(t, r, session, started.UserCode)
	body["approvalToken"] = pending.ApprovalToken
	return apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPost, "/v1/connect/requests/"+started.UserCode+"/approve", body, session,
	)), started.DeviceCode
}

func setPermissions(t *testing.T, r *gin.Engine, session *http.Cookie, appID string, permissions []string) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, r, apitest.BrowserRequest(
		t, http.MethodPut, "/v1/connected-apps/"+appID+"/permissions",
		map[string]any{"permissions": permissions}, session,
	))
}

func TestOnlyTheOwnersChoiceGrantsPermissions(t *testing.T) {
	t.Parallel()
	r, session, _ := harness.NewConnectRouter(t)

	for _, test := range []struct {
		name string
		body map[string]any
		want []string
	}{
		{"nothing chosen", map[string]any{"permissions": []string{}}, []string{}},
		{"choice left out", map[string]any{}, []string{}},
		{"receiving only", map[string]any{"permissions": []string{apitest.ReceivePermission}}, []string{apitest.ReceivePermission}},
		{"library only", map[string]any{"permissions": []string{"library:sync"}}, []string{"library:sync"}},
		{"both", map[string]any{"permissions": []string{"library:sync", apitest.ReceivePermission}}, bothPermissions},
	} {
		approved, deviceCode := approveRequestFor(t, r, session, test.body)
		if approved.Code != http.StatusOK {
			t.Fatalf("%s: approval status = %d, want 200: %s", test.name, approved.Code, approved.Body.String())
		}
		credentials := apitest.DecodeResponse[apitest.PolledConnection](t, apitest.Poll(t, r, deviceCode))
		if credentials.ConnectedApp == nil || !slices.Equal(credentials.ConnectedApp.Permissions, test.want) {
			t.Errorf("%s: the app was told it holds %+v, want %v", test.name, credentials.ConnectedApp, test.want)
		}
	}

	for _, test := range []struct {
		name string
		body map[string]any
	}{
		{"an unknown permission", map[string]any{"permissions": []string{"works:delete"}}},
		{"the old field name", map[string]any{"scopes": bothPermissions}},
	} {
		if refused, _ := approveRequestFor(t, r, session, test.body); refused.Code != http.StatusBadRequest {
			t.Errorf("approval with %s status = %d, want 400", test.name, refused.Code)
		}
	}
}

func TestAChangedPermissionBindsIssuedCredentialsAndSurvivesRefresh(t *testing.T) {
	t.Parallel()
	r, session, pool := harness.NewConnectRouter(t)
	app := apitest.ConnectApp(t, r, session, "Paper Lantern", "desk", bothPermissions)
	workID := apitest.PublishedCharacter(t, r, session)
	syncLibrary(t, r, app.AccessToken, false, []map[string]any{{"workId": workID}}, nil)

	changed := setPermissions(t, r, session, app.ConnectedApp.ID, []string{})
	if changed.Code != http.StatusOK {
		t.Fatalf("change status = %d, want 200: %s", changed.Code, changed.Body.String())
	}
	if got := rowCount(t, pool, `select count(*) from app_library_entries where connected_app_id = $1`, app.ConnectedApp.ID); got != 0 {
		t.Errorf("the library mirror kept %d entries after sharing stopped", got)
	}
	libraryWrite := func(token string) int {
		return apitest.Send(t, r, apitest.AsApp(t, http.MethodPost, "/v1/library/sync", token,
			map[string]any{"snapshot": true, "entries": []any{}})).Code
	}
	if code := libraryWrite(app.AccessToken); code != http.StatusForbidden {
		t.Errorf("library write with an already-issued token = %d, want 403", code)
	}

	apitest.DeclareFormats(t, r, app.AccessToken, []string{"test_opaque"})
	refreshed := apitest.SendJSON(t, r, http.MethodPost, "/v1/connect/refresh",
		apitest.JSONText(t, map[string]string{"refreshToken": app.RefreshToken}))
	credentials := apitest.DecodeResponse[apitest.AppCredentials](t, refreshed)
	if len(credentials.ConnectedApp.Permissions) != 0 {
		t.Errorf("after an update and a refresh the app holds %v, want nothing", credentials.ConnectedApp.Permissions)
	}
	if code := apitest.Collect(t, r, credentials.AccessToken, nil).Code; code != http.StatusForbidden {
		t.Errorf("collect after refresh = %d, want 403", code)
	}
	if code := libraryWrite(credentials.AccessToken); code != http.StatusForbidden {
		t.Errorf("library write after refresh = %d, want 403", code)
	}

	if refused := setPermissions(t, r, session, app.ConnectedApp.ID, []string{"works:delete"}); refused.Code != http.StatusBadRequest {
		t.Errorf("unknown permission status = %d, want 400", refused.Code)
	}
	stranger := apitest.AddVerifiedUser(t, r, pool, "stranger@example.com", "stranger.creator")
	if refused := setPermissions(t, r, stranger, app.ConnectedApp.ID, bothPermissions); refused.Code != http.StatusNotFound {
		t.Errorf("another account's change status = %d, want 404", refused.Code)
	}
}
