package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type publicationApp struct {
	ID       string           `json:"id"`
	Slug     string           `json:"slug"`
	Name     string           `json:"name"`
	Home     string           `json:"home"`
	Mark     *distinctionMark `json:"mark"`
	Position int              `json:"position"`
	Retired  bool             `json:"retired"`
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

// profileAvatar carries the same url, width and height a hosted mark does.
type profileAvatar = distinctionMark

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

func (s distinctionStack) apps(t *testing.T) []publicationApp {
	t.Helper()
	response := send(t, s.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/apps", nil), s.authority,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list apps status = %d: %s", response.Code, response.Body.String())
	}
	var listed publicationAppList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode apps: %v", err)
	}
	return listed.Apps
}

func (s distinctionStack) categories(t *testing.T) []publicationCategory {
	t.Helper()
	response := send(t, s.router, authorized(
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

func (s distinctionStack) categoryBySlug(t *testing.T, slug string) publicationCategory {
	t.Helper()
	for _, category := range s.categories(t) {
		if category.Slug == slug {
			return category
		}
	}
	t.Fatalf("no seeded category with the slug %q", slug)
	return publicationCategory{}
}

func (s distinctionStack) appBySlug(t *testing.T, slug string) publicationApp {
	t.Helper()
	for _, app := range s.apps(t) {
		if app.Slug == slug {
			return app
		}
	}
	t.Fatalf("no configured app with the slug %q", slug)
	return publicationApp{}
}

func (s distinctionStack) configureApp(t *testing.T, slug, name, home string) publicationApp {
	t.Helper()
	response := send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/apps",
		`{"slug":"`+slug+`","name":"`+name+`","home":"`+home+`"}`,
	), s.authority))
	if response.Code != http.StatusCreated {
		t.Fatalf("configure %s status = %d: %s", slug, response.Code, response.Body.String())
	}
	var configured publicationApp
	if err := json.Unmarshal(response.Body.Bytes(), &configured); err != nil {
		t.Fatalf("decode app: %v", err)
	}
	return configured
}

func (s distinctionStack) approve(
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
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/grants", string(body),
	), session))
}

func (s distinctionStack) approved(
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

func (s distinctionStack) workspace(t *testing.T, session *http.Cookie) publicationWorkspace {
	t.Helper()
	response := send(t, s.router, authorized(
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
	stack := newDistinctionStack(t)

	illarin := stack.appBySlug(t, "illarin")
	if illarin.Name != "Illarin" || illarin.Home != "https://illarin.xyz" {
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

func TestOnlyThePublicationAuthorityManagesAppsCategoriesAndGrants(t *testing.T) {
	stack := newDistinctionStack(t)
	outsider := stack.member(t, "outsider@example.com", "publication.outsider")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")

	reads := []string{
		"/v1/publication/apps",
		"/v1/publication/categories",
		"/v1/publication/grants",
	}
	for _, role := range []string{"user", "moderator", "admin"} {
		setRole(t, stack.pool, "publication.outsider", role)
		for _, path := range reads {
			refused := send(t, stack.router, authorized(
				httptest.NewRequest(http.MethodGet, path, nil), outsider,
			))
			if refused.Code != http.StatusForbidden {
				t.Fatalf("%s read of %s = %d, want 403", role, path, refused.Code)
			}
		}
		configured := send(t, stack.router, authorized(jsonRequest(t,
			http.MethodPost, "/v1/publication/apps",
			`{"slug":"lumiverse","name":"Lumiverse","home":"https://lumiverse.example"}`,
		), outsider))
		if configured.Code != http.StatusForbidden {
			t.Fatalf("%s configure app = %d, want 403: %s", role, configured.Code, configured.Body.String())
		}
		relabelled := send(t, stack.router, authorized(jsonRequest(t,
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

func TestTheAuthorityConfiguresOrdersAndRetiresApps(t *testing.T) {
	stack := newDistinctionStack(t)
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	illarin := stack.appBySlug(t, "illarin")

	taken := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/apps",
		`{"slug":"lumiverse","name":"Lumiverse again","home":"https://lumiverse.example"}`,
	), stack.authority))
	if taken.Code != http.StatusConflict {
		t.Fatalf("duplicate slug status = %d, want 409: %s", taken.Code, taken.Body.String())
	}

	insecure := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/apps",
		`{"slug":"tavern","name":"Tavern","home":"http://tavern.example"}`,
	), stack.authority))
	if insecure.Code != http.StatusBadRequest {
		t.Fatalf("plain http address status = %d, want 400", insecure.Code)
	}

	ordered := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/publication/apps",
		`{"appIds":["`+lumiverse.ID+`","`+illarin.ID+`"]}`,
	), stack.authority))
	if ordered.Code != http.StatusOK {
		t.Fatalf("order apps status = %d: %s", ordered.Code, ordered.Body.String())
	}
	if listed := stack.apps(t); listed[0].Slug != "lumiverse" || listed[1].Slug != "illarin" {
		t.Fatalf("apps after ordering = %+v", listed)
	}

	retired := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPatch, "/v1/publication/apps/"+lumiverse.ID, `{"retired":true}`,
	), stack.authority))
	if retired.Code != http.StatusOK {
		t.Fatalf("retire app status = %d: %s", retired.Code, retired.Body.String())
	}
	if stack.appBySlug(t, "lumiverse").Retired != true {
		t.Fatal("the retired app does not read as retired")
	}
}

