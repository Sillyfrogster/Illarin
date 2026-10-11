package profile_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestRecommendationsStaySeparateFromFeaturedAndCreditTheCreator(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	other := s.reader(t, "other@example.com", "other.creator")
	work := apitest.PublishedCharacter(t, s.router, other)
	for range 2 {
		if got := s.recommend(t, s.creator, http.MethodPost, work); got.Code != http.StatusOK {
			t.Fatalf("recommend = %d: %s", got.Code, got.Body.String())
		}
	}
	shown := s.profile(t, nil)
	if len(shown.Recommendations) != 1 || shown.Recommendations[0].ID != work || shown.Recommendations[0].Creator != "other.creator" || shown.Works != 0 {
		t.Fatalf("recommendations = %+v, works = %d", shown.Recommendations, shown.Works)
	}
	featured := apitest.PublishedCharacter(t, s.router, s.creator)
	s.feature(t, featured)
	if got := s.recommend(t, other, http.MethodDelete, work); got.Code != http.StatusOK {
		t.Fatalf("another account removing its own recommendation = %d", got.Code)
	}
	if shown := s.profile(t, other); len(shown.Recommendations) != 1 || len(shown.Featured) != 1 || shown.Works != 1 {
		t.Fatalf("another account changed the profile: %+v", shown)
	}
	if got := s.recommend(t, s.creator, http.MethodDelete, work); got.Code != http.StatusOK {
		t.Fatalf("remove = %d: %s", got.Code, got.Body.String())
	}
	if shown := s.profile(t, nil); len(shown.Recommendations) != 0 || len(shown.Featured) != 1 || shown.Featured[0].ID != featured || shown.Works != 1 {
		t.Fatalf("removing a recommendation changed Featured or works: %+v", shown)
	}
}

func TestRecommendationsRefuseUnauthorizedAndUnavailableWorks(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	other := s.reader(t, "other@example.com", "other.creator")
	listed := apitest.PublishedCharacter(t, s.router, other)
	draft := apitest.StartCharacter(t, s.router, other).ID
	own := apitest.PublishedCharacter(t, s.router, s.creator)
	unlisted := apitest.PublishedCharacter(t, s.router, other)
	deleted := apitest.PublishedCharacter(t, s.router, other)
	takenDown := apitest.PublishedCharacter(t, s.router, other)
	if _, err := s.pool.Exec(t.Context(), `update works set visibility = 'unlisted' where id = $1`, unlisted); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `update works set deleted_at = now(), recoverable_until = now() + interval '30 days' where id = $1`, deleted); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `update works set taken_down_at = now(), taken_down_by = owner_id, taken_down_reason = 'Test removal' where id = $1`, takenDown); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{draft, own, unlisted, deleted, takenDown, "00000000-0000-4000-8000-000000000000", "bad-id"} {
		if got := s.recommend(t, s.creator, http.MethodPost, id); got.Code != http.StatusBadRequest {
			t.Errorf("recommend %s = %d: %s", id, got.Code, got.Body.String())
		}
	}
	unverified := apitest.SignUp(t, s.router, "unverified@example.com", "unverified.one")
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		for _, check := range []struct {
			session *http.Cookie
			status  int
		}{{nil, http.StatusUnauthorized}, {unverified, http.StatusForbidden}} {
			if got := s.recommend(t, check.session, method, listed); got.Code != check.status {
				t.Errorf("unauthorized %s = %d, want %d", method, got.Code, check.status)
			}
		}
	}
	if got := s.recommend(t, s.creator, http.MethodPost, listed); got.Code != http.StatusOK {
		t.Fatalf("recommend = %d", got.Code)
	}
	staff := s.reader(t, "staff@example.com", "moon.staff")
	apitest.SetRole(t, s.pool, "moon.staff", "admin")
	restricted := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(t, http.MethodPut, "/v1/profiles/"+creatorHandle+"/restricted", `{"reason":"Test restriction."}`, staff))
	if restricted.Code != http.StatusOK && restricted.Code != http.StatusNoContent {
		t.Fatalf("restrict = %d: %s", restricted.Code, restricted.Body.String())
	}
	if got := s.profile(t, nil); len(got.Recommendations) != 0 {
		t.Fatalf("restricted recommendations = %+v", got.Recommendations)
	}
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		if got := s.recommend(t, s.creator, method, listed); got.Code != http.StatusForbidden {
			t.Errorf("restricted %s = %d", method, got.Code)
		}
	}
}

