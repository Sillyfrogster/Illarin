package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/asset"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type restrictedProfile struct {
	publicProfile
	Positions  []profileDistinction `json:"positions"`
	Titles     []profileDistinction `json:"titles"`
	Badges     []profileDistinction `json:"badges"`
	Restricted bool                 `json:"restricted"`
}

type profileRestriction struct {
	Reason       string    `json:"reason"`
	RestrictedBy *string   `json:"restrictedBy"`
	RestrictedAt time.Time `json:"restrictedAt"`
}

type restrictionStack struct {
	distinctionStack
	assets  *asset.Service
	admin   *http.Cookie
	owner   *http.Cookie
	ownerID uuid.UUID
}

const ownerHandle = "shown.creator"

func newRestrictionStack(t *testing.T) restrictionStack {
	t.Helper()
	outbox := &verificationOutbox{}
	router, pool, handlers := newTestRouterWithSenderPoolAndHandlers(t, 1<<20, DefaultDeadlines(), outbox)
	authority := verifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	holdsAuthority(t, pool, "publication.authority")
	admin := verifiedSignUp(t, router, outbox, "admin@example.com", "site.admin")
	setRole(t, pool, "site.admin", "admin")
	owner := verifiedSignUp(t, router, outbox, "owner@example.com", ownerHandle)
	return restrictionStack{
		distinctionStack: distinctionStack{
			router: router, pool: pool, outbox: outbox, authority: authority,
		},
		assets:  handlers.assets,
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

func (s restrictionStack) fillProfile(t *testing.T) {
	t.Helper()
	saved := saveProfile(t, s.router, s.owner, `{
		"displayName":"Wren Ashdown",
		"biography":"Writes lorebooks about weather.",
		"contactEmail":"hello@example.com",
		"links":[{"label":"Notes","address":"https://example.com/notes"}]
	}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("fill profile status = %d, want 200: %s", saved.Code, saved.Body.String())
	}
	uploaded := send(t, s.router, authorized(
		avatarUploadRequest(t, httpTestPNG(t, 200, 200)), s.owner,
	))
	if uploaded.Code != http.StatusOK {
		t.Fatalf("avatar status = %d, want 200: %s", uploaded.Code, uploaded.Body.String())
	}
	for _, given := range []struct{ form, name string }{
		{"position", "Curator"},
		{"title", "Weatherwright"},
		{"badge", "Early reader"},
	} {
		s.assigned(t, ownerHandle, s.defined(t, given.form, given.name, "").ID)
	}
}

func (s restrictionStack) restrict(
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
		http.MethodPut, "/v1/profiles/"+handle+"/restriction", strings.NewReader(string(body)),
	)
	request.Header.Set("Content-Type", "application/json")
	return send(t, s.router, authorized(request, session))
}

func (s restrictionStack) restore(
	t *testing.T,
	session *http.Cookie,
	handle string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/profiles/"+handle+"/restriction", nil,
	), session))
}

func (s restrictionStack) readRestriction(
	t *testing.T,
	session *http.Cookie,
	handle string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+handle+"/restriction", nil,
	), session))
}

func (s restrictionStack) publicProfile(t *testing.T, handle string) restrictedProfile {
	t.Helper()
	response := send(t, s.router, httptest.NewRequest(http.MethodGet, "/v1/profiles/"+handle, nil))
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
	stack := newRestrictionStack(t)
	stack.fillProfile(t)
	createProfileAsset(t, stack.assets, stack.ownerID, "Fen weather", false, asset.DiscoveryListed)

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
	if len(shown.Positions) != 0 || len(shown.Titles) != 0 || len(shown.Badges) != 0 {
		t.Fatalf("restricted profile still carries distinctions: %+v", shown)
	}

	listing := send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/assets?creator="+ownerHandle, nil,
	))
	if listing.Code != http.StatusOK {
		t.Fatalf("listing status = %d, want 200: %s", listing.Code, listing.Body.String())
	}
	var listed profileListingResponse
	if err := json.Unmarshal(listing.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode listing: %v", err)
	}
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].Name != "Fen weather" {
		t.Fatalf("listing = %+v, want the restricted creator's published work", listed)
	}
	if listed.Items[0].Withhold != nil {
		t.Fatalf("restriction withheld an asset: %+v", listed.Items[0].Withhold)
	}
}

func TestTheCompletePublicResponseOfARestrictedProfileHidesNothingInIt(t *testing.T) {
	stack := newRestrictionStack(t)
	stack.fillProfile(t)
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	response := send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+ownerHandle, nil,
	))
	body := response.Body.String()
	for _, hidden := range []string{
		"Wren Ashdown", "weather", "hello@example.com", "example.com/notes",
		"Curator", "Weatherwright", "Early reader", "Impersonating",
	} {
		if strings.Contains(body, hidden) {
			t.Fatalf("the public response still contains %q: %s", hidden, body)
		}
	}
}

func TestOnlyAnAdminRestrictsOrRestoresAProfile(t *testing.T) {
	stack := newRestrictionStack(t)
	outsider := stack.member(t, "outsider@example.com", "ordinary.member")

	for _, role := range []string{"user", "moderator"} {
		setRole(t, stack.pool, "ordinary.member", role)
		refused := stack.restrict(t, outsider, ownerHandle, "Because I say so.")
		if refused.Code != http.StatusForbidden {
			t.Fatalf("%s restrict status = %d, want 403: %s", role, refused.Code, refused.Body.String())
		}
	}
	if refused := stack.restrict(t, stack.authority, ownerHandle, "Because I say so."); refused.Code != http.StatusForbidden {
		t.Fatalf("authority restrict status = %d, want 403: %s", refused.Code, refused.Body.String())
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

func TestARestrictionNeedsAReasonOnlyAnAdminEverReads(t *testing.T) {
	stack := newRestrictionStack(t)

	if blank := stack.restrict(t, stack.admin, ownerHandle, "   "); blank.Code != http.StatusBadRequest {
		t.Fatalf("blank reason status = %d, want 400: %s", blank.Code, blank.Body.String())
	}
	if missing := stack.restrict(t, stack.admin, "nobody.here", "Impersonation."); missing.Code != http.StatusNotFound {
		t.Fatalf("unknown account status = %d, want 404", missing.Code)
	}
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	read := stack.readRestriction(t, stack.admin, ownerHandle)
	if read.Code != http.StatusOK {
		t.Fatalf("admin read status = %d, want 200: %s", read.Code, read.Body.String())
	}
	var found profileRestriction
	if err := json.Unmarshal(read.Body.Bytes(), &found); err != nil {
		t.Fatalf("decode restriction: %v", err)
	}
	if found.Reason != "Impersonating another creator." {
		t.Fatalf("reason = %q", found.Reason)
	}
	if found.RestrictedBy == nil || *found.RestrictedBy != "site.admin" {
		t.Fatalf("restrictedBy = %v, want the acting admin", found.RestrictedBy)
	}

	if hidden := stack.readRestriction(t, stack.owner, ownerHandle); hidden.Code != http.StatusForbidden {
		t.Fatalf("owner reason status = %d, want 403: %s", hidden.Code, hidden.Body.String())
	}
	anonymous := send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/profiles/"+ownerHandle+"/restriction", nil,
	))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("visitor reason status = %d, want 401", anonymous.Code)
	}
}

func TestARestrictedOwnerKeepsItsAccountAndLosesOnlyProfileEdits(t *testing.T) {
	stack := newRestrictionStack(t)
	stack.fillProfile(t)
	published := createProfileAsset(
		t, stack.assets, stack.ownerID, "Fen weather", false, asset.DiscoveryListed,
	)
	before := accountFactsOf(sessionState(t, stack.router, stack.owner))
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")

	blocked := saveProfile(t, stack.router, stack.owner, `{
		"displayName":"Another name","biography":"","contactEmail":"","links":[]
	}`)
	if blocked.Code != http.StatusForbidden {
		t.Fatalf("restricted save status = %d, want 403: %s", blocked.Code, blocked.Body.String())
	}
	if strings.Contains(blocked.Body.String(), "Impersonating") {
		t.Fatalf("the owner was told the private reason: %s", blocked.Body.String())
	}
	replaced := send(t, stack.router, authorized(
		avatarUploadRequest(t, httpTestPNG(t, 200, 200)), stack.owner,
	))
	if replaced.Code != http.StatusForbidden {
		t.Fatalf("restricted avatar status = %d, want 403: %s", replaced.Code, replaced.Body.String())
	}
	removed := send(t, stack.router, authorized(httptest.NewRequest(
		http.MethodDelete, "/v1/account/profile/avatar", nil,
	), stack.owner))
	if removed.Code != http.StatusForbidden {
		t.Fatalf("restricted avatar removal status = %d, want 403", removed.Code)
	}

	after := accountFactsOf(sessionState(t, stack.router, stack.owner))
	if after != before {
		t.Fatalf("restriction changed the account: %+v became %+v", before, after)
	}
	if role := stack.role(t); role != "user" {
		t.Fatalf("restriction changed the account role to %q", role)
	}
	stack.expectAssetUntouched(t, published)
	if createProfileAsset(
		t, stack.assets, stack.ownerID, "Still working", false, asset.DiscoveryListed,
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

func accountFactsOf(state accountState) accountFacts {
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

func (s restrictionStack) role(t *testing.T) string {
	t.Helper()
	var role string
	err := s.pool.QueryRow(context.Background(),
		`select role from users where id = $1`, s.ownerID).Scan(&role)
	if err != nil {
		t.Fatalf("read account role: %v", err)
	}
	return role
}

func (s restrictionStack) expectAssetUntouched(t *testing.T, assetID uuid.UUID) {
	t.Helper()
	var owner uuid.UUID
	var discovery string
	var withheld bool
	err := s.pool.QueryRow(context.Background(), `
		select owner_id, discovery, withheld_at is not null from assets where id = $1
	`, assetID).Scan(&owner, &discovery, &withheld)
	if err != nil {
		t.Fatalf("read asset state: %v", err)
	}
	if owner != s.ownerID || discovery != "listed" || withheld {
		t.Fatalf("restriction changed the asset: owner %s, discovery %q, withheld %v",
			owner, discovery, withheld)
	}
}

func TestRestoringGivesBackEveryRetainedField(t *testing.T) {
	stack := newRestrictionStack(t)
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
	if len(shown.Positions) != 1 || len(shown.Titles) != 1 || len(shown.Badges) != 1 {
		t.Fatalf("restored distinctions = %+v %+v %+v", shown.Positions, shown.Titles, shown.Badges)
	}

	saved := saveProfile(t, stack.router, stack.owner, `{
		"displayName":"Wren A.","biography":"","contactEmail":"","links":[]
	}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("restored save status = %d, want 200: %s", saved.Code, saved.Body.String())
	}
	if again := stack.readRestriction(t, stack.admin, ownerHandle); again.Code != http.StatusNotFound {
		t.Fatalf("restriction survived restoration: %d", again.Code)
	}
}

func TestRestrictionAndRestorationBothLeaveAnAuditRecord(t *testing.T) {
	stack := newRestrictionStack(t)
	stack.restrict(t, stack.admin, ownerHandle, "Impersonating another creator.")
	stack.restore(t, stack.admin, ownerHandle)
	stack.restrict(t, stack.admin, ownerHandle, "Doing it again.")

	rows, err := stack.pool.Query(context.Background(), `
		select actor.username, audit.subject_id, audit.action, audit.recorded_at
		  from profile_restriction_audits audit
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