func TestConfiguringAnAppGrantsNobodyAnything(t *testing.T) {
	stack := newDistinctionStack(t)
	developer := stack.member(t, "developer@example.com", "lumiverse.developer")
	stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")

	if open := stack.workspace(t, developer); len(open.Grants) != 0 {
		t.Fatalf("an app alone opened a workspace of %+v", open.Grants)
	}
	if shown := stack.profile(t, "lumiverse.developer"); len(shown.Badges) != 0 {
		t.Fatalf("an app alone put %+v on a profile", shown.Badges)
	}
	refused := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/grants", nil), developer,
	))
	if refused.Code != http.StatusForbidden {
		t.Fatalf("an app alone let a developer read grants: %d", refused.Code)
	}
}

func TestTheAuthorityRelabelsAndOrdersCategories(t *testing.T) {
	stack := newDistinctionStack(t)
	announcement := stack.categoryBySlug(t, "announcement")
	article := stack.categoryBySlug(t, "article")
	release := stack.categoryBySlug(t, "release")

	relabelled := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPatch, "/v1/publication/categories/"+announcement.ID, `{"label":"Update"}`,
	), stack.authority))
	if relabelled.Code != http.StatusOK {
		t.Fatalf("relabel status = %d: %s", relabelled.Code, relabelled.Body.String())
	}

	ordered := send(t, stack.router, authorized(jsonRequest(t,
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
	stack := newDistinctionStack(t)
	stack.member(t, "writer@example.com", "lumiverse.writer")
	signUp(t, stack.router, "unverified@example.com", "unverified.writer")
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
	stack := newDistinctionStack(t)
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

	refused := send(t, stack.router, authorized(
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
	stack := newDistinctionStack(t)
	first := stack.member(t, "first@example.com", "first.developer")
	second := stack.member(t, "second@example.com", "second.developer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")

	held := stack.approved(
		t, "first.developer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	stack.approved(t, "second.developer", lumiverse.ID, []string{announcement.ID}, announcement.ID)

	revoked := send(t, stack.router, authorized(
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
	if shown := stack.profile(t, "second.developer"); len(shown.Badges) != 1 {
		t.Fatalf("the other contributor's badges = %+v", shown.Badges)
	}
}

func TestAnActiveGrantCarriesTheContributorBadge(t *testing.T) {
	stack := newDistinctionStack(t)
	writer := stack.member(t, "badged@example.com", "badged.writer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")

	first := stack.approved(
		t, "badged.writer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	shown := stack.profile(t, "badged.writer")
	if len(shown.Badges) != 1 || shown.Badges[0].Name != "Verified App Contributor" {
		t.Fatalf("badges after the grant = %+v", shown.Badges)
	}

	second := stack.approved(
		t, "badged.writer", illarin.ID, []string{announcement.ID}, announcement.ID,
	)
	if after := stack.profile(t, "badged.writer"); len(after.Badges) != 1 {
		t.Fatalf("a second grant gave a second badge: %+v", after.Badges)
	}

	for _, grantID := range []string{first.ID, second.ID} {
		revoked := send(t, stack.router, authorized(
			httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+grantID, nil),
			stack.authority,
		))
		if revoked.Code != http.StatusNoContent {
			t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
		}
	}

	if after := stack.profile(t, "badged.writer"); len(after.Badges) != 0 {
		t.Fatalf("badges after every grant went = %+v", after.Badges)
	}
	if open := stack.workspace(t, writer); len(open.Grants) != 0 {
		t.Fatalf("editor access outlived the grants: %+v", open.Grants)
	}

	var records int
	err := stack.pool.QueryRow(context.Background(), `
		select count(*) from profile_distinction_assignments assignment
		  join users account on account.id = assignment.user_id
		 where account.username = 'badged.writer' and assignment.source = 'publication-grant'
	`).Scan(&records)
	if err != nil {
		t.Fatalf("count badge assignments: %v", err)
	}
	if records != 1 {
		t.Fatalf("the badge left %d records, want the one it made", records)
	}
}

func TestRevocationKeepsTheAccountAndTheGrantRecord(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "ended@example.com", "ended.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	held := stack.approved(
		t, "ended.writer", illarin.ID, []string{announcement.ID}, announcement.ID,
	)

	revoked := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if revoked.Code != http.StatusNoContent {
		t.Fatalf("revoke status = %d: %s", revoked.Code, revoked.Body.String())
	}
	twice := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/publication/grants/"+held.ID, nil), stack.authority,
	))
	if twice.Code != http.StatusNotFound {
		t.Fatalf("second revoke status = %d, want 404", twice.Code)
	}

	listed := send(t, stack.router, authorized(
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
	stack := newDistinctionStack(t)
	writer := stack.member(t, "bounded@example.com", "bounded.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")
	stack.approved(t, "bounded.writer", illarin.ID, []string{announcement.ID}, announcement.ID)

	elsewhere := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/v1/distinctions", ""},
		{http.MethodPost, "/v1/publication/apps", `{"slug":"x","name":"X","home":"https://x.example"}`},
		{http.MethodGet, "/v1/publication/grants", ""},
		{http.MethodGet, "/v1/accounts/publication.authority/distinctions", ""},
	}
	for _, refused := range elsewhere {
		var request *http.Request
		if refused.body == "" {
			request = httptest.NewRequest(refused.method, refused.path, nil)
		} else {
			request = jsonRequest(t, refused.method, refused.path, refused.body)
		}
		response := send(t, stack.router, authorized(request, writer))
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

func TestAppCategoryAndGrantChangesLeaveSafeIdentifiersBehind(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "audited@example.com", "audited.writer")
	lumiverse := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	held := stack.approved(
		t, "audited.writer", lumiverse.ID, []string{announcement.ID}, announcement.ID,
	)
	revoked := send(t, stack.router, authorized(
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
	want := []string{"app.defined", "grant.created", "grant.revoked"}
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
		   and column_name <> 'action'
	`).Scan(&columns)
	if err != nil {
		t.Fatalf("read audit columns: %v", err)
	}
	if columns != 0 {
		t.Fatalf("the audit table has %d columns that could hold copied content", columns)
	}
}

func TestAGrantNamesItsContributorTheWayTheirProfileDoes(t *testing.T) {
	stack := newDistinctionStack(t)
	writer := stack.member(t, "named@example.com", "named.writer")
	illarin := stack.appBySlug(t, "illarin")
	announcement := stack.categoryBySlug(t, "announcement")

	saved := send(t, stack.router, authorized(jsonRequest(t,
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
	setRole(t, stack.pool, "restricting.admin", "admin")
	restricted := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/profiles/named.writer/restriction",
		`{"reason":"Under review"}`,
	), admin))
	if restricted.Code != http.StatusOK && restricted.Code != http.StatusNoContent {
		t.Fatalf("restrict status = %d: %s", restricted.Code, restricted.Body.String())
	}

	listed := send(t, stack.router, authorized(
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
