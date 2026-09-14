package http

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type publicationToken struct {
	ID         string     `json:"id"`
	GrantID    string     `json:"grantId"`
	Name       string     `json:"name"`
	Prefix     string     `json:"prefix"`
	CreatedAt  time.Time  `json:"createdAt"`
	ExpiresAt  *time.Time `json:"expiresAt"`
	LastUsedAt *time.Time `json:"lastUsedAt"`
	RevokedAt  *time.Time `json:"revokedAt"`
	Active     bool       `json:"active"`
}

type publicationTokenList struct {
	Tokens []publicationToken `json:"tokens"`
}

type issuedPublicationToken struct {
	Token publicationToken `json:"token"`
	Value string           `json:"value"`
}

type publicationCredential struct {
	Token publicationToken `json:"token"`
	Grant publicationGrant `json:"grant"`
}

type contributor struct {
	handle  string
	session *http.Cookie
	grant   publicationGrant
}

func (s publicationStack) contributor(t *testing.T, email, handle string) contributor {
	t.Helper()
	session := s.member(t, email, handle)
	illarin := s.appBySlug(t, "illarin")
	announcement := s.categoryBySlug(t, "announcement")
	made := s.approved(t, handle, illarin.ID, []string{announcement.ID}, announcement.ID)
	return contributor{handle: handle, session: session, grant: made}
}

func (s publicationStack) issue(
	t *testing.T,
	session *http.Cookie,
	grantID, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/grants/"+grantID+"/tokens", body,
	), session))
}

