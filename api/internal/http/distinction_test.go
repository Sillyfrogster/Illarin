package http

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type distinctionMark struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type distinction struct {
	ID          string           `json:"id"`
	Form        string           `json:"form"`
	Name        string           `json:"name"`
	Explanation string           `json:"explanation"`
	Mark        *distinctionMark `json:"mark"`
	Position    int              `json:"position"`
	Retired     bool             `json:"retired"`
}

type distinctionList struct {
	Definitions []distinction `json:"definitions"`
}

type distinctionAssignment struct {
	ID          string      `json:"id"`
	Distinction distinction `json:"distinction"`
	IssuedBy    *string     `json:"issuedBy"`
	Source      string      `json:"source"`
	AssignedAt  time.Time   `json:"assignedAt"`
	Active      bool        `json:"active"`
	Position    int         `json:"position"`
}

type assignmentList struct {
	Handle      string                  `json:"handle"`
	Assignments []distinctionAssignment `json:"assignments"`
}

type profileDistinction struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Explanation string           `json:"explanation"`
	Mark        *distinctionMark `json:"mark"`
}

type distinctionStack struct {
	router    *gin.Engine
	pool      *pgxpool.Pool
	outbox    *verificationOutbox
	authority *http.Cookie
}

func newDistinctionStack(t *testing.T) distinctionStack {
	t.Helper()
	outbox := &verificationOutbox{}
	router, pool, _ := newTestRouterWithSenderPoolAndHandlers(t, 1<<20, DefaultDeadlines(), outbox)
	session := verifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	holdsAuthority(t, pool, "publication.authority")
	return distinctionStack{router: router, pool: pool, outbox: outbox, authority: session}
}

func (s distinctionStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return verifiedSignUp(t, s.router, s.outbox, email, handle)
}

func holdsAuthority(t *testing.T, pool *pgxpool.Pool, handle string) {
	t.Helper()
	_, err := pool.Exec(context.Background(), `
		insert into publication_authorities (user_id)
		select id from users where username = $1
	`, handle)
	if err != nil {
		t.Fatalf("assign publication authority: %v", err)
	}
}

func setRole(t *testing.T, pool *pgxpool.Pool, handle, role string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		update users set role = $2 where username = $1
	`, handle, role); err != nil {
		t.Fatalf("set %s role: %v", handle, err)
	}
}

func (s distinctionStack) define(
	t *testing.T,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/v1/distinctions", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return send(t, s.router, authorized(request, session))
}

func (s distinctionStack) defined(t *testing.T, form, name, explanation string) distinction {
	t.Helper()
	body, err := json.Marshal(map[string]string{
		"form": form, "name": name, "explanation": explanation,
	})
	if err != nil {
		t.Fatalf("encode definition: %v", err)
	}
	response := s.define(t, s.authority, string(body))
	if response.Code != http.StatusCreated {
		t.Fatalf("define %s status = %d, want 201: %s", name, response.Code, response.Body.String())
	}
	var created distinction
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode definition: %v", err)
	}
	return created
}

func (s distinctionStack) assign(
	t *testing.T,
	session *http.Cookie,
	handle, distinctionID string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodPost,
		"/v1/accounts/"+handle+"/distinctions",
		strings.NewReader(`{"distinctionId":"`+distinctionID+`"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	return send(t, s.router, authorized(request, session))
}

func (s distinctionStack) assigned(t *testing.T, handle, distinctionID string) distinctionAssignment {
	t.Helper()
	response := s.assign(t, s.authority, handle, distinctionID)
	if response.Code != http.StatusCreated {
		t.Fatalf("assign status = %d, want 201: %s", response.Code, response.Body.String())
	}
	var made distinctionAssignment
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode assignment: %v", err)
	}
	return made
}

