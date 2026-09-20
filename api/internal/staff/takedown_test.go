package staff_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/format"
	"github.com/google/uuid"
)

func TestOnlyAnAdminCanTakeDownAnWorkAndTheDecisionIsRecordedTogether(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, "")

	request := func() *http.Request {
		return apitest.AuthorizedJSONRequest(
			t,
			http.MethodPut,
			"/v1/works/"+workID+"/takedown",
			`{"reason":"Copyright report under review"}`,
			session,
		)
	}

	refused := apitest.Send(t, router, request())
	if refused.Code != http.StatusForbidden {
		t.Fatalf("creator takedown status = %d, want 403: %s", refused.Code, refused.Body.String())
	}
	if _, err := pool.Exec(context.Background(), `
		update users set role = 'moderator' where username = 'verified.creator'
	`); err != nil {
		t.Fatalf("make test account a moderator: %v", err)
	}
	refused = apitest.Send(t, router, request())
	if refused.Code != http.StatusForbidden {
		t.Fatalf("moderator takedown status = %d, want 403: %s", refused.Code, refused.Body.String())
	}

	var adminID uuid.UUID
	if err := pool.QueryRow(context.Background(), `
		update users set role = 'admin' where username = 'verified.creator' returning id
	`).Scan(&adminID); err != nil {
		t.Fatalf("make test account an admin: %v", err)
	}

	takenDown := apitest.Send(t, router, request())
	if takenDown.Code != http.StatusNoContent {
		t.Fatalf("admin takedown status = %d, want 204: %s", takenDown.Code, takenDown.Body.String())
	}

	var actor uuid.UUID
	var reason string
	var at time.Time
	if err := pool.QueryRow(context.Background(), `
		select taken_down_by, taken_down_reason, taken_down_at from works where id = $1
	`, workID).Scan(&actor, &reason, &at); err != nil {
		t.Fatalf("read takedown decision: %v", err)
	}
	if actor != adminID || reason != "Copyright report under review" || at.IsZero() {
		t.Fatalf("takedown = actor %s, reason %q, at %v", actor, reason, at)
	}

	clear := httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID+"/takedown", nil)
	apitest.Authorized(clear, session)
	cleared := apitest.Send(t, router, clear)
	if cleared.Code != http.StatusNoContent {
		t.Fatalf("admin clear status = %d, want 204: %s", cleared.Code, cleared.Body.String())
	}
	var decisionCleared bool
	if err := pool.QueryRow(context.Background(), `
		select taken_down_by is null and taken_down_reason is null and taken_down_at is null
		  from works where id = $1
	`, workID).Scan(&decisionCleared); err != nil {
		t.Fatalf("read cleared takedown: %v", err)
	}
	if !decisionCleared {
		t.Fatal("clearing did not remove the complete takedown decision")
	}
}

func TestOwnerCanViewAndDownloadATakenDownWorkWithItsDecision(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, "")
	mediaID := addTakedownTestMedia(t, router, session, workID)

	if _, err := pool.Exec(context.Background(), `
		update works work
		   set taken_down_at = '2026-08-14 12:30:00+00',
		       taken_down_by = owner.id,
		       taken_down_reason = 'Copyright report under review'
		  from users owner
		 where work.id = $1 and owner.username = 'verified.creator'
	`, workID); err != nil {
		t.Fatalf("take down work: %v", err)
	}

	pageRequest, err := http.NewRequest(http.MethodGet, "/v1/works/"+workID, nil)
	if err != nil {
		t.Fatalf("make owner page request: %v", err)
	}
	pageRequest.AddCookie(session)
	pageResponse := apitest.Send(t, router, pageRequest)
	if pageResponse.Code != http.StatusOK {
		t.Fatalf("owner page status = %d, want 200: %s", pageResponse.Code, pageResponse.Body.String())
	}
	var page apitest.WorkPageResponse
	if err := json.Unmarshal(pageResponse.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode owner page: %v", err)
	}
	if page.Takedown == nil || page.Takedown.Reason != "Copyright report under review" ||
		page.Takedown.At.IsZero() {
		t.Fatalf("takedown shown to owner = %+v", page.Takedown)
	}

	downloadRequest, err := http.NewRequest(http.MethodGet, "/download/"+workID, nil)
	if err != nil {
		t.Fatalf("make owner download request: %v", err)
	}
	downloadRequest.AddCookie(session)
	download := apitest.Send(t, router, downloadRequest)
	if download.Code != http.StatusOK {
		t.Fatalf("owner download status = %d, want 200: %s", download.Code, download.Body.String())
	}

	for _, path := range []string{
		"/v1/works/" + workID + "/media",
		works.SignedURL("/media/" + mediaID + "/grid/2"),
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(session)
		response := apitest.Send(t, router, request)
		if response.Code != http.StatusOK {
			t.Fatalf("owner GET %s status = %d, want 200: %s", path, response.Code, response.Body.String())
		}
	}
}