func (s publicationStack) issued(
	t *testing.T,
	who contributor,
	name string,
) issuedPublicationToken {
	t.Helper()
	response := s.issue(t, who.session, who.grant.ID, `{"name":"`+name+`"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("issue %s status = %d: %s", name, response.Code, response.Body.String())
	}
	var made issuedPublicationToken
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode issued token: %v", err)
	}
	return made
}

func (s publicationStack) tokens(
	t *testing.T,
	session *http.Cookie,
	grantID string,
) []publicationToken {
	t.Helper()
	response := send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/grants/"+grantID+"/tokens", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("list tokens status = %d: %s", response.Code, response.Body.String())
	}
	var listed publicationTokenList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	return listed.Tokens
}

func (s publicationStack) revokeToken(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/tokens/"+id, nil), session,
	))
}

func (s publicationStack) bearing(t *testing.T, value string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodGet, "/v1/publication/token", nil)
	request.Header.Set("Authorization", "Bearer "+value)
	return send(t, s.router, request)
}

func TestATokenValueIsShownOnceAndNeverKeptWhole(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")

	made := stack.issued(t, who, "Release robot")
	if !strings.HasPrefix(made.Value, "ip1."+made.Token.Prefix+".") {
		t.Fatalf("issued value %q does not carry its kind and prefix", made.Value)
	}
	if !made.Token.Active || made.Token.Name != "Release robot" {
		t.Fatalf("issued token = %+v", made.Token)
	}

	for _, listed := range stack.tokens(t, who.session, who.grant.ID) {
		body, err := json.Marshal(listed)
		if err != nil {
			t.Fatalf("encode listed token: %v", err)
		}
		if strings.Contains(string(body), made.Value) {
			t.Fatalf("a listed token carried its value: %s", body)
		}
	}

	var stored int
	err := stack.pool.QueryRow(context.Background(), `
		select count(*) from publication_tokens
		 where name = $1 or prefix = $1 or encode(token_hash, 'escape') = $1
	`, made.Value).Scan(&stored)
	if err != nil {
		t.Fatalf("search for a stored token value: %v", err)
	}
	if stored != 0 {
		t.Fatalf("%d rows kept the token value", stored)
	}
}

func TestOneGrantCarriesSeveralTokensAndRevokingOneLeavesTheRest(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")

	robot := stack.issued(t, who, "Release robot")
	desk := stack.issued(t, who, "Writing desk")

	if stack.bearing(t, robot.Value).Code != http.StatusOK {
		t.Fatal("a fresh token did not authenticate")
	}
	if revoked := stack.revokeToken(t, who.session, robot.Token.ID); revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}
	if refused := stack.bearing(t, robot.Value); refused.Code != http.StatusUnauthorized {
		t.Fatalf("a revoked token answered %d", refused.Code)
	}
	if stack.bearing(t, desk.Value).Code != http.StatusOK {
		t.Fatal("revoking one token stopped another")
	}
	if open := stack.workspace(t, who.session); len(open.Grants) != 1 {
		t.Fatalf("revoking a token changed the grant: %+v", open.Grants)
	}

	listed := stack.tokens(t, who.session, who.grant.ID)
	if len(listed) != 2 {
		t.Fatalf("listed %d tokens, want 2", len(listed))
	}
	if !listed[0].Active || listed[1].Active || listed[1].RevokedAt == nil {
		t.Fatalf("listed tokens = %+v", listed)
	}
}

func TestAnExpiredTokenStopsAuthenticatingWhileItsHashStillMatches(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")

	soon := time.Now().Add(2 * time.Minute).UTC().Format(time.RFC3339)
	response := stack.issue(t, who.session, who.grant.ID,
		`{"name":"Temporary tool","expiresAt":"`+soon+`"}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("issue with expiry status = %d: %s", response.Code, response.Body.String())
	}
	var made issuedPublicationToken
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode issued token: %v", err)
	}
	if made.Token.ExpiresAt == nil {
		t.Fatal("the issued token kept no expiry")
	}
	if stack.bearing(t, made.Value).Code != http.StatusOK {
		t.Fatal("a token expiring in two minutes did not authenticate")
	}

	_, err := stack.pool.Exec(context.Background(), `
		update publication_tokens
		   set created_at = now() - interval '2 hours', expires_at = now() - interval '1 hour'
		 where id = $1
	`, made.Token.ID)
	if err != nil {
		t.Fatalf("age the token: %v", err)
	}

	digest := sha256.Sum256([]byte(made.Value))
	var matches bool
	err = stack.pool.QueryRow(context.Background(), `
		select exists (select 1 from publication_tokens where id = $1 and token_hash = $2)
	`, made.Token.ID, digest[:]).Scan(&matches)
	if err != nil {
		t.Fatalf("read the stored hash: %v", err)
	}
	if !matches {
		t.Fatal("the expired token no longer hashes to what was stored")
	}
	if refused := stack.bearing(t, made.Value); refused.Code != http.StatusUnauthorized {
		t.Fatalf("an expired token answered %d", refused.Code)
	}
	for _, listed := range stack.tokens(t, who.session, who.grant.ID) {
		if listed.ID == made.Token.ID && listed.Active {
			t.Fatal("an expired token still reads as active")
		}
	}
}

func TestRevokingAGrantStopsEveryTokenBeneathIt(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")

	revoked := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/publication/grants/"+who.grant.ID, nil,
	), stack.authority))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke grant status = %d: %s", revoked.Code, revoked.Body.String())
	}

	if refused := stack.bearing(t, made.Value); refused.Code != http.StatusUnauthorized {
		t.Fatalf("a token under a revoked grant answered %d", refused.Code)
	}
	again := stack.issue(t, who.session, who.grant.ID, `{"name":"Another"}`)
	if again.Code != http.StatusBadRequest {
		t.Fatalf("issuing under a revoked grant answered %d: %s", again.Code, again.Body.String())
	}
}