func (s distinctionStack) profile(t *testing.T, handle string) struct {
	Positions []profileDistinction `json:"positions"`
	Titles    []profileDistinction `json:"titles"`
	Badges    []profileDistinction `json:"badges"`
} {
	t.Helper()
	var shown struct {
		Positions []profileDistinction `json:"positions"`
		Titles    []profileDistinction `json:"titles"`
		Badges    []profileDistinction `json:"badges"`
	}
	response := send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/profiles/"+handle, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read profile status = %d: %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &shown); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return shown
}

func TestOnlyThePublicationAuthorityManagesDistinctions(t *testing.T) {
	stack := newDistinctionStack(t)
	outsider := stack.member(t, "other@example.com", "ordinary.member")

	for _, role := range []string{"user", "moderator", "admin"} {
		setRole(t, stack.pool, "ordinary.member", role)
		refused := stack.define(t, outsider, `{"form":"position","name":"Founder"}`)
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s define status = %d, want 403: %s", role, refused.Code, refused.Body.String())
		}
		listed := send(t, stack.router, authorized(
			httptest.NewRequest(http.MethodGet, "/v1/distinctions", nil), outsider,
		))
		if listed.Code != http.StatusForbidden {
			t.Fatalf("%s list status = %d, want 403", role, listed.Code)
		}
	}

	allowed := stack.define(t, stack.authority, `{"form":"position","name":"Founder"}`)
	if allowed.Code != http.StatusCreated {
		t.Fatalf("authority define status = %d, want 201: %s", allowed.Code, allowed.Body.String())
	}
}

func TestTheAuthorityDefinesRenamesOrdersAndRetiresDistinctions(t *testing.T) {
	stack := newDistinctionStack(t)
	founder := stack.defined(t, "position", "Founder", "")
	keeper := stack.defined(t, "position", "Catalogue keeper", "")

	renamed := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPatch, "/v1/distinctions/"+keeper.ID, `{"name":"Catalogue steward"}`,
	), stack.authority))
	if renamed.Code != http.StatusOK {
		t.Fatalf("rename status = %d: %s", renamed.Code, renamed.Body.String())
	}

	ordered := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/distinctions",
		`{"form":"position","distinctionIds":["`+keeper.ID+`","`+founder.ID+`"]}`,
	), stack.authority))
	if ordered.Code != http.StatusOK {
		t.Fatalf("order status = %d: %s", ordered.Code, ordered.Body.String())
	}
	var list distinctionList
	if err := json.Unmarshal(ordered.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode ordered list: %v", err)
	}
	positions := make([]distinction, 0, 2)
	for _, defined := range list.Definitions {
		if defined.Form == "position" {
			positions = append(positions, defined)
		}
	}
	if len(positions) != 2 ||
		positions[0].Name != "Catalogue steward" ||
		positions[1].Name != "Founder" {
		t.Fatalf("positions = %+v", positions)
	}

	stack.member(t, "holder@example.com", "position.holder")
	stack.assigned(t, "position.holder", founder.ID)
	stack.assigned(t, "position.holder", keeper.ID)

	shown := stack.profile(t, "position.holder")
	if len(shown.Positions) != 2 ||
		shown.Positions[0].Name != "Catalogue steward" ||
		shown.Positions[1].Name != "Founder" {
		t.Fatalf("public positions = %+v", shown.Positions)
	}

	retired := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPatch, "/v1/distinctions/"+founder.ID, `{"retired":true}`,
	), stack.authority))
	if retired.Code != http.StatusOK {
		t.Fatalf("retire status = %d: %s", retired.Code, retired.Body.String())
	}

	after := stack.profile(t, "position.holder")
	if len(after.Positions) != 1 || after.Positions[0].Name != "Catalogue steward" {
		t.Fatalf("positions after retirement = %+v", after.Positions)
	}

	var kept int
	if err := stack.pool.QueryRow(context.Background(), `
		select count(*) from profile_distinction_assignments where active
	`).Scan(&kept); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if kept != 2 {
		t.Fatalf("retiring a definition changed %d active assignments, want both kept", kept)
	}
}