func TestUnavailableWorksAnswerTheSameAcrossEveryPublicRead(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	takenDownID := apitest.UploadVisibilityTestWork(t, router, session, works, "")
	deletedID := apitest.UploadVisibilityTestWork(t, router, session, works, "")
	takenDownMediaID := addTakedownTestMedia(t, router, session, takenDownID)
	deletedMediaID := addTakedownTestMedia(t, router, session, deletedID)

	if _, err := pool.Exec(context.Background(), `
		update works work
		   set taken_down_at = now(), taken_down_by = owner.id, taken_down_reason = 'Review'
		  from users owner
		 where work.id = $1 and owner.username = 'verified.creator'
	`, takenDownID); err != nil {
		t.Fatalf("take down work: %v", err)
	}
	if _, err := pool.Exec(context.Background(),
		`update works set deleted_at = now(), recoverable_until = now() + interval '30 days' where id = $1`, deletedID,
	); err != nil {
		t.Fatalf("delete work: %v", err)
	}

	missingWorkID := "22222222-2222-4222-8222-222222222222"
	missingMediaID := "33333333-3333-4333-8333-333333333333"
	for name, paths := range map[string][]string{
		"asset page": {
			"/v1/works/" + takenDownID,
			"/v1/works/" + deletedID,
			"/v1/works/" + missingWorkID,
		},
		"download": {
			"/download/" + takenDownID,
			"/download/" + deletedID,
			"/download/" + missingWorkID,
		},
		"media list": {
			"/v1/works/" + takenDownID + "/media",
			"/v1/works/" + deletedID + "/media",
			"/v1/works/" + missingWorkID + "/media",
		},
		"media file": {
			"/media/" + takenDownMediaID + "/grid/2",
			"/media/" + deletedMediaID + "/grid/2",
			"/media/" + missingMediaID + "/grid/2",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var first *httptest.ResponseRecorder
			for _, path := range paths {
				response := apitest.Send(t, router, httptest.NewRequest(http.MethodGet, path, nil))
				if response.Code != http.StatusNotFound {
					t.Fatalf("GET %s status = %d, want 404: %s", path, response.Code, response.Body.String())
				}
				if first == nil {
					first = response
					continue
				}
				if response.Body.String() != first.Body.String() ||
					!reflect.DeepEqual(response.Header(), first.Header()) {
					t.Fatalf("GET %s response differs: body %q headers %v; want body %q headers %v",
						path, response.Body.String(), response.Header(), first.Body.String(), first.Header())
				}
			}
		})
	}
}