func TestTheAuthorityRevokesAnyTokenAndAnAdminCannot(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	outsider := stack.member(t, "outsider@example.com", "publication.outsider")
	made := stack.issued(t, who, "Release robot")

	for _, role := range []string{"user", "moderator", "admin"} {
		setRole(t, stack.pool, "publication.outsider", role)
		if listed := send(t, stack.router, authorized(httptest.NewRequest(
			http.MethodGet, "/v1/publication/grants/"+who.grant.ID+"/tokens", nil,
		), outsider)); listed.Code != http.StatusForbidden {
			t.Fatalf("%s listed another's tokens: %d", role, listed.Code)
		}
		if refused := stack.revokeToken(t, outsider, made.Token.ID); refused.Code != http.StatusForbidden {
			t.Fatalf("%s revoke = %d, want 403: %s", role, refused.Code, refused.Body.String())
		}
	}
	if stack.bearing(t, made.Value).Code != http.StatusOK {
		t.Fatal("a refused revocation stopped the token anyway")
	}

	if listed := stack.tokens(t, stack.authority, who.grant.ID); len(listed) != 1 {
		t.Fatalf("the authority listed %d tokens, want 1", len(listed))
	}
	if revoked := stack.revokeToken(t, stack.authority, made.Token.ID); revoked.Code != http.StatusNoContent {
		t.Fatalf("authority revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}
	if refused := stack.bearing(t, made.Value); refused.Code != http.StatusUnauthorized {
		t.Fatalf("a token the authority revoked answered %d", refused.Code)
	}
}

func TestOneContributorNeverReachesAnothersTokens(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	other := stack.member(t, "other@example.com", "lumiverse.writer")
	theirs := stack.approved(
		t, "lumiverse.writer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	made := stack.issued(t, who, "Release robot")

	if listed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/grants/"+who.grant.ID+"/tokens", nil,
	), other)); listed.Code != http.StatusForbidden {
		t.Fatalf("another contributor listed these tokens: %d", listed.Code)
	}
	if refused := stack.revokeToken(t, other, made.Token.ID); refused.Code != http.StatusForbidden {
		t.Fatalf("another contributor revoked this token: %d", refused.Code)
	}
	if refused := stack.issue(t, other, who.grant.ID, `{"name":"Theirs"}`); refused.Code != http.StatusForbidden {
		t.Fatalf("another contributor issued under this grant: %d", refused.Code)
	}
	if listed := stack.tokens(t, other, theirs.ID); len(listed) != 0 {
		t.Fatalf("their own grant already carried %d tokens", len(listed))
	}
}

func TestAPublicationTokenReachesNothingOutsideThePublication(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")

	elsewhere := []struct{ method, path string }{
		{http.MethodGet, "/v1/instances"},
		{http.MethodGet, "/v1/instances/me"},
		{http.MethodPost, "/v1/library/sync"},
		{http.MethodGet, "/v1/publication/apps"},
		{http.MethodGet, "/v1/publication/grants"},
		{http.MethodGet, "/v1/publication/workspace"},
		{http.MethodGet, "/v1/publication/grants/" + who.grant.ID + "/tokens"},
		{http.MethodPost, "/v1/publication/grants/" + who.grant.ID + "/tokens"},
		{http.MethodDelete, "/v1/publication/tokens/" + made.Token.ID},
		{http.MethodPut, "/v1/account/profile"},
		{http.MethodPut, "/v1/profiles/publication.writer/restriction"},
	}
	for _, reached := range elsewhere {
		request := httptest.NewRequest(reached.method, reached.path, strings.NewReader("{}"))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+made.Value)
		response := send(t, stack.router, request)
		if response.Code != http.StatusUnauthorized && response.Code != http.StatusForbidden {
			t.Errorf(
				"%s %s answered %d to a publication token: %s",
				reached.method, reached.path, response.Code, response.Body.String(),
			)
		}
	}

	signedIn := httptest.NewRequest(http.MethodGet, "/v1/auth/session", nil)
	signedIn.Header.Set("Authorization", "Bearer "+made.Value)
	answered := send(t, stack.router, signedIn)
	if answered.Code != http.StatusUnauthorized || strings.Contains(answered.Body.String(), "user") {
		t.Fatalf("a publication token reached the session route: %d %s",
			answered.Code, answered.Body.String())
	}

	if session := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/token", nil), who.session,
	)); session.Code != http.StatusUnauthorized {
		t.Fatalf("a session stood in for a publication token: %d", session.Code)
	}
}

