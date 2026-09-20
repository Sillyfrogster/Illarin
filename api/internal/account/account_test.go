package account_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/account"
	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/google/uuid"
)

func TestSignUpStartsAnUnverifiedSessionAndSendsAVerificationLink(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r := harness.NewRouterWithSender(t, 1<<20, api.DefaultDeadlines(), outbox)

	rec := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-up", `{
		"email":"reader@example.com",
		"password":"correct horse battery staple",
		"handle":"book.worm"
	}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201. body: %s", rec.Code, rec.Body.String())
	}
	if len(outbox.Messages) != 1 {
		t.Fatalf("sent %d verification messages, want 1", len(outbox.Messages))
	}
	if message := outbox.Messages[0]; message.Address != "reader@example.com" ||
		!strings.HasPrefix(message.Link, "http://localhost:3000/verify-email?token=") {
		t.Errorf("verification message = %+v", message)
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("set %d cookies, want one session cookie", len(cookies))
	}
	session := cookies[0]
	if session.Name != api.SessionCookie || !session.HttpOnly || session.Path != "/" ||
		session.SameSite != http.SameSiteLaxMode {
		t.Errorf("session cookie = %+v", session)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	req.AddCookie(session)
	state := apitest.Send(t, r, req)
	if state.Code != http.StatusOK {
		t.Fatalf("session status = %d, want 200. body: %s", state.Code, state.Body.String())
	}
	var got struct {
		User *struct {
			ID            uuid.UUID `json:"id"`
			Handle        string    `json:"handle"`
			Email         *string   `json:"email"`
			EmailVerified bool      `json:"emailVerified"`
		} `json:"user"`
	}
	if err := json.Unmarshal(state.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode session: %v", err)
	}
	if got.User == nil || got.User.ID == uuid.Nil || got.User.Handle != "book.worm" ||
		got.User.Email == nil || *got.User.Email != "reader@example.com" || got.User.EmailVerified {
		t.Errorf("session user = %+v", got.User)
	}
}

func TestAccountEntryPointsLimitRepeatedAttemptsFromOneSource(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)
	tests := []struct {
		name  string
		path  string
		limit int
	}{
		{name: "sign up", path: "/v1/auth/sign-up", limit: account.SignUpLimit},
		{name: "sign in", path: "/v1/auth/sign-in", limit: account.SignInLimit},
		{name: "request password reset", path: "/v1/auth/password-reset", limit: account.PasswordResetLimit},
		{name: "complete password reset", path: "/v1/auth/password-reset/complete", limit: account.PasswordResetCompleteLimit},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for attempt := 0; attempt < test.limit; attempt++ {
				response := accountAttempt(t, r, test.path, "198.51.100.20:4000")
				if response.Code == http.StatusTooManyRequests {
					t.Fatalf("attempt %d was limited early", attempt+1)
				}
			}
			limited := accountAttempt(t, r, test.path, "198.51.100.20:4000")
			if limited.Code != http.StatusTooManyRequests {
				t.Fatalf("status = %d, want 429. body: %s", limited.Code, limited.Body.String())
			}
			if limited.Header().Get("Retry-After") == "" {
				t.Error("the limited response has no Retry-After header")
			}
		})
	}
}