func TestAProfileShowsEveryPositionAndABoundedTitleAndBadgeShowcase(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "showcase@example.com", "showcase.member")

	for _, name := range []string{"Founder", "Catalogue steward", "Release editor"} {
		stack.assigned(t, "showcase.member", stack.defined(t, "position", name, "").ID)
	}
	titleNames := []string{
		"Archivist", "Cartographer", "Lorekeeper", "Wayfinder",
		"Binder", "Almanac keeper", "Weather clerk", "Ferrier",
	}
	titles := make([]string, 0, len(titleNames))
	for _, name := range titleNames {
		made := stack.defined(t, "title", name, "")
		stack.assigned(t, "showcase.member", made.ID)
		titles = append(titles, made.ID)
	}
	for _, name := range []string{"First light", "Long service", "Field report", "Quiet fix"} {
		stack.assigned(t, "showcase.member", stack.defined(t, "badge", name, "Given by hand.").ID)
	}

	shown := stack.profile(t, "showcase.member")
	if len(shown.Positions) != 3 {
		t.Fatalf("positions = %+v, want every one", shown.Positions)
	}
	if len(shown.Titles) != 6 || len(shown.Badges) != 4 {
		t.Fatalf("showcase = %d titles and %d badges, want 6 and 4", len(shown.Titles), len(shown.Badges))
	}
	if shown.Titles[0].Name != "Archivist" {
		t.Fatalf("titles = %+v", shown.Titles)
	}

	reordered := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/accounts/showcase.member/distinctions",
		`{"assignmentIds":`+assignmentOrder(t, stack, "showcase.member", titles[4], titles[0])+`}`,
	), stack.authority))
	if reordered.Code != http.StatusOK {
		t.Fatalf("assignment order status = %d: %s", reordered.Code, reordered.Body.String())
	}
	after := stack.profile(t, "showcase.member")
	if after.Titles[0].Name != "Binder" {
		t.Fatalf("titles after reorder = %+v", after.Titles)
	}
}

func TestAnAccountOwnerCannotAwardHideOrReorderTheirOwnDistinctions(t *testing.T) {
	stack := newDistinctionStack(t)
	owner := stack.member(t, "owner@example.com", "profile.owner")
	badge := stack.defined(t, "badge", "First light", "Given by hand.")
	assignment := stack.assigned(t, "profile.owner", badge.ID)

	awarded := stack.assign(t, owner, "profile.owner", badge.ID)
	if awarded.Code != http.StatusForbidden {
		t.Fatalf("self award status = %d, want 403: %s", awarded.Code, awarded.Body.String())
	}
	hidden := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/accounts/profile.owner/distinctions/"+assignment.ID, nil,
	), owner))
	if hidden.Code != http.StatusForbidden {
		t.Fatalf("self removal status = %d, want 403", hidden.Code)
	}
	reordered := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/accounts/profile.owner/distinctions",
		`{"assignmentIds":["`+assignment.ID+`"]}`,
	), owner))
	if reordered.Code != http.StatusForbidden {
		t.Fatalf("self reorder status = %d, want 403", reordered.Code)
	}

	shown := stack.profile(t, "profile.owner")
	if len(shown.Badges) != 1 || shown.Badges[0].Name != "First light" {
		t.Fatalf("badges = %+v", shown.Badges)
	}
}