func TestRecommendationsRespectAdultSettingsAndLaterVisibilityChanges(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	other := s.reader(t, "other@example.com", "other.creator")
	ids := make([]string, 4)
	for i := range ids {
		ids[i] = apitest.PublishedCharacter(t, s.router, other)
		if got := s.recommend(t, s.creator, http.MethodPost, ids[i]); got.Code != http.StatusOK {
			t.Fatalf("recommend = %d", got.Code)
		}
	}
	if _, err := s.pool.Exec(t.Context(), `update works set is_nsfw = true where id = $1`, ids[0]); err != nil {
		t.Fatal(err)
	}
	preference := apitest.Send(t, s.router, apitest.AuthorizedJSONRequest(t, http.MethodPut, "/v1/account/nsfw-preference", `{"preference":"hidden"}`, s.creator))
	if preference.Code != http.StatusOK && preference.Code != http.StatusNoContent {
		t.Fatalf("preference = %d: %s", preference.Code, preference.Body.String())
	}
	if got := s.profile(t, s.creator); len(got.Recommendations) != 3 || got.RecommendationCount != 4 {
		t.Fatalf("hidden recommendations = %+v, count %d", got.Recommendations, got.RecommendationCount)
	}
	if _, err := s.pool.Exec(t.Context(), `update works set visibility = 'unlisted' where id = $1`, ids[1]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `update works set deleted_at = now(), recoverable_until = now() + interval '30 days' where id = $1`, ids[2]); err != nil {
		t.Fatal(err)
	}
	if _, err := s.pool.Exec(t.Context(), `update works set taken_down_at = now(), taken_down_by = owner_id, taken_down_reason = 'Test removal' where id = $1`, ids[3]); err != nil {
		t.Fatal(err)
	}
	if got := s.profile(t, s.creator); len(got.Recommendations) != 0 || got.RecommendationCount != 1 {
		t.Fatalf("unavailable recommendations = %+v, count %d", got.Recommendations, got.RecommendationCount)
	}
	replacement := apitest.PublishedCharacter(t, s.router, other)
	if got := s.recommend(t, s.creator, http.MethodPost, replacement); got.Code != http.StatusOK {
		t.Fatalf("replace unavailable = %d: %s", got.Code, got.Body.String())
	}
	if got := s.profile(t, s.creator); len(got.Recommendations) != 1 || got.Recommendations[0].ID != replacement || got.RecommendationCount != 2 {
		t.Fatalf("replacement = %+v", got)
	}
}

func TestConcurrentRecommendationsCannotExceedFour(t *testing.T) {
	t.Parallel()
	s := newPortfolioStack(t)
	other := s.reader(t, "other@example.com", "other.creator")
	requests := make([]*http.Request, 6)
	for i := range requests {
		id := apitest.PublishedCharacter(t, s.router, other)
		requests[i] = apitest.AuthorizedJSONRequest(t, http.MethodPost, "/v1/account/profile/recommendations", fmt.Sprintf(`{"workId":%q}`, id), s.creator)
	}
	var group sync.WaitGroup
	statuses := make(chan int, len(requests))
	for _, request := range requests {
		group.Go(func() { rec := httptest.NewRecorder(); s.router.ServeHTTP(rec, request); statuses <- rec.Code })
	}
	group.Wait()
	close(statuses)
	added, refused := 0, 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			added++
		case http.StatusBadRequest:
			refused++
		default:
			t.Errorf("concurrent recommend = %d", status)
		}
	}
	if got := s.profile(t, nil); added != 4 || refused != 2 || len(got.Recommendations) != 4 {
		t.Fatalf("added %d, refused %d, recommendations %+v", added, refused, got.Recommendations)
	}
	shown := s.profile(t, nil)
	if got := s.recommend(t, s.creator, http.MethodPost, shown.Recommendations[0].ID); got.Code != http.StatusOK {
		t.Fatalf("repeat add at limit = %d", got.Code)
	}
}

func (s portfolioStack) recommend(t *testing.T, session *http.Cookie, method, id string) *httptest.ResponseRecorder {
	t.Helper()
	path := "/v1/account/profile/recommendations"
	var request *http.Request
	if method == http.MethodPost {
		request = httptest.NewRequest(method, path, strings.NewReader(fmt.Sprintf(`{"workId":%q}`, id)))
		request.Header.Set("Content-Type", "application/json")
	} else {
		request = httptest.NewRequest(method, path+"/"+id, nil)
	}
	request = apitest.BrowserMutation(request)
	if session != nil {
		request = apitest.Authorized(request, session)
	}
	return apitest.Send(t, s.router, request)
}