func accountAttempt(t *testing.T, r http.Handler, path, remoteAddress string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{"))
	request.Header.Set("Content-Type", "application/json")
	request.RemoteAddr = remoteAddress
	return apitest.Send(t, r, request)
}

func TestSignUpAcceptsAnyNonemptyPassword(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)

	short := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-up", `{
		"email":"short.password@example.com",
		"password":"x",
		"handle":"short.password"
	}`)
	if short.Code != http.StatusCreated {
		t.Errorf("one-character password status = %d, want 201. body: %s",
			short.Code, short.Body.String())
	}

	longPassword := strings.Repeat("long password ", 16)
	long := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-up", `{
		"email":"long.password@example.com",
		"password":"`+longPassword+`",
		"handle":"long.password"
	}`)
	if long.Code != http.StatusCreated {
		t.Errorf("long password status = %d, want 201. body: %s",
			long.Code, long.Body.String())
	}
	longSignIn := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-in", `{
		"email":"long.password@example.com",
		"password":"`+longPassword+`"
	}`)
	if longSignIn.Code != http.StatusOK {
		t.Errorf("long password sign-in status = %d, want 200. body: %s",
			longSignIn.Code, longSignIn.Body.String())
	}

	empty := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-up", `{
		"email":"empty.password@example.com",
		"password":"",
		"handle":"empty.password"
	}`)
	if empty.Code != http.StatusBadRequest {
		t.Errorf("empty password status = %d, want 400. body: %s",
			empty.Code, empty.Body.String())
	}
}

func TestSignUpAcceptsOnlyTheHandleVocabulary(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)
	invalid := []string{
		"ab",
		strings.Repeat("a", 33),
		"UPPER",
		"has-hyphen",
		"12345",
		"._._",
	}
	for _, handle := range invalid {
		t.Run(handle, func(t *testing.T) {
			rec := apitest.SignUpRequest(t, r, "handles@example.com", handle)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("handle %q status = %d, want 400. body: %s",
					handle, rec.Code, rec.Body.String())
			}
		})
	}

	mixedNumbersAndPunctuation := apitest.SignUpRequest(t, r, "mixed@example.com", "1._")
	if mixedNumbersAndPunctuation.Code != http.StatusCreated {
		t.Errorf("mixed number and punctuation status = %d, want 201. body: %s",
			mixedNumbersAndPunctuation.Code, mixedNumbersAndPunctuation.Body.String())
	}
}

func TestTheFirstAccountToVerifyAnAddressClaimsIt(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r := harness.NewRouterWithSender(t, 1<<20, api.DefaultDeadlines(), outbox)

	first := apitest.SignUp(t, r, "shared@example.com", "first.reader")
	second := apitest.SignUp(t, r, "shared@example.com", "second.reader")
	if len(outbox.Messages) != 2 {
		t.Fatalf("sent %d verification messages, want 2", len(outbox.Messages))
	}

	verificationURL, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	verified := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200. body: %s", verified.Code, verified.Body.String())
	}

	firstState := apitest.SessionState(t, r, first)
	if firstState.Email == nil || *firstState.Email != "shared@example.com" || !firstState.EmailVerified {
		t.Errorf("winner = %+v", firstState)
	}
	secondState := apitest.SessionState(t, r, second)
	if secondState.Email != nil || secondState.EmailVerified {
		t.Errorf("pending copy was not cleared: %+v", secondState)
	}

	losingURL, err := url.Parse(outbox.Messages[1].Link)
	if err != nil {
		t.Fatalf("parse second verification link: %v", err)
	}
	losingVerification := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+losingURL.Query().Get("token")+`"}`)
	if losingVerification.Code != http.StatusBadRequest {
		t.Errorf("losing verification status = %d, want 400", losingVerification.Code)
	}

	alreadyClaimed := apitest.SignUpRequest(t, r, "shared@example.com", "third.reader")
	if alreadyClaimed.Code != http.StatusConflict {
		t.Errorf("signup on verified address = %d, want 409. body: %s",
			alreadyClaimed.Code, alreadyClaimed.Body.String())
	}
}

func TestConcurrentVerificationProducesOneWinnerWithoutDeadlock(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r, pool := harness.NewRouterWithSenderAndPool(t, 1<<20, api.DefaultDeadlines(), outbox)
	apitest.SignUp(t, r, "race@example.com", "race.first")
	apitest.SignUp(t, r, "race@example.com", "race.second")

	tokens := make([]string, 0, len(outbox.Messages))
	for _, message := range outbox.Messages {
		verificationURL, err := url.Parse(message.Link)
		if err != nil {
			t.Fatalf("parse verification link: %v", err)
		}
		tokens = append(tokens, verificationURL.Query().Get("token"))
	}

	ctx := context.Background()
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin advisory lock blocker: %v", err)
	}
	defer blocker.Rollback(ctx)
	if _, err := blocker.Exec(ctx, `select pg_advisory_xact_lock(
		hashtextextended('illarin-email:' || $1::text, 0)
	)`, "race@example.com"); err != nil {
		t.Fatalf("hold email lock: %v", err)
	}

	start := make(chan struct{})
	ready := make(chan struct{}, len(tokens))
	statuses := make(chan int, len(tokens))
	for _, token := range tokens {
		go func() {
			ready <- struct{}{}
			<-start
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/verify-email",
				strings.NewReader(`{"token":"`+token+`"}`))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)
			statuses <- rec.Code
		}()
	}
	for range tokens {
		<-ready
	}
	close(start)

	deadline := time.Now().Add(5 * time.Second)
	for {
		var waiting int
		if err := pool.QueryRow(ctx, `select count(*) from pg_locks
			where locktype = 'advisory' and not granted`).Scan(&waiting); err != nil {
			t.Fatalf("count waiting verification requests: %v", err)
		}
		if waiting >= len(tokens) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d verification requests reached the shared email lock", waiting)
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := blocker.Commit(ctx); err != nil {
		t.Fatalf("release email lock: %v", err)
	}

	counts := map[int]int{}
	for range tokens {
		select {
		case status := <-statuses:
			counts[status]++
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent verification did not finish")
		}
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusBadRequest] != 1 {
		t.Errorf("verification statuses = %v, want one 200 and one 400", counts)
	}
}