func TestRemovingAnAssignmentHidesItAndKeepsItsAccountableHistory(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "held@example.com", "held.member")
	title := stack.defined(t, "title", "Archivist", "")
	assignment := stack.assigned(t, "held.member", title.ID)

	removed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/accounts/held.member/distinctions/"+assignment.ID, nil,
	), stack.authority))
	if removed.Code != http.StatusNoContent {
		t.Fatalf("remove status = %d, want 204: %s", removed.Code, removed.Body.String())
	}
	if shown := stack.profile(t, "held.member"); len(shown.Titles) != 0 {
		t.Fatalf("removed title is still public: %+v", shown.Titles)
	}

	var issuer *uuid.UUID
	var source string
	var assignedAt, deactivatedAt *time.Time
	err := stack.pool.QueryRow(context.Background(), `
		select issued_by, source, assigned_at, deactivated_at
		  from profile_distinction_assignments where id = $1
	`, assignment.ID).Scan(&issuer, &source, &assignedAt, &deactivatedAt)
	if err != nil {
		t.Fatalf("read removed assignment: %v", err)
	}
	if issuer == nil || source != "manual" || assignedAt == nil || deactivatedAt == nil {
		t.Fatalf("history = issuer %v, source %q, assigned %v, deactivated %v",
			issuer, source, assignedAt, deactivatedAt)
	}

	listed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/accounts/held.member/distinctions", nil,
	), stack.authority))
	var held assignmentList
	if err := json.Unmarshal(listed.Body.Bytes(), &held); err != nil {
		t.Fatalf("decode assignment list: %v", err)
	}
	if len(held.Assignments) != 1 || held.Assignments[0].Active {
		t.Fatalf("authority view = %+v", held.Assignments)
	}
}

func TestATakenBackDistinctionCanBeGivenAgainWithoutLosingTheFirstRecord(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "again@example.com", "second.chance")
	badge := stack.defined(t, "badge", "First light", "Given by hand.")
	first := stack.assigned(t, "second.chance", badge.ID)

	send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/accounts/second.chance/distinctions/"+first.ID, nil,
	), stack.authority))

	second := stack.assigned(t, "second.chance", badge.ID)
	if second.ID == first.ID {
		t.Fatalf("giving it again reused the withdrawn record %s", first.ID)
	}
	if shown := stack.profile(t, "second.chance"); len(shown.Badges) != 1 {
		t.Fatalf("badges = %+v, want the one it holds now", shown.Badges)
	}

	var records int
	if err := stack.pool.QueryRow(context.Background(), `
		select count(*) from profile_distinction_assignments assignment
		  join users account on account.id = assignment.user_id
		 where account.username = 'second.chance'
	`).Scan(&records); err != nil {
		t.Fatalf("count assignments: %v", err)
	}
	if records != 2 {
		t.Fatalf("account kept %d assignment records, want both", records)
	}
}

func TestADistinctionSatisfiesNoPermissionCheck(t *testing.T) {
	stack := newDistinctionStack(t)
	member := stack.member(t, "titled@example.com", "titled.member")
	for _, name := range []string{"Administrator", "Moderator", "Publication authority"} {
		stack.assigned(t, "titled.member", stack.defined(t, "title", name, "").ID)
	}

	refused := stack.define(t, member, `{"form":"title","name":"Self made"}`)
	if refused.Code != http.StatusForbidden {
		t.Fatalf("titled define status = %d, want 403", refused.Code)
	}
	listed := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodGet, "/v1/accounts/publication.authority/distinctions", nil), member,
	))
	if listed.Code != http.StatusForbidden {
		t.Fatalf("titled read of another account status = %d, want 403", listed.Code)
	}

	withheld := send(t, stack.router, authorized(jsonRequest(t,
		http.MethodPut, "/v1/assets/"+uuid.New().String()+"/withhold",
		`{"reason":"Titled, not an admin"}`,
	), member))
	if withheld.Code != http.StatusForbidden {
		t.Fatalf("titled withhold status = %d, want 403: %s", withheld.Code, withheld.Body.String())
	}

	var role string
	if err := stack.pool.QueryRow(context.Background(), `
		select role from users where username = 'titled.member'
	`).Scan(&role); err != nil {
		t.Fatalf("read role: %v", err)
	}
	if role != "user" {
		t.Fatalf("a distinction changed the account role to %q", role)
	}
	var authorities int
	if err := stack.pool.QueryRow(context.Background(), `
		select count(*) from publication_authorities authority
		  join users account on account.id = authority.user_id
		 where account.username = 'titled.member'
	`).Scan(&authorities); err != nil {
		t.Fatalf("count authority rows: %v", err)
	}
	if authorities != 0 {
		t.Fatalf("a distinction created %d authority assignments", authorities)
	}
}