func TestOnlyAPublicationTokenOfTheRightShapeAuthenticates(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")

	refused := []string{
		"",
		"not-a-token",
		"ia1." + made.Token.Prefix + strings.Repeat("A", 44),
		strings.Replace(made.Value, "ip1.", "ia1.", 1),
		made.Value[:len(made.Value)-1] + "x",
		made.Value + "A",
	}
	for _, value := range refused {
		if answered := stack.bearing(t, value); answered.Code != http.StatusUnauthorized {
			t.Errorf("%q authenticated with %d", value, answered.Code)
		}
	}

	answered := stack.bearing(t, made.Value)
	if answered.Code != http.StatusOK {
		t.Fatalf("the real token answered %d: %s", answered.Code, answered.Body.String())
	}
	var credential publicationCredential
	if err := json.Unmarshal(answered.Body.Bytes(), &credential); err != nil {
		t.Fatalf("decode credential: %v", err)
	}
	if credential.Grant.ID != who.grant.ID || credential.Grant.App.Slug != "illarin" {
		t.Fatalf("the token resolved to %+v", credential.Grant)
	}
	if credential.Token.Prefix != made.Token.Prefix {
		t.Fatalf("the token named itself %+v", credential.Token)
	}
	if strings.Contains(answered.Body.String(), made.Value) {
		t.Fatal("reading a credential handed its value back")
	}
	if answered.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("a credential response was cacheable: %q", answered.Header().Get("Cache-Control"))
	}
}

func TestAnUnnamedOrBadlyDatedTokenIsRefused(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")

	past := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	refused := []string{
		`{"name":""}`,
		`{"name":"   "}`,
		`{"name":"` + strings.Repeat("n", 49) + `"}`,
		`{"name":"Past","expiresAt":"` + past + `"}`,
	}
	for _, body := range refused {
		if answered := stack.issue(t, who.session, who.grant.ID, body); answered.Code != http.StatusBadRequest {
			t.Errorf("%s answered %d, want 400", body, answered.Code)
		}
	}
	if listed := stack.tokens(t, who.session, who.grant.ID); len(listed) != 0 {
		t.Fatalf("a refused request still made %d tokens", len(listed))
	}
}

func TestIssuingAndRevokingATokenIsAuditedWithoutItsValue(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")
	if revoked := stack.revokeToken(t, who.session, made.Token.ID); revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d", revoked.Code)
	}

	rows, err := stack.pool.Query(context.Background(), `
		select action, token_id from publication_audits
		 where token_id = $1 order by recorded_at
	`, made.Token.ID)
	if err != nil {
		t.Fatalf("read publication audits: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var action, tokenID string
		if err := rows.Scan(&action, &tokenID); err != nil {
			t.Fatalf("scan a publication audit: %v", err)
		}
		actions = append(actions, action)
	}
	if len(actions) != 2 || actions[0] != "token.issued" || actions[1] != "token.revoked" {
		t.Fatalf("audited actions = %v", actions)
	}

	var carried int
	err = stack.pool.QueryRow(context.Background(), `
		select count(*) from publication_audits where action = $1
	`, made.Value).Scan(&carried)
	if err != nil {
		t.Fatalf("search the audits for the token value: %v", err)
	}
	if carried != 0 {
		t.Fatal("an audit carried the token value")
	}
}

func TestNothingCarryingATokenIsCacheable(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")

	listed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/grants/"+who.grant.ID+"/tokens", nil,
	), who.session))
	if listed.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("the token listing was cacheable: %q", listed.Header().Get("Cache-Control"))
	}

	issuing := stack.issue(t, who.session, who.grant.ID, `{"name":"Another"}`)
	if issuing.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("a new token was cacheable: %q", issuing.Header().Get("Cache-Control"))
	}

	if bearing := stack.bearing(t, made.Value); bearing.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("a credential read was cacheable: %q", bearing.Header().Get("Cache-Control"))
	}
}

func TestUsingATokenRecordsThatItWasUsed(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	who := stack.contributor(t, "writer@example.com", "publication.writer")
	made := stack.issued(t, who, "Release robot")

	if made.Token.LastUsedAt != nil {
		t.Fatalf("a token was used before it was handed over: %+v", made.Token)
	}
	if stack.bearing(t, made.Value).Code != http.StatusOK {
		t.Fatal("a fresh token did not authenticate")
	}
	listed := stack.tokens(t, who.session, who.grant.ID)
	if len(listed) != 1 || listed[0].LastUsedAt == nil {
		t.Fatalf("using a token recorded nothing: %+v", listed)
	}
}