func TestAnUnverifiedAccountCanSignOutAndBackIn(t *testing.T) {
	t.Parallel()
	r := harness.NewRouter(t)
	session := apitest.SignUp(t, r, "Return.Reader@example.com", "return.reader")

	signOutRequest := httptest.NewRequest(http.MethodPost, "/v1/auth/sign-out", nil)
	apitest.Authorized(signOutRequest, session)
	signedOut := apitest.Send(t, r, signOutRequest)
	if signedOut.Code != http.StatusNoContent {
		t.Fatalf("sign out status = %d, want 204. body: %s", signedOut.Code, signedOut.Body.String())
	}
	if cleared := signedOut.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge >= 0 {
		t.Errorf("cleared cookie = %+v", cleared)
	}
	if current := apitest.SessionState(t, r, session); current.Email != nil {
		t.Errorf("signed-out session still reaches %+v", current)
	}

	wrongPassword := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-in", `{
		"email":"return.reader@example.com",
		"password":"this is not the password"
	}`)
	if wrongPassword.Code != http.StatusUnauthorized {
		t.Errorf("wrong password status = %d, want 401", wrongPassword.Code)
	}

	signedIn := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-in", `{
		"email":"RETURN.READER@example.com",
		"password":"correct horse battery staple"
	}`)
	if signedIn.Code != http.StatusOK {
		t.Fatalf("sign in status = %d, want 200. body: %s", signedIn.Code, signedIn.Body.String())
	}
	state := apitest.SessionState(t, r, signedIn.Result().Cookies()[0])
	if state.Email == nil || *state.Email != "return.reader@example.com" || state.EmailVerified {
		t.Errorf("signed-in account = %+v", state)
	}
}

func TestRenamingAHandleRetiresTheOldProfileWithoutARedirect(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r := harness.NewRouterWithSender(t, 1<<20, api.DefaultDeadlines(), outbox)
	session := apitest.SignUp(t, r, "rename@example.com", "first.handle")

	renameRequest := httptest.NewRequest(http.MethodPatch, "/v1/account/handle",
		strings.NewReader(`{"handle":"second.handle"}`))
	renameRequest.Header.Set("Content-Type", "application/json")
	apitest.Authorized(renameRequest, session)
	if unverified := apitest.Send(t, r, renameRequest); unverified.Code != http.StatusForbidden {
		t.Fatalf("unverified rename status = %d, want 403. body: %s",
			unverified.Code, unverified.Body.String())
	}

	verificationURL, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	verified := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200. body: %s", verified.Code, verified.Body.String())
	}

	renamedRequest := httptest.NewRequest(http.MethodPatch, "/v1/account/handle",
		strings.NewReader(`{"handle":"second.handle"}`))
	renamedRequest.Header.Set("Content-Type", "application/json")
	apitest.Authorized(renamedRequest, session)
	renamed := apitest.Send(t, r, renamedRequest)
	if renamed.Code != http.StatusOK {
		t.Fatalf("rename status = %d, want 200. body: %s", renamed.Code, renamed.Body.String())
	}

	oldProfile := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/profiles/first.handle", nil))
	if oldProfile.Code != http.StatusNotFound {
		t.Errorf("old profile status = %d, want 404", oldProfile.Code)
	}
	if location := oldProfile.Header().Get("Location"); location != "" {
		t.Errorf("old profile redirects to %q", location)
	}
	newProfile := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/profiles/second.handle", nil))
	if newProfile.Code != http.StatusOK {
		t.Errorf("new profile status = %d, want 200. body: %s", newProfile.Code, newProfile.Body.String())
	}

	reused := apitest.SignUpRequest(t, r, "reuse@example.com", "first.handle")
	if reused.Code != http.StatusConflict {
		t.Errorf("retired handle signup = %d, want 409. body: %s", reused.Code, reused.Body.String())
	}
}

func TestOnlyAVerifiedAccountCanUpload(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r, _, handlers := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	unverified := apitest.SignUp(t, r, "uploader@example.com", "new.uploader")

	if browse := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works", nil)); browse.Code != http.StatusOK {
		t.Errorf("unverified browse status = %d, want 200", browse.Code)
	}
	withoutSession := apitest.Send(t, r, apitest.UploadRequest(t, apitest.ExampleMetadata("Anonymous"), []byte("file")))
	if withoutSession.Code != http.StatusUnauthorized {
		t.Errorf("anonymous upload status = %d, want 401", withoutSession.Code)
	}
	unverifiedUpload := apitest.UploadRequest(t, apitest.ExampleMetadata("Unverified"), []byte("file"))
	apitest.Authorized(unverifiedUpload, unverified)
	if rec := apitest.Send(t, r, unverifiedUpload); rec.Code != http.StatusForbidden {
		t.Errorf("unverified upload status = %d, want 403. body: %s", rec.Code, rec.Body.String())
	}

	verificationURL, err := url.Parse(outbox.Messages[0].Link)
	if err != nil {
		t.Fatalf("parse verification link: %v", err)
	}
	verified := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/verify-email",
		`{"token":"`+verificationURL.Query().Get("token")+`"}`)
	if verified.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200. body: %s", verified.Code, verified.Body.String())
	}

	metadata := apitest.ExampleMetadata("Verified")
	metadata["filename"] = "verified.lumitheme"
	created := apitest.UploadAndFinish(t, r, unverified, handlers.Works, metadata, []byte("file"))
	var work struct {
		Work *struct {
			ID string `json:"id"`
		} `json:"work"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &work); err != nil {
		t.Fatalf("decode work: %v", err)
	}
	if work.Work == nil {
		t.Fatal("completed upload has no work")
	}

	viewer := apitest.SignUp(t, r, "viewer@example.com", "plain.viewer")
	download := httptest.NewRequest(http.MethodGet, "/download/"+work.Work.ID, nil)
	download.AddCookie(viewer)
	if viewed := apitest.Send(t, r, download); viewed.Code != http.StatusOK {
		t.Errorf("unverified view status = %d, want 200. body: %s", viewed.Code, viewed.Body.String())
	}
}