func TestABadgeMarkIsHostedByIllarinAndServedOnTheSharedMediaPath(t *testing.T) {
	stack := newDistinctionStack(t)
	badge := stack.defined(t, "badge", "First light", "Given by hand.")
	stack.member(t, "marked@example.com", "marked.member")
	stack.assigned(t, "marked.member", badge.ID)

	uploaded := send(t, stack.router, authorized(
		markUploadRequest(t, badge.ID, httpTestPNG(t, 240, 240)), stack.authority,
	))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("mark status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
	var marked distinction
	if err := json.Unmarshal(uploaded.Body.Bytes(), &marked); err != nil {
		t.Fatalf("decode marked badge: %v", err)
	}
	if marked.Mark == nil || marked.Mark.Width != 240 || marked.Mark.Height != 240 {
		t.Fatalf("mark = %+v", marked.Mark)
	}

	image := send(t, stack.router, httptest.NewRequest(http.MethodGet, marked.Mark.URL, nil))
	if image.Code != http.StatusOK {
		t.Fatalf("mark image status = %d, want 200: %s", image.Code, image.Body.String())
	}
	if image.Header().Get("X-Accel-Redirect") == "" {
		t.Fatalf("mark image did not hand off to the byte server: %+v", image.Header())
	}

	shown := stack.profile(t, "marked.member")
	if len(shown.Badges) != 1 || shown.Badges[0].Mark == nil {
		t.Fatalf("public badge = %+v", shown.Badges)
	}
	if shown.Badges[0].Mark.URL != marked.Mark.URL {
		t.Fatalf("public mark address = %q, want %q", shown.Badges[0].Mark.URL, marked.Mark.URL)
	}

	position := stack.defined(t, "position", "Founder", "")
	refused := send(t, stack.router, authorized(
		markUploadRequest(t, position.ID, httpTestPNG(t, 240, 240)), stack.authority,
	))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("position mark status = %d, want 400", refused.Code)
	}
}

func TestAMarkMakesATitleABadgeAndRemovingItMakesItATitleAgain(t *testing.T) {
	stack := newDistinctionStack(t)
	title := stack.defined(t, "title", "First light", "Published a first asset.")
	stack.member(t, "promoted@example.com", "promoted.member")
	stack.assigned(t, "promoted.member", title.ID)

	uploaded := send(t, stack.router, authorized(
		markUploadRequest(t, title.ID, httpTestPNG(t, 240, 240)), stack.authority,
	))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("mark status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
	var promoted distinction
	if err := json.Unmarshal(uploaded.Body.Bytes(), &promoted); err != nil {
		t.Fatalf("decode the marked recognition: %v", err)
	}
	if promoted.Form != "badge" || promoted.Mark == nil {
		t.Fatalf("marked recognition = %+v", promoted)
	}

	shown := stack.profile(t, "promoted.member")
	if len(shown.Badges) != 1 || len(shown.Titles) != 0 {
		t.Fatalf("profile after marking = %d badges, %d titles", len(shown.Badges), len(shown.Titles))
	}

	cleared := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/distinctions/"+title.ID+"/mark", nil),
		stack.authority,
	))
	if cleared.Code != http.StatusOK {
		t.Fatalf("clear status = %d, want 200: %s", cleared.Code, cleared.Body.String())
	}
	var demoted distinction
	if err := json.Unmarshal(cleared.Body.Bytes(), &demoted); err != nil {
		t.Fatalf("decode the unmarked recognition: %v", err)
	}
	if demoted.Form != "title" || demoted.Mark != nil {
		t.Fatalf("unmarked recognition = %+v", demoted)
	}

	after := stack.profile(t, "promoted.member")
	if len(after.Titles) != 1 || len(after.Badges) != 0 {
		t.Fatalf("profile after clearing = %d badges, %d titles", len(after.Badges), len(after.Titles))
	}

	position := stack.defined(t, "position", "Founder", "")
	refused := send(t, stack.router, authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/distinctions/"+position.ID+"/mark", nil),
		stack.authority,
	))
	if refused.Code != http.StatusBadRequest {
		t.Fatalf("clearing a position's mark = %d, want 400", refused.Code)
	}
}

