package blog_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

type publicationStack struct {
	router    *gin.Engine
	pool      *pgxpool.Pool
	handlers  apitest.Services
	outbox    *apitest.VerificationOutbox
	authority *http.Cookie
}

type profileAvatar struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type publicationApp struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Name     string `json:"name"`
	Home     string `json:"home"`
	Position int    `json:"position"`
	Retired  bool   `json:"retired"`
}

type publicationAppList struct {
	Apps []publicationApp `json:"apps"`
}

type publicationCategory struct {
	ID       string `json:"id"`
	Slug     string `json:"slug"`
	Label    string `json:"label"`
	Position int    `json:"position"`
	Retired  bool   `json:"retired"`
}

type publicationCategoryList struct {
	Categories []publicationCategory `json:"categories"`
}

type publicationGrantHolder struct {
	Handle      string         `json:"handle"`
	DisplayName string         `json:"displayName"`
	Avatar      *profileAvatar `json:"avatar"`
	Restricted  bool           `json:"restricted"`
}

type publicationGrant struct {
	ID              string                 `json:"id"`
	Holder          publicationGrantHolder `json:"holder"`
	App             publicationApp         `json:"app"`
	Categories      []publicationCategory  `json:"categories"`
	DefaultCategory publicationCategory    `json:"defaultCategory"`
	GrantedAt       time.Time              `json:"grantedAt"`
	RevokedAt       *time.Time             `json:"revokedAt"`
	Active          bool                   `json:"active"`
}

type publicationGrantList struct {
	Grants []publicationGrant `json:"grants"`
}

type publicationWorkspace struct {
	Handle string             `json:"handle"`
	Grants []publicationGrant `json:"grants"`
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

func newPublicationStack(t *testing.T) publicationStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, handlers := harness.NewRouterWithSenderPoolAndServices(
		t, 1<<20, api.DefaultDeadlines(), outbox,
	)
	session := apitest.VerifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	apitest.HoldsAuthority(t, pool, "publication.authority")
	return publicationStack{
		router: router, pool: pool, handlers: handlers, outbox: outbox, authority: session,
	}
}

func (s publicationStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

func jsonRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func (s publicationStack) apps(t *testing.T) []publicationApp {
	t.Helper()
	rows, err := s.pool.Query(context.Background(), `
		select id::text, slug, name, home_url, position, retired_at is not null
		  from publication_apps
		 order by position, created_at
	`)
	if err != nil {
		t.Fatalf("read apps: %v", err)
	}
	defer rows.Close()
	listed := []publicationApp{}
	for rows.Next() {
		var one publicationApp
		if err := rows.Scan(&one.ID, &one.Slug, &one.Name, &one.Home, &one.Position, &one.Retired); err != nil {
			t.Fatalf("read an app: %v", err)
		}
		listed = append(listed, one)
	}
	return listed
}

func (s publicationStack) categories(t *testing.T) []publicationCategory {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/categories", nil), s.authority,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list categories status = %d: %s", response.Code, response.Body.String())
	}
	var listed publicationCategoryList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode categories: %v", err)
	}
	return listed.Categories
}

func (s publicationStack) categoryBySlug(t *testing.T, slug string) publicationCategory {
	t.Helper()
	for _, category := range s.categories(t) {
		if category.Slug == slug {
			return category
		}
	}
	t.Fatalf("no seeded category with the slug %q", slug)
	return publicationCategory{}
}

func (s publicationStack) appBySlug(t *testing.T, slug string) publicationApp {
	t.Helper()
	for _, app := range s.apps(t) {
		if app.Slug == slug {
			return app
		}
	}
	t.Fatalf("no configured app with the slug %q", slug)
	return publicationApp{}
}

