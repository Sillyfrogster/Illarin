package connect_test

import (
	"net/http"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestAnAppCanRequestAWorkOnlyForItselfWithReceivePermission(t *testing.T) {
	t.Parallel()
	router, session, _ := harness.NewConnectRouter(t)
	app := apitest.ConnectApp(t, router, session, "Paper Lantern", "desk", []string{"work:receive"})
	apitest.DeclareFormats(t, router, app.AccessToken, []string{"test_opaque"})
	other := apitest.ConnectApp(t, router, session, "Paper Lantern", "phone", []string{"library:sync"})
	workID := apitest.PublishedCharacter(t, router, session)
	path := "/v1/works/" + workID + "/receive"
	request := apitest.AsApp(t, http.MethodPost, path, app.AccessToken, nil)
	request.Header.Set("Origin", "http://localhost:8001")
	rec := apitest.Send(t, router, request)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("request work = %d: %s", rec.Code, rec.Body.String())
	}
	assertAppCORS(t, rec)
	queued := apitest.DecodeResponse[apitest.QueuedSend](t, rec)
	if queued.ConnectedAppID != app.ConnectedApp.ID {
		t.Fatalf("queued for another installation: %+v", queued)
	}
	for _, token := range []string{"", other.AccessToken} {
		refused := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, path, token, nil))
		want := http.StatusForbidden
		if token == "" {
			want = http.StatusUnauthorized
		}
		if refused.Code != want {
			t.Fatalf("missing credential or permission = %d, want %d", refused.Code, want)
		}
	}
	deleted := apitest.Send(t, router, apitest.BrowserRequest(t, http.MethodDelete, "/v1/works/"+workID, nil, session))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete = %d", deleted.Code)
	}
	refused := apitest.Send(t, router, apitest.AsApp(t, http.MethodPost, path, app.AccessToken, nil))
	if refused.Code != http.StatusNotFound {
		t.Fatalf("withdrawn work = %d, want 404", refused.Code)
	}
}