func TestEveryDistinctionChangeIsAuditedWithoutCopyingProfileContent(t *testing.T) {
	stack := newDistinctionStack(t)
	stack.member(t, "audited@example.com", "audited.member")
	title := stack.defined(t, "title", "Archivist", "Keeps the shelves in order.")
	assignment := stack.assigned(t, "audited.member", title.ID)
	send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/accounts/audited.member/distinctions/"+assignment.ID, nil,
	), stack.authority))

	rows, err := stack.pool.Query(context.Background(), `
		select action, actor_id is not null, distinction_id is not null, subject_id is not null
		  from profile_distinction_audits order by recorded_at, action
	`)
	if err != nil {
		t.Fatalf("read audits: %v", err)
	}
	defer rows.Close()
	actions := make([]string, 0, 3)
	for rows.Next() {
		var action string
		var hasActor, hasDistinction, hasSubject bool
		if err := rows.Scan(&action, &hasActor, &hasDistinction, &hasSubject); err != nil {
			t.Fatalf("read audit row: %v", err)
		}
		if !hasActor {
			t.Fatalf("audit %q recorded no actor", action)
		}
		if !hasDistinction {
			t.Fatalf("audit %q recorded no distinction", action)
		}
		_ = hasSubject
		actions = append(actions, action)
	}
	if len(actions) != 3 {
		t.Fatalf("audited actions = %v, want one for each change", actions)
	}

	var copied int
	if err := stack.pool.QueryRow(context.Background(), `
		select count(*) from profile_distinction_audits
		 where action like '%Archivist%' or action like '%audited%'
	`).Scan(&copied); err != nil {
		t.Fatalf("search audits: %v", err)
	}
	if copied != 0 {
		t.Fatalf("%d audit rows copied profile content", copied)
	}
}

func assignmentOrder(t *testing.T, stack distinctionStack, handle string, leading ...string) string {
	t.Helper()
	listed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/accounts/"+handle+"/distinctions", nil,
	), stack.authority))
	var held assignmentList
	if err := json.Unmarshal(listed.Body.Bytes(), &held); err != nil {
		t.Fatalf("decode assignment list: %v", err)
	}
	byDistinction := make(map[string]string, len(held.Assignments))
	ordered := make([]string, 0, len(held.Assignments))
	for _, assignment := range held.Assignments {
		byDistinction[assignment.Distinction.ID] = assignment.ID
	}
	wanted := make(map[string]bool, len(leading))
	for _, distinctionID := range leading {
		id, ok := byDistinction[distinctionID]
		if !ok {
			t.Fatalf("no assignment for distinction %s", distinctionID)
		}
		ordered = append(ordered, id)
		wanted[id] = true
	}
	for _, assignment := range held.Assignments {
		if !wanted[assignment.ID] {
			ordered = append(ordered, assignment.ID)
		}
	}
	encoded, err := json.Marshal(ordered)
	if err != nil {
		t.Fatalf("encode assignment order: %v", err)
	}
	return string(encoded)
}

func jsonRequest(t *testing.T, method, target, body string) *http.Request {
	t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	return request
}

func markUploadRequest(t *testing.T, distinctionID string, file []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	writeFilePartNamed(t, form, "mark.png", file)
	if err := form.Close(); err != nil {
		t.Fatalf("close mark form: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPut, "/v1/distinctions/"+distinctionID+"/mark", &body,
	)
	request.Header.Set("Content-Type", form.FormDataContentType())
	return request
}