func TestTakenDownWorkRefusesCreatorMutations(t *testing.T) {
	t.Parallel()
	router, session, works, pool := harness.NewVerifiedUploadRouterWithPool(t, format.NewRegistry())
	workID := apitest.UploadVisibilityTestWork(t, router, session, works, "")
	if _, err := pool.Exec(context.Background(), `
		update works work
		   set taken_down_at = now(), taken_down_by = owner.id, taken_down_reason = 'Review'
		  from users owner
		 where work.id = $1 and owner.username = 'verified.creator'
	`, workID); err != nil {
		t.Fatalf("take down work: %v", err)
	}

	changes := []*http.Request{
		apitest.AuthorizedJSONRequest(
			t, http.MethodPut, "/v1/works/"+workID+"/visibility",
			`{"visibility":"unlisted"}`, session,
		),
		apitest.Authorized(apitest.MediaUploadRequest(
			t, workID, "gallery", apitest.PNG(t, 2, 2),
		), session),
	}
	for _, request := range changes {
		response := apitest.Send(t, router, request)
		if response.Code != http.StatusConflict {
			t.Fatalf("%s %s status = %d, want 409: %s",
				request.Method, request.URL.Path, response.Code, response.Body.String())
		}
	}

	clear := httptest.NewRequest(http.MethodDelete, "/v1/works/"+workID+"/takedown", nil)
	apitest.Authorized(clear, session)
	response := apitest.Send(t, router, clear)
	if response.Code != http.StatusForbidden {
		t.Fatalf("creator clear status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestTakenDownWorkRefusesEveryPrivatePromptMutation(t *testing.T) {
	t.Parallel()
	_, router, session, _, pool := harness.NewVerifiedRoutersWithPool(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartPreset(t, router, session, "lumiverse")
	coreBlock := apitest.BlockNamed(t, started.Blocks, "preset_core")
	core := apitest.EditableBlock(coreBlock)
	const privateText = "This prompt stays frozen."
	core.Elements[0].Content = json.RawMessage(`{"groups":[],"fragments":[
		{"name":"Frozen","role":"system","text":"` + privateText + `","private":true,"enabled":true}
	]}`)
	apps := []string{"lumiverse"}
	core.AllowedApps = &apps
	if response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, core); response.Code != http.StatusOK {
		t.Fatalf("save private prompt: %d %s", response.Code, response.Body.String())
	}
	owner := apitest.FetchStartedWork(t, router, session, started.ID)
	core = apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	if _, err := pool.Exec(t.Context(), `
		update works work
		   set taken_down_at = now(), taken_down_by = owner.id, taken_down_reason = 'Review'
		  from users owner
		 where work.id = $1 and owner.username = 'verified.creator'
	`, started.ID); err != nil {
		t.Fatalf("take down work: %v", err)
	}

	textChange := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	textChange.Elements[0].Content = json.RawMessage(strings.Replace(
		string(core.Elements[0].Content), privateText, "This prompt tried to change.", 1,
	))
	stateChange := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	stateChange.Elements[0].Content = json.RawMessage(strings.Replace(
		string(core.Elements[0].Content), `,"private":true`, "", 1,
	))
	stateChange.AllowedApps = &[]string{}
	policyChange := apitest.EditableBlock(apitest.BlockNamed(t, owner.Blocks, "preset_core"))
	policyChange.AllowedApps = &[]string{}
	mutations := map[string]apitest.SaveBlockBody{
		"private text": textChange,
		"privacy":      stateChange,
		"allowed apps": policyChange,
	}

	for name, mutation := range mutations {
		t.Run(name, func(t *testing.T) {
			response := apitest.SaveBlock(t, router, session, started.ID, coreBlock.ID, mutation)
			if response.Code != http.StatusConflict {
				t.Fatalf("status = %d, want 409: %s", response.Code, response.Body.String())
			}
		})
	}

	var storedText string
	if err := pool.QueryRow(t.Context(), `
		select payload ->> 'text' from private_prompts where work_id = $1
	`, started.ID).Scan(&storedText); err != nil {
		t.Fatalf("read private prompt after refused saves: %v", err)
	}
	if storedText != privateText {
		t.Fatalf("private prompt after refused saves = %q, want %q", storedText, privateText)
	}
	if payloads, policies := apitest.PrivatePromptCounts(t, pool, started.ID); payloads != 1 || policies != 1 {
		t.Fatalf("after refused saves: %d payloads and %d policy rows, want 1 and 1", payloads, policies)
	}
}

func addTakedownTestMedia(
	t *testing.T,
	router http.Handler,
	session *http.Cookie,
	workID string,
) string {
	t.Helper()
	response := apitest.Send(t, router, apitest.Authorized(apitest.MediaUploadRequest(
		t, workID, "gallery", apitest.PNG(t, 1, 1),
	), session))
	if response.Code != http.StatusCreated {
		t.Fatalf("add media status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var media struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &media); err != nil {
		t.Fatalf("decode media: %v", err)
	}
	return media.ID
}
