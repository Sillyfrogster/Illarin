package staff_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestASiteVisitCountsOncePerReaderPerDayAndNeverForABot(t *testing.T) {
	t.Parallel()
	router, _, pool := harness.NewConnectRouter(t)
	visit := func(agent, address string) {
		t.Helper()
		request := apitest.BrowserMutation(httptest.NewRequest(http.MethodPost, "/v1/visits", nil))
		request.Header.Set("User-Agent", agent)
		request.RemoteAddr = address + ":1234"
		if response := apitest.Send(t, router, request); response.Code != http.StatusNoContent {
			t.Fatalf("visit status = %d, want 204: %s", response.Code, response.Body.String())
		}
	}
	visit("Mozilla/5.0 Firefox", "198.51.100.7")
	visit("Mozilla/5.0 Firefox", "198.51.100.7")
	visit("Mozilla/5.0 Firefox", "198.51.100.8")
	visit("Googlebot/2.1", "198.51.100.9")

	if got := eventCounts(t, pool)["visit"]; got != 2 {
		t.Fatalf("visit events = %d, want one for each of the two readers", got)
	}
}
