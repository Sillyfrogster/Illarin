package staff_test

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
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type restrictedProfile struct {
	apitest.PublicProfile
	Restricted bool `json:"restricted"`
}

type restrictedRecord struct {
	Reason       string    `json:"reason"`
	RestrictedBy *string   `json:"restrictedBy"`
	RestrictedAt time.Time `json:"restrictedAt"`
}

type restrictedStack struct {
	router  *gin.Engine
	pool    *pgxpool.Pool
	outbox  *apitest.VerificationOutbox
	works   *work.Service
	admin   *http.Cookie
	owner   *http.Cookie
	ownerID uuid.UUID
}

func (s restrictedStack) member(t *testing.T, email, handle string) *http.Cookie {
	t.Helper()
	return apitest.VerifiedSignUp(t, s.router, s.outbox, email, handle)
}

const ownerHandle = "shown.creator"

func newRestrictedStack(t *testing.T) restrictedStack {
	t.Helper()
	outbox := &apitest.VerificationOutbox{}
	router, pool, handlers := harness.NewRouterWithSenderPoolAndServices(t, 1<<20, api.DefaultDeadlines(), outbox)
	admin := apitest.VerifiedSignUp(t, router, outbox, "admin@example.com", "site.admin")
	apitest.SetRole(t, pool, "site.admin", "admin")
	owner := apitest.VerifiedSignUp(t, router, outbox, "owner@example.com", ownerHandle)
	return restrictedStack{
		router: router, pool: pool, outbox: outbox,
		works:   handlers.Works,
		admin:   admin,
		owner:   owner,
		ownerID: accountID(t, pool, ownerHandle),
	}
}

func accountID(t *testing.T, pool *pgxpool.Pool, handle string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(context.Background(),
		`select id from users where username = $1`, handle).Scan(&id)
	if err != nil {
		t.Fatalf("read %s: %v", handle, err)
	}
	return id
}