func (s publicationStack) configureApp(t *testing.T, slug, name, home string) publicationApp {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		insert into publication_apps (id, slug, name, home_url, position)
		values (gen_random_uuid(), $1, $2, $3,
		        (select coalesce(max(position) + 1, 0) from publication_apps))
	`, slug, name, home)
	if err != nil {
		t.Fatalf("configure %s: %v", slug, err)
	}
	return s.appBySlug(t, slug)
}

func (s publicationStack) approve(
	t *testing.T,
	session *http.Cookie,
	handle, appID string,
	categoryIDs []string,
	defaultID string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"handle": handle, "appId": appID,
		"categoryIds": categoryIDs, "defaultCategoryId": defaultID,
	})
	if err != nil {
		t.Fatalf("encode approval: %v", err)
	}
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/grants", string(body),
	), session))
}

func (s publicationStack) approved(
	t *testing.T,
	handle, appID string,
	categoryIDs []string,
	defaultID string,
) publicationGrant {
	t.Helper()
	response := s.approve(t, s.authority, handle, appID, categoryIDs, defaultID)
	if response.Code != http.StatusCreated {
		t.Fatalf("approve %s status = %d: %s", handle, response.Code, response.Body.String())
	}
	var made publicationGrant
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode grant: %v", err)
	}
	return made
}

func (s publicationStack) workspace(t *testing.T, session *http.Cookie) publicationWorkspace {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/workspace", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("read workspace status = %d: %s", response.Code, response.Body.String())
	}
	var open publicationWorkspace
	if err := json.Unmarshal(response.Body.Bytes(), &open); err != nil {
		t.Fatalf("decode workspace: %v", err)
	}
	return open
}

func TestIllarinAndTheThreeCategoriesAreSeeded(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)

	illarin := stack.appBySlug(t, "illarin")
	if illarin.Name != "Illarin" || illarin.Home != "https://illarin.com" {
		t.Fatalf("seeded app = %+v", illarin)
	}

	seeded := stack.categories(t)
	if len(seeded) != 3 ||
		seeded[0].Slug != "announcement" ||
		seeded[1].Slug != "release" ||
		seeded[2].Slug != "article" {
		t.Fatalf("seeded categories = %+v", seeded)
	}
}

func TestOnlyThePublicationAuthorityManagesCategoriesAndGrants(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	outsider := stack.member(t, "outsider@example.com", "publication.outsider")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")

	reads := []string{
		"/v1/publication/categories",
		"/v1/publication/grants",
	}
	for _, role := range []string{"user", "moderator", "admin"} {
		apitest.SetRole(t, stack.pool, "publication.outsider", role)
		for _, path := range reads {
			refused := apitest.Send(t, stack.router, apitest.Authorized(
				httptest.NewRequest(http.MethodGet, path, nil), outsider,
			))
			if refused.Code != http.StatusForbidden {
				t.Fatalf("%s read of %s = %d, want 403", role, path, refused.Code)
			}
		}
		relabelled := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
			http.MethodPatch, "/v1/publication/categories/"+announcement.ID, `{"label":"News"}`,
		), outsider))
		if relabelled.Code != http.StatusForbidden {
			t.Fatalf("%s relabel category = %d, want 403", role, relabelled.Code)
		}
		granted := stack.approve(
			t, outsider, "publication.outsider", illarin.ID,
			[]string{announcement.ID}, announcement.ID,
		)
		if granted.Code != http.StatusForbidden {
			t.Fatalf("%s approve = %d, want 403: %s", role, granted.Code, granted.Body.String())
		}
	}
}

func TestTheAuthorityRelabelsAndOrdersCategories(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	announcement := stack.categoryBySlug(t, "announcement")
	article := stack.categoryBySlug(t, "article")
	release := stack.categoryBySlug(t, "release")

	relabelled := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPatch, "/v1/publication/categories/"+announcement.ID, `{"label":"Update"}`,
	), stack.authority))
	if relabelled.Code != http.StatusOK {
		t.Fatalf("relabel status = %d: %s", relabelled.Code, relabelled.Body.String())
	}

	ordered := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/categories",
		`{"categoryIds":["`+article.ID+`","`+release.ID+`","`+announcement.ID+`"]}`,
	), stack.authority))
	if ordered.Code != http.StatusOK {
		t.Fatalf("order categories status = %d: %s", ordered.Code, ordered.Body.String())
	}

	after := stack.categories(t)
	if after[0].Slug != "article" || after[2].Slug != "announcement" {
		t.Fatalf("categories after ordering = %+v", after)
	}
	if after[2].Label != "Update" {
		t.Fatalf("relabelled category = %+v", after[2])
	}
	if after[2].ID != announcement.ID {
		t.Fatal("a relabelled category changed its stable id")
	}
}

func TestAGrantNeedsAVerifiedAccountAnAppAndABackedDefaultCategory(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	stack.member(t, "writer@example.com", "lumiverse.writer")
	apitest.SignUp(t, stack.router, "unverified@example.com", "unverified.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	release := stack.categoryBySlug(t, "release")

	unverified := stack.approve(
		t, stack.authority, "unverified.writer", illarin.ID,
		[]string{announcement.ID}, announcement.ID,
	)
	if unverified.Code != http.StatusBadRequest {
		t.Fatalf("unverified approval status = %d, want 400: %s", unverified.Code, unverified.Body.String())
	}

	empty := stack.approve(
		t, stack.authority, "lumiverse.writer", illarin.ID, []string{}, announcement.ID,
	)
	if empty.Code != http.StatusBadRequest {
		t.Fatalf("empty category set status = %d, want 400", empty.Code)
	}

	unbacked := stack.approve(
		t, stack.authority, "lumiverse.writer", illarin.ID,
		[]string{announcement.ID}, release.ID,
	)
	if unbacked.Code != http.StatusBadRequest {
		t.Fatalf("unbacked default status = %d, want 400", unbacked.Code)
	}

	stack.approved(t, "lumiverse.writer", illarin.ID, []string{announcement.ID}, announcement.ID)
	again := stack.approve(
		t, stack.authority, "lumiverse.writer", illarin.ID,
		[]string{announcement.ID}, announcement.ID,
	)
	if again.Code != http.StatusConflict {
		t.Fatalf("second grant for the same app status = %d, want 409", again.Code)
	}
}

func TestAContributorSeesOnlyTheirOwnEffectiveScope(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.member(t, "scoped@example.com", "scoped.writer")
	other := stack.member(t, "elsewhere@example.com", "other.writer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	release := stack.categoryBySlug(t, "release")

	stack.approved(
		t, "scoped.writer", lumiverse.ID,
		[]string{announcement.ID, release.ID}, release.ID,
	)
	stack.approved(t, "other.writer", illarin.ID, []string{announcement.ID}, announcement.ID)

	open := stack.workspace(t, writer)
	if open.Handle != "scoped.writer" || len(open.Grants) != 1 {
		t.Fatalf("workspace = %+v", open)
	}
	held := open.Grants[0]
	if held.App.Slug != "lumiverse" {
		t.Fatalf("workspace app = %+v", held.App)
	}
	if len(held.Categories) != 2 || held.DefaultCategory.Slug != "release" {
		t.Fatalf("workspace categories = %+v, default = %+v", held.Categories, held.DefaultCategory)
	}

	refused := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/grants", nil), writer,
	))
	if refused.Code != http.StatusForbidden {
		t.Fatalf("a contributor read every grant: %d", refused.Code)
	}
	if elsewhere := stack.workspace(t, other); elsewhere.Grants[0].App.Slug != "illarin" {
		t.Fatalf("the other contributor's workspace = %+v", elsewhere.Grants)
	}
}

func TestSeparatePeopleForOneAppGetSeparateGrants(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	first := stack.member(t, "first@example.com", "first.developer")
	second := stack.member(t, "second@example.com", "second.developer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")

	held := stack.approved(
		t, "first.developer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	stack.approved(t, "second.developer", lumiverse.ID, []string{announcement.ID}, announcement.ID)

	revoked := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}

	if open := stack.workspace(t, first); len(open.Grants) != 0 {
		t.Fatalf("the revoked contributor kept %+v", open.Grants)
	}
	if open := stack.workspace(t, second); len(open.Grants) != 1 {
		t.Fatalf("the other contributor lost their grant: %+v", open.Grants)
	}
}

func TestRevocationKeepsTheAccountAndTheGrantRecord(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	stack.member(t, "ended@example.com", "ended.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	held := stack.approved(
		t, "ended.writer", illarin.ID, []string{announcement.ID}, announcement.ID,
	)

	revoked := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}
	twice := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if twice.Code != http.StatusNotFound {
		t.Fatalf("second revoke status = %d, want 404", twice.Code)
	}

	listed := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/grants", nil), stack.authority,
	))
	if listed.Code != http.StatusOK {
		t.Fatalf("list grants status = %d: %s", listed.Code, listed.Body.String())
	}
	var grants publicationGrantList
	if err := json.Unmarshal(listed.Body.Bytes(), &grants); err != nil {
		t.Fatalf("decode grants: %v", err)
	}
	if len(grants.Grants) != 1 || grants.Grants[0].Active || grants.Grants[0].RevokedAt == nil {
		t.Fatalf("grant record after revocation = %+v", grants.Grants)
	}

	var role string
	var verified bool
	err := stack.pool.QueryRow(context.Background(), `
		select role, email_verified_at is not null from users where username = 'ended.writer'
	`).Scan(&role, &verified)
	if err != nil {
		t.Fatalf("read the account: %v", err)
	}
	if role != "user" || !verified {
		t.Fatalf("revocation changed the account: role %q, verified %v", role, verified)
	}
}

func TestAGrantIsDirectPublicationAndNothingElse(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.member(t, "bounded@example.com", "bounded.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	stack.approved(t, "bounded.writer", illarin.ID, []string{announcement.ID}, announcement.ID)

	elsewhere := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPatch, "/v1/publication/categories/" + announcement.ID, `{"label":"News"}`},
		{http.MethodGet, "/v1/publication/grants", ""},
	}
	for _, refused := range elsewhere {
		var request *http.Request
		if refused.body == "" {
			request = httptest.NewRequest(refused.method, refused.path, nil)
		} else {
			request = jsonRequest(t, refused.method, refused.path, refused.body)
		}
		response := apitest.Send(t, stack.router, apitest.Authorized(request, writer))
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s %s = %d, want 403: %s",
				refused.method, refused.path, response.Code, response.Body.String())
		}
	}

	var role string
	if err := stack.pool.QueryRow(context.Background(), `
		select role from users where username = 'bounded.writer'
	`).Scan(&role); err != nil {
		t.Fatalf("read role: %v", err)
	}
	if role != "user" {
		t.Fatalf("a grant changed the account role to %q", role)
	}

	if open := stack.workspace(t, writer); len(open.Grants) != 1 || open.Grants[0].Active != true {
		t.Fatalf("workspace = %+v, want one active grant and no review state", open.Grants)
	}
}

func TestGrantChangesLeaveSafeIdentifiersBehind(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	stack.member(t, "audited@example.com", "audited.writer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	held := stack.approved(
		t, "audited.writer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	revoked := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d", revoked.Code)
	}

	rows, err := stack.pool.Query(context.Background(), `
		select action from publication_audits order by recorded_at
	`)
	if err != nil {
		t.Fatalf("read audits: %v", err)
	}
	defer rows.Close()
	recorded := make([]string, 0, 3)
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			t.Fatalf("read an audit: %v", err)
		}
		recorded = append(recorded, action)
	}
	want := []string{"grant.created", "grant.revoked"}
	if len(recorded) != len(want) {
		t.Fatalf("audit actions = %v, want %v", recorded, want)
	}
	for index, action := range want {
		if recorded[index] != action {
			t.Fatalf("audit actions = %v, want %v", recorded, want)
		}
	}

	var columns int
	err = stack.pool.QueryRow(context.Background(), `
		select count(*) from information_schema.columns
		 where table_name = 'publication_audits'
		   and data_type not in ('uuid', 'timestamp with time zone')
		   and column_name not in ('action', 'credential', 'before_state', 'after_state')
	`).Scan(&columns)
	if err != nil {
		t.Fatalf("read audit columns: %v", err)
	}
	if columns != 0 {
		t.Fatalf("the audit table has %d columns that could hold copied content", columns)
	}
	for _, pinned := range []string{
		"publication_audits_credential_check",
		"publication_audits_state_check",
		"publication_audits_next_state_check",
	} {
		var exists bool
		err = stack.pool.QueryRow(context.Background(), `
			select exists (select 1 from pg_constraint where conname = $1)
		`, pinned).Scan(&exists)
		if err != nil {
			t.Fatalf("read %s: %v", pinned, err)
		}
		if !exists {
			t.Errorf("%s does not pin its column to a closed set of words", pinned)
		}
	}
}

func TestAGrantNamesItsContributorTheWayTheirProfileDoes(t *testing.T) {
	t.Parallel()
	stack := newPublicationStack(t)
	writer := stack.member(t, "named@example.com", "named.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")

	saved := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/account/profile",
		`{"displayName":"Kestrel","biography":"","contactEmail":"","links":[]}`,
	), writer))
	if saved.Code != http.StatusOK {
		t.Fatalf("save profile status = %d: %s", saved.Code, saved.Body.String())
	}

	made := stack.approved(t, "named.writer", illarin.ID, []string{announcement.ID}, announcement.ID)
	if made.Holder.Handle != "named.writer" || made.Holder.DisplayName != "Kestrel" {
		t.Fatalf("grant holder = %+v", made.Holder)
	}

	admin := stack.member(t, "restrictor@example.com", "restricting.admin")
	apitest.SetRole(t, stack.pool, "restricting.admin", "admin")
	restricted := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPut, "/v1/profiles/named.writer/restriction",
		`{"reason":"Under review"}`,
	), admin))
	if restricted.Code != http.StatusOK && restricted.Code != http.StatusNoContent {
		t.Fatalf("restrict status = %d: %s", restricted.Code, restricted.Body.String())
	}

	listed := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/grants", nil), stack.authority,
	))
	if listed.Code != http.StatusOK {
		t.Fatalf("list grants status = %d: %s", listed.Code, listed.Body.String())
	}
	var grants publicationGrantList
	if err := json.Unmarshal(listed.Body.Bytes(), &grants); err != nil {
		t.Fatalf("decode grants: %v", err)
	}
	holder := grants.Grants[0].Holder
	if !holder.Restricted || holder.DisplayName != "" || holder.Avatar != nil {
		t.Fatalf("a restricted contributor still reads as %+v", holder)
	}
	if holder.Handle != "named.writer" {
		t.Fatalf("a restricted contributor lost their handle: %+v", holder)
	}
}