func TestAnUnverifiedAccountCanCorrectItsEmail(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	r := harness.NewRouterWithSender(t, 1<<20, api.DefaultDeadlines(), outbox)
	session := apitest.SignUp(t, r, "mistyped@example.com", "careful.creator")

	change := httptest.NewRequest(http.MethodPatch, "/v1/account/email",
		strings.NewReader(`{"email":"correct@example.com"}`))
	change.Header.Set("Content-Type", "application/json")
	apitest.Authorized(change, session)
	changed := apitest.Send(t, r, change)
	if changed.Code != http.StatusOK {
		t.Fatalf("change email status = %d, want 200. body: %s", changed.Code, changed.Body.String())
	}
	if len(outbox.Messages) != 2 || outbox.Messages[1].Address != "correct@example.com" {
		t.Fatalf("verification outbox = %+v", outbox.Messages)
	}
	state := apitest.SessionState(t, r, session)
	if state.Email == nil || *state.Email != "correct@example.com" || state.EmailVerified {
		t.Errorf("account after correction = %+v", state)
	}

	oldAddress := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-in", `{
		"email":"mistyped@example.com",
		"password":"correct horse battery staple"
	}`)
	if oldAddress.Code != http.StatusUnauthorized {
		t.Errorf("old address sign in = %d, want 401", oldAddress.Code)
	}
	newAddress := apitest.SendJSON(t, r, http.MethodPost, "/v1/auth/sign-in", `{
		"email":"correct@example.com",
		"password":"correct horse battery staple"
	}`)
	if newAddress.Code != http.StatusOK {
		t.Errorf("corrected address sign in = %d, want 200. body: %s",
			newAddress.Code, newAddress.Body.String())
	}
}