func (s restrictedStack) fillProfile(t *testing.T) {
	t.Helper()
	saved := apitest.SaveProfile(t, s.router, s.owner, `{
		"displayName":"Wren Ashdown",
		"biography":"Writes lorebooks about weather.",
		"contactEmail":"hello@example.com",
		"links":[{"label":"Notes","address":"https://example.com/notes"}]
	}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("fill profile status = %d, want 200: %s", saved.Code, saved.Body.String())
	}
	uploaded := apitest.Send(t, s.router, apitest.Authorized(
		apitest.AvatarUploadRequest(t, apitest.PNG(t, 200, 200)), s.owner,
	))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("avatar status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
}

func (s restrictedStack) restrict(
	t *testing.T,
	session *http.Cookie,
	handle, reason string,
) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(map[string]string{"reason": reason})
	if err != nil {
		t.Fatalf("encode reason: %v", err)
	}
	request := httptest.NewRequest(
		http.MethodPut, "/v1/profiles/"+handle+"/restricted", strings.NewReader(string(body)),
	)
	request.Header.Set("Content-Type", "application/json")
	return apitest.Send(t, s.router, apitest.Authorized(request, session))
}

func (s restrictedStack) restore(
	t *testing.T,
	session *http.Cookie,
	handle string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/profiles/"+handle+"/restricted", nil,
	), session))
}

func (s restrictedStack) readRestricted(
	t *testing.T,
	session *http.Cookie,
	handle string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+handle+"/restricted", nil,
	), session))
}

func (s restrictedStack) publicProfile(t *testing.T, handle string) restrictedProfile {
	t.Helper()
	response := apitest.Send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/profiles/"+handle, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read profile status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var found restrictedProfile
	if err := json.Unmarshal(response.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	return found
}

func TestRestrictingAProfileLeavesOnlyItsHandleAndItsWork(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	stack.fillProfile(t)
	apitest.CreateProfileWork(t, stack.works, stack.ownerID, "Fen weather", false, work.VisibilityListed)

	restricted := stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")
	if restricted.Code != http.StatusOK {
		t.Fatalf("restrict status = %d, want 200: %s", restricted.Code, restricted.Body.String())
	}

	shown := stack.publicProfile(t, ownerHandle)
	if shown.Handle != ownerHandle || !shown.Restricted {
		t.Fatalf("profile = %+v, want the handle and the restricted flag", shown)
	}
	if shown.DisplayName != "" || shown.Biography != "" || shown.ContactEmail != "" {
		t.Fatalf("restricted profile still carries text: %+v", shown)
	}
	if shown.Avatar != nil || len(shown.Links) != 0 {
		t.Fatalf("restricted profile still carries an avatar or links: %+v", shown)
	}

	listing := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/works?creator="+ownerHandle, nil,
	))
	if listing.Code != http.StatusOK {
		t.Fatalf("listing status = %d, want 200: %s", listing.Code, listing.Body.String())
	}
	var listed apitest.ProfileListingResponse
	if err := json.Unmarshal(listing.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode listing: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].Name != "Fen weather" {
		t.Fatalf("listing = %+v, want the restricted creator's published work", listed)
	}
	if listed.Items[0].Takedown != nil {
		t.Fatalf("restricting took down a work: %+v", listed.Items[0].Takedown)
	}
}

func TestTheCompletePublicResponseOfARestrictedProfileHidesNothingInIt(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	stack.fillProfile(t)
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	response := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+ownerHandle, nil,
	))
	body := response.Body.String()
	for _, hidden := range []string{
		"Wren Ashdown", "weather", "hello@example.com", "example.com/notes", "Impersonating",
	} {
		if strings.Contains(body, hidden) {
			t.Fatalf("the public response still contains %q: %s", hidden, body)
		}
	}
}

func TestOnlyAnAdminRestrictsOrRestoresAProfile(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	outsider := stack.member(t, "outsider@example.com", "ordinary.member")

	for _, role := range []string{"user", "moderator"} {
		apitest.SetRole(t, stack.pool, "ordinary.member", role)
		refused := stack.restrict(t, outsider, ownerHandle, "Because I say so.")
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s restrict status = %d, want 403: %s", role, refused.Code, refused.Body.String())
		}
	}
	if refused := stack.restrict(t, stack.owner, ownerHandle, "Hiding myself."); refused.Code != http.StatusForbidden {
		t.Fatalf("owner restrict status = %d, want 403: %s", refused.Code, refused.Body.String())
	}

	if allowed := stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator."); allowed.Code != http.StatusOK {
		t.Fatalf("admin restrict status = %d, want 200: %s", allowed.Code, allowed.Body.String())
	}
	if refused := stack.restore(t, stack.owner, ownerHandle); refused.Code != http.StatusForbidden {
		t.Fatalf("owner restore status = %d, want 403", refused.Code)
	}
	if allowed := stack.restore(t, stack.admin, ownerHandle); allowed.Code != http.StatusNoContent {
		t.Fatalf("admin restore status = %d, want 204: %s", allowed.Code, allowed.Body.String())
	}
}

func TestRestrictingNeedsAReasonAndOnlyAnAdminReadsItsRecord(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)

	if blank := stack.restrict(t, stack.admin, ownerHandle, "   "); blank.Code != http.StatusBadRequest {
		t.Fatalf("blank reason status = %d, want 400: %s", blank.Code, blank.Body.String())
	}
	if missing := stack.restrict(t, stack.admin, "nobody.here", "Impersonation."); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown account status = %d, want 404", missing.Code)
	}
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	read := stack.readRestricted(t, stack.admin, ownerHandle)
	if read.Code != http.StatusOK {
		t.Fatalf("admin read status = %d, want 200: %s", read.Code, read.Body.String())
	}
	var found restrictedRecord
	if err := json.Unmarshal(read.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode restricted profile: %v", err)
	}
	if found.Reason != "Impersonating another creator." {
		t.Fatalf("reason = %q", found.Reason)
	}
	if found.RestrictedBy == nil || *found.RestrictedBy != "site.admin" {
		t.Fatalf("restrictedBy = %v, want the acting admin", found.RestrictedBy)
	}

	if hidden := stack.readRestricted(t, stack.owner, ownerHandle); hidden.Code != http.StatusForbidden {
		t.Fatalf("owner reason status = %d, want 403: %s", hidden.Code, hidden.Body.String())
	}
	anonymous := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+ownerHandle+"/restricted", nil,
	))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("visitor reason status = %d, want 401", anonymous.Code)
	}
}

func TestARestrictedOwnerKeepsItsAccountAndLosesOnlyProfileEdits(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	stack.fillProfile(t)
	published := apitest.CreateProfileWork(
		t, stack.works, stack.ownerID, "Fen weather", false, work.VisibilityListed,
	)
	before := accountFactsOf(apitest.SessionState(t, stack.router, stack.owner))
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	blocked := apitest.SaveProfile(t, stack.router, stack.owner, `{
		"displayName":"Another name","biography":"","contactEmail":"","links":[]
	}`)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("restricted save status = %d, want 403: %s", blocked.Code, blocked.Body.String())
	}
	if strings.Contains(blocked.Body.String(), "Impersonating") {
		t.Fatalf("the refused save repeats the reason for restricting: %s", blocked.Body.String())
	}
	replaced := apitest.Send(t, stack.router, apitest.Authorized(
		apitest.AvatarUploadRequest(t, apitest.PNG(t, 200, 200)), stack.owner,
	))
	if replaced.Code != http.StatusForbidden {
		t.Fatalf("restricted avatar status = %d, want 403: %s", replaced.Code, replaced.Body.String())
	}
	removed := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/account/profile/avatar", nil,
	), stack.owner))
	if removed.Code != http.StatusForbidden {
		t.Fatalf("restricted avatar removal status = %d, want 403", removed.Code)
	}

	after := accountFactsOf(apitest.SessionState(t, stack.router, stack.owner))
	if after != before {
		t.Fatalf("restricting changed the account: %+v became %+v", before, after)
	}
	if role := stack.role(t); role != "user" {
		t.Fatalf("restricting changed the account role to %q", role)
	}
	stack.expectWorkUntouched(t, published)
	if apitest.CreateProfileWork(
		t, stack.works, stack.ownerID, "Still working", false, work.VisibilityListed,
	) == uuid.Nil {
		t.Fatal("a restricted owner could not publish")
	}
}

type accountFacts struct {
	Handle        string
	Email         string
	EmailVerified bool
	DiscordLinked bool
	HasPassword   bool
}

func accountFactsOf(state apitest.AccountState) accountFacts {
	facts := accountFacts{
		Handle:        state.Handle,
		EmailVerified: state.EmailVerified,
		DiscordLinked: state.DiscordLinked,
		HasPassword:   state.HasPassword,
	}
	if state.Email != nil {
		facts.Email = *state.Email
	}
	return facts
}

func (s restrictedStack) role(t *testing.T) string {
	t.Helper()
	var role string
	err := s.pool.QueryRow(context.Background(),
		`select role from users where id = $1`, s.ownerID).Scan(&role)
	if err != nil {
		t.Fatalf("read account role: %v", err)
	}
	return role
}

func (s restrictedStack) expectWorkUntouched(t *testing.T, workID uuid.UUID) {
	t.Helper()
	var owner uuid.UUID
	var visibility string
	var takenDown bool
	err := s.pool.QueryRow(context.Background(), `
		select owner_id, visibility, taken_down_at is not null from works where id = $1
	`, workID).Scan(&owner, &visibility, &takenDown)
	if err != nil {
		t.Fatalf("read work state: %v", err)
	}
	if owner != s.ownerID || visibility != "listed" || takenDown {
		t.Fatalf("restricting changed the work: owner %s, visibility %q, taken down %v",
			owner, visibility, takenDown)
	}
}

func TestRestoringGivesBackEveryRetainedField(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	stack.fillProfile(t)
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")
	if lifted := stack.restore(t, stack.admin, ownerHandle); lifted.Code != http.StatusNoContent {
		t.Fatalf("restore status = %d, want 204: %s", lifted.Code, lifted.Body.String())
	}

	shown := stack.publicProfile(t, ownerHandle)
	if shown.Restricted {
		t.Fatalf("profile is still restricted: %+v", shown)
	}
	if shown.DisplayName != "Wren Ashdown" || shown.Biography != "Writes lorebooks about weather." {
		t.Fatalf("restored profile = %+v", shown)
	}
	if shown.ContactEmail != "hello@example.com" || len(shown.Links) != 1 {
		t.Fatalf("restored contact = %+v, links = %+v", shown.ContactEmail, shown.Links)
	}
	if shown.Avatar == nil {
		t.Fatal("the avatar did not come back")
	}

	saved := apitest.SaveProfile(t, stack.router, stack.owner, `{
		"displayName":"Wren A.","biography":"","contactEmail":"","links":[]
	}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("restored save status = %d, want 200: %s", saved.Code, saved.Body.String())
	}
	if again := stack.readRestricted(t, stack.admin, ownerHandle); again.Code != http.StatusNotFound {
		t.Fatalf("restricted survived restoration: %d", again.Code)
	}
}

func TestRestrictingAndRestoringBothLeaveAnAuditRecord(t *testing.T) {
	t.Parallel()
	stack := newRestrictedStack(t)
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")
	stack.restore(t, stack.admin, ownerHandle)
	stack.restrict(t, stack.admin, ownerHandle, "Doing it again.")

	rows, err := stack.pool.Query(context.Background(), `
		select actor.username, audit.subject_id, audit.action, audit.recorded_at
		  from restricted_profile_audits audit
		  join users actor on actor.id = audit.actor_id
		 order by audit.recorded_at, audit.action
	`)
	if err != nil {
		t.Fatalf("read audits: %v", err)
	}
	defer rows.Close()
	var actions []string
	for rows.Next() {
		var actor, action string
		var subject uuid.UUID
		var at time.Time
		if err := rows.Scan(&actor, &subject, &action, &at); err != nil {
			t.Fatalf("read an audit: %v", err)
		}
		if actor != "site.admin" || subject != stack.ownerID || at.IsZero() {
			t.Fatalf("audit = %s %s %s %v", actor, subject, action, at)
		}
		actions = append(actions, action)
	}
	if len(actions) != 3 {
		t.Fatalf("audits = %v, want one per action", actions)
	}
}
