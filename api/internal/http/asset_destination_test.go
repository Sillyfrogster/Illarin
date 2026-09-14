package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
)

const updateDestinationsPath = "/v1/account/update-destinations"

func (s destinationStack) updateDestinationRequest(t *testing.T, session *http.Cookie, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := jsonRequest(t, method, path, body)
	if session != nil {
		request = authorized(request, session)
	}
	return send(t, s.router, request)
}

func (s destinationStack) addUpdateDestination(t *testing.T, session *http.Cookie, kind, address string) addedDestination {
	t.Helper()
	response := s.updateDestinationRequest(t, session, http.MethodPost, updateDestinationsPath,
		fmt.Sprintf(`{"name":"Creator updates","kind":%q,"address":%q}`, kind, address))
	if response.Code != http.StatusCreated {
		t.Fatalf("create update destination = %d, want 201: %s", response.Code, response.Body.String())
	}
	var made addedDestination
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Error("destination credentials can be cached")
	}
	return made
}

func TestAssetUpdateDestinationsBelongOnlyToTheirCreator(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	creator := verifiedSignUp(t, stack.router, stack.outbox, "creator@example.com", "asset.creator")
	made := stack.addUpdateDestination(t, creator, "webhook", stack.to.address())
	if !strings.HasPrefix(made.Secret, "whsec_") || made.Destination.State != "unverified" {
		t.Fatal("a new webhook needs its own signing secret and verification")
	}
	for _, session := range []*http.Cookie{creator, stack.authority, stack.editor} {
		response := stack.updateDestinationRequest(t, session, http.MethodGet, updateDestinationsPath, "")
		var list destinationList
		if response.Code != http.StatusOK || json.Unmarshal(response.Body.Bytes(), &list) != nil {
			t.Fatalf("list creator destinations = %d", response.Code)
		}
		want := 0
		if session == creator {
			want = 1
		}
		if len(list.Destinations) != want {
			t.Errorf("listed %d destinations, want %d", len(list.Destinations), want)
		}
		if strings.Contains(response.Body.String(), made.Secret) || strings.Contains(response.Body.String(), stack.to.address()) {
			t.Error("ordinary destination responses contain credentials")
		}
		inspection := stack.updateDestinationRequest(t, session, http.MethodGet, updateDestinationsPath+"/"+made.Destination.ID, "")
		wantStatus := http.StatusNotFound
		if session == creator {
			wantStatus = http.StatusOK
		}
		if inspection.Code != wantStatus {
			t.Errorf("inspect destination = %d, want %d", inspection.Code, wantStatus)
		}
	}
	if len(stack.destinations(t, stack.authority).Destinations) != 0 {
		t.Error("creator destinations appeared in blog configuration")
	}
	denied := stack.add(t, creator, "Blog destination", stack.to.address())
	if denied.Code != http.StatusForbidden {
		t.Errorf("creator gained blog configuration permission: %d", denied.Code)
	}
}

func TestAssetUpdateDestinationVerificationRequiresASignedChallenge(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.addUpdateDestination(t, stack.editor, "webhook", stack.to.address())
	path := updateDestinationsPath + "/" + made.Destination.ID + "/verification"
	failed := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path, "")
	if failed.Code != http.StatusBadRequest {
		t.Fatalf("an endpoint that ignored the challenge = %d, want 400", failed.Code)
	}
	stack.to.answers(echoesTheChallenge)
	verified := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path, "")
	var found destination
	if verified.Code != http.StatusOK || json.Unmarshal(verified.Body.Bytes(), &found) != nil || found.State != "active" {
		t.Fatalf("verified destination = %d, want active", verified.Code)
	}
	requests := stack.to.arrivals()
	last := requests[len(requests)-1]
	var event struct {
		Type   string    `json:"type"`
		SentAt time.Time `json:"sentAt"`
	}
	if err := json.Unmarshal(last.Body, &event); err != nil {
		t.Fatal(err)
	}
	if event.Type != "asset.endpoint.verification.v1" {
		t.Errorf("asset verification event = %q", event.Type)
	}
	if !webhook.Accepts(last.Headers.Get(webhook.SignatureHeader), made.Secret,
		last.Headers.Get(webhook.IDHeader), event.SentAt, last.Body) {
		t.Error("the verification request did not prove its signature")
	}
	if strings.Contains(verified.Body.String(), made.Secret) {
		t.Error("verification revealed the signing secret")
	}
}

func TestAssetUpdateDestinationChangesStayWithTheOwner(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	creator := verifiedSignUp(t, stack.router, stack.outbox, "updates@example.com", "updates.creator")
	made := stack.addUpdateDestination(t, creator, "webhook", stack.to.address())
	path := updateDestinationsPath + "/" + made.Destination.ID
	for _, change := range []struct{ method, suffix, body string }{
		{http.MethodPatch, "", `{"name":"Stolen"}`},
		{http.MethodPost, "/verification", ""},
		{http.MethodDelete, "/verification", ""},
		{http.MethodPost, "/secret", ""},
		{http.MethodDelete, "", ""},
	} {
		for _, stranger := range []*http.Cookie{stack.authority, stack.editor} {
			got := stack.updateDestinationRequest(t, stranger, change.method, path+change.suffix, change.body)
			if got.Code != http.StatusNotFound {
				t.Errorf("another account's %s %s = %d", change.method, change.suffix, got.Code)
			}
		}
	}
	stack.to.answers(echoesTheChallenge)
	if got := stack.updateDestinationRequest(t, creator, http.MethodPost, path+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("could not verify")
	}
	rotated := stack.updateDestinationRequest(t, creator, http.MethodPost, path+"/secret", "")
	var next addedDestination
	if rotated.Code != http.StatusOK || json.Unmarshal(rotated.Body.Bytes(), &next) != nil || next.Secret == made.Secret || next.Secret == "" {
		t.Fatalf("rotation = %d, want a new secret", rotated.Code)
	}
	if got := stack.updateDestinationRequest(t, creator, http.MethodPost, path+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("could not verify after rotation")
	}
	requests := stack.to.arrivals()
	last := requests[len(requests)-1]
	var event struct {
		SentAt time.Time `json:"sentAt"`
	}
	if err := json.Unmarshal(last.Body, &event); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{made.Secret, next.Secret} {
		if !webhook.Accepts(last.Headers.Get(webhook.SignatureHeader), secret, last.Headers.Get(webhook.IDHeader), event.SentAt, last.Body) {
			t.Error("rotation lost the overlapping signature")
		}
	}
	for _, change := range []struct{ method, suffix, body, state string }{
		{http.MethodDelete, "/verification", "", "disabled"},
		{http.MethodPatch, "", `{"name":"Quiet receiver"}`, "disabled"},
		{http.MethodPost, "/verification", "", "active"},
		{http.MethodPatch, "", fmt.Sprintf(`{"address":%q}`, stack.to.address()+"/new"), "unverified"},
	} {
		got := stack.updateDestinationRequest(t, creator, change.method, path+change.suffix, change.body)
		var updated destination
		if got.Code != http.StatusOK || json.Unmarshal(got.Body.Bytes(), &updated) != nil || updated.State != change.state {
			t.Fatalf("%s %s = %d, want %s", change.method, change.suffix, got.Code, change.state)
		}
		if strings.Contains(got.Body.String(), made.Secret) || strings.Contains(got.Body.String(), next.Secret) {
			t.Error("a change revealed a secret")
		}
	}
	removed := stack.updateDestinationRequest(t, creator, http.MethodDelete, path, "")
	if removed.Code != http.StatusNoContent {
		t.Errorf("remove = %d", removed.Code)
	}
	if got := stack.updateDestinationRequest(t, creator, http.MethodGet, path, ""); got.Code != http.StatusNotFound {
		t.Error("removed destination still exists")
	}
}

func TestAssetUpdateDiscordDestinationsVerifyCapabilitiesWithoutMentionControls(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.addUpdateDestination(t, stack.editor, "discord", discordCapability())
	if made.Secret != "" || made.Destination.State != "active" || made.Destination.Channel == nil || made.Destination.Channel.ChannelID != discordChannelID {
		t.Fatal("Discord creation must verify the channel without exposing a secret")
	}
	path := updateDestinationsPath + "/" + made.Destination.ID
	for _, extra := range []string{`"roleId":"333333333333333333"`, `"roles":[]`, `"allowed_mentions":{"parse":["everyone"]}`, `"content":"@everyone"`} {
		refused := stack.updateDestinationRequest(t, stack.editor, http.MethodPatch, path, `{"name":"@everyone",`+extra+`}`)
		if refused.Code != http.StatusBadRequest {
			t.Errorf("a mention control was accepted: %d", refused.Code)
		}
	}
	stack.discord.answersReadWith(func() (int, string) { return http.StatusUnauthorized, discordToken })
	failed := stack.updateDestinationRequest(t, stack.editor, http.MethodPatch, path, fmt.Sprintf(`{"address":%q}`, discordCapability()))
	if failed.Code != http.StatusBadRequest || strings.Contains(failed.Body.String(), discordToken) {
		t.Error("failed Discord replacement must refuse safely")
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodDelete, path+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("disable Discord failed")
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/verification", ""); got.Code != http.StatusBadRequest {
		t.Fatal("invalid capability reactivated Discord")
	}
	stack.discord.answersReadWith(nil)
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("reverify Discord failed")
	}
	inspection := stack.updateDestinationRequest(t, stack.editor, http.MethodGet, path, "")
	if strings.Contains(inspection.Body.String(), discordToken) || strings.Contains(inspection.Body.String(), "roleId") {
		t.Error("Discord configuration exposes credentials or role controls")
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/secret", ""); got.Code != http.StatusBadRequest {
		t.Fatal("Discord accepted generic secret rotation")
	}
	if len(stack.discord.announcements()) != 0 {
		t.Error("configuration sent a Discord message")
	}
}

func TestAssetUpdateDestinationDefaultsRememberOnlyEligibleOwnedDestinations(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	first := startCharacter(t, stack.router, stack.editor)
	second := startCharacter(t, stack.router, stack.editor)
	webhook := stack.addUpdateDestination(t, stack.editor, "webhook", stack.to.address())
	channel := stack.addUpdateDestination(t, stack.editor, "discord", discordCapability())
	foreign := stack.addUpdateDestination(t, stack.authority, "discord", discordCapability())
	path := "/v1/assets/" + first.ID + "/update-destinations"
	for _, refusedID := range []string{webhook.Destination.ID, foreign.Destination.ID} {
		got := stack.updateDestinationRequest(t, stack.editor, http.MethodPut, path, fmt.Sprintf(`{"destinationIds":[%q]}`, refusedID))
		if got.Code != http.StatusBadRequest {
			t.Errorf("ineligible selection = %d", got.Code)
		}
	}
	chosen := stack.updateDestinationRequest(t, stack.editor, http.MethodPut, path, fmt.Sprintf(`{"destinationIds":[%q]}`, channel.Destination.ID))
	if chosen.Code != http.StatusNoContent {
		t.Fatalf("save destination defaults = %d", chosen.Code)
	}
	for _, assetID := range []string{first.ID, second.ID} {
		got := stack.updateDestinationRequest(t, stack.editor, http.MethodGet, "/v1/assets/"+assetID+"/update-destinations", "")
		var choices destinationChoiceList
		if got.Code != http.StatusOK || json.Unmarshal(got.Body.Bytes(), &choices) != nil {
			t.Fatalf("read choices = %d", got.Code)
		}
		if len(choices.Destinations) != 1 || choices.Destinations[0].ID != channel.Destination.ID || choices.Destinations[0].ByDefault != (assetID == first.ID) {
			t.Error("choices include an ineligible destination or defaults leaked between assets")
		}
		if strings.Contains(got.Body.String(), "address") || strings.Contains(got.Body.String(), "secret") {
			t.Error("selection exposed configuration")
		}
	}
	for _, stranger := range []*http.Cookie{stack.authority, nil} {
		got := stack.updateDestinationRequest(t, stranger, http.MethodGet, path, "")
		if got.Code != http.StatusNotFound && got.Code != http.StatusUnauthorized {
			t.Errorf("stranger read choices: %d", got.Code)
		}
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodDelete, updateDestinationsPath+"/"+channel.Destination.ID+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("disable failed")
	}
	disabled := stack.updateDestinationRequest(t, stack.editor, http.MethodGet, path, "")
	var choices destinationChoiceList
	if json.Unmarshal(disabled.Body.Bytes(), &choices) != nil || len(choices.Destinations) != 0 {
		t.Error("disabled destination remains eligible")
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodPut, path, `{"destinationIds":[]}`); got.Code != http.StatusNoContent {
		t.Fatal("quiet defaults failed")
	}
	if got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, updateDestinationsPath+"/"+channel.Destination.ID+"/verification", ""); got.Code != http.StatusOK {
		t.Fatal("reverification failed")
	}
	quiet := stack.updateDestinationRequest(t, stack.editor, http.MethodGet, path, "")
	if json.Unmarshal(quiet.Body.Bytes(), &choices) != nil || len(choices.Destinations) != 1 || choices.Destinations[0].ByDefault {
		t.Error("quiet publication defaults were not remembered")
	}
}

func TestAssetUpdateDestinationsRefuseUnsafeAddressesAndChangedDNS(t *testing.T) {
	t.Parallel()
	private := false
	stack := newDestinationStackThrough(t, func(string) ([]netip.Addr, error) {
		if private {
			return []netip.Addr{netip.MustParseAddr("93.184.216.34"), netip.MustParseAddr("10.0.0.1")}, nil
		}
		return []netip.Addr{netip.MustParseAddr("93.184.216.34")}, nil
	})
	for _, address := range []string{
		"http://hooks.example.com/update", "https://hooks.example.com:8443/update",
		"https://user:password@hooks.example.com/update", "https://hooks.example.com/update#secret",
		"https://127.0.0.1/update", "https://[::1]/update", "https://169.254.169.254/update",
	} {
		got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, updateDestinationsPath,
			fmt.Sprintf(`{"kind":"webhook","name":"Unsafe receiver","address":%q}`, address))
		if got.Code != http.StatusBadRequest {
			t.Errorf("unsafe address was accepted: %d", got.Code)
		}
		if strings.Contains(got.Body.String(), address) {
			t.Error("an error echoed the endpoint address")
		}
	}
	made := stack.addUpdateDestination(t, stack.editor, "webhook", stack.to.address())
	stack.to.answers(echoesTheChallenge)
	private = true
	got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, updateDestinationsPath+"/"+made.Destination.ID+"/verification", "")
	if got.Code != http.StatusBadRequest || len(stack.to.arrivals()) != 0 {
		t.Error("verification reached a host with a private DNS result")
	}
	for _, address := range []string{discordCapability() + "?thread_id=123", strings.Replace(discordCapability(), "discord.com", "example.com", 1)} {
		got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, updateDestinationsPath,
			fmt.Sprintf(`{"kind":"discord","name":"Invalid channel","address":%q}`, address))
		if got.Code != http.StatusBadRequest {
			t.Error("an arbitrary Discord target was accepted")
		}
	}
}

func TestAssetUpdateVerificationCannotReactivateADisabledDestination(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.addUpdateDestination(t, stack.editor, "webhook", stack.to.address())
	path := updateDestinationsPath + "/" + made.Destination.ID
	started, release := make(chan struct{}), make(chan struct{})
	stack.to.answers(func(one arrived) (int, string) { close(started); <-release; return echoesTheChallenge(one) })
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		result <- stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/verification", "")
	}()
	<-started
	disabled := stack.updateDestinationRequest(t, stack.editor, http.MethodDelete, path+"/verification", "")
	close(release)
	if disabled.Code != http.StatusOK {
		t.Fatal("disable failed")
	}
	if got := <-result; got.Code != http.StatusConflict {
		t.Errorf("stale verification = %d, want 409", got.Code)
	}
	got := stack.updateDestinationRequest(t, stack.editor, http.MethodGet, path, "")
	var found destination
	if json.Unmarshal(got.Body.Bytes(), &found) != nil || found.State != "disabled" {
		t.Error("verification undid the owner's disabling")
	}
}

func TestAssetUpdateDestinationRotationExpiresTheOldSignature(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.addUpdateDestination(t, stack.editor, "webhook", stack.to.address())
	path := updateDestinationsPath + "/" + made.Destination.ID
	got := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/secret", "")
	var rotated addedDestination
	if got.Code != http.StatusOK || json.Unmarshal(got.Body.Bytes(), &rotated) != nil {
		t.Fatal("rotation failed")
	}
	if again := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/secret", ""); again.Code != http.StatusBadRequest {
		t.Error("a second rotation cut short the promised overlap")
	}
	if n, err := stack.handlers.updateDestinations.ForgetOldSecrets(t.Context(), time.Now().Add(25*time.Hour)); err != nil || n != 1 {
		t.Fatalf("expired secret cleanup = %d, %v", n, err)
	}
	stack.to.answers(echoesTheChallenge)
	if verified := stack.updateDestinationRequest(t, stack.editor, http.MethodPost, path+"/verification", ""); verified.Code != http.StatusOK {
		t.Fatal("verification failed after expiry")
	}
	last := stack.to.arrivals()[0]
	var event struct {
		SentAt time.Time `json:"sentAt"`
	}
	if err := json.Unmarshal(last.Body, &event); err != nil {
		t.Fatal(err)
	}
	if webhook.Accepts(last.Headers.Get(webhook.SignatureHeader), made.Secret, last.Headers.Get(webhook.IDHeader), event.SentAt, last.Body) {
		t.Error("an expired secret still signs requests")
	}
	if !webhook.Accepts(last.Headers.Get(webhook.SignatureHeader), rotated.Secret, last.Headers.Get(webhook.IDHeader), event.SentAt, last.Body) {
		t.Error("cleanup removed the current signature")
	}
}

func TestAssetUpdateDestinationCredentialsAreEncryptedAtRest(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	for _, target := range []struct{ kind, address string }{
		{"webhook", stack.to.address()},
		{"discord", discordCapability()},
	} {
		made := stack.addUpdateDestination(t, stack.editor, target.kind, target.address)
		var address, secret []byte
		err := stack.pool.QueryRow(t.Context(), `
			select address, signing_secret from asset_update_destinations where id = $1
		`, made.Destination.ID).Scan(&address, &secret)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(address), target.address) {
			t.Fatal("stored address contains the plaintext capability")
		}
		opened, err := testSealingKey().Open(address)
		if err != nil || string(opened) != target.address {
			t.Fatal("stored address cannot be opened with the application key")
		}
		if target.kind == "discord" {
			if secret != nil {
				t.Error("Discord stored an unrelated signing secret")
			}
			continue
		}
		if strings.Contains(string(secret), made.Secret) {
			t.Fatal("stored signing secret contains plaintext")
		}
		opened, err = testSealingKey().Open(secret)
		if err != nil || string(opened) != made.Secret {
			t.Fatal("stored signing secret cannot be opened with the application key")
		}
	}
}

func TestAssetUpdateDestinationDefaultsDoNotAnnounceFirstPublication(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	started := startCharacter(t, stack.router, stack.editor)
	writeCharacterFloor(t, stack.router, stack.editor, started)
	made := stack.addUpdateDestination(t, stack.editor, "discord", discordCapability())
	path := "/v1/assets/" + started.ID + "/update-destinations"
	chosen := stack.updateDestinationRequest(t, stack.editor, http.MethodPut, path,
		fmt.Sprintf(`{"destinationIds":[%q]}`, made.Destination.ID))
	if chosen.Code != http.StatusNoContent {
		t.Fatal("could not save defaults before first publication")
	}
	if published := publishAsset(t, stack.router, stack.editor, started.ID); published.Code != http.StatusOK {
		t.Fatal("could not publish the asset")
	}
	if len(stack.discord.announcements()) != 0 || len(stack.to.arrivals()) != 0 {
		t.Error("first asset publication sent an announcement")
	}
	var deliveries int
	if err := stack.pool.QueryRow(t.Context(), `select count(*) from publication_deliveries`).Scan(&deliveries); err != nil {
		t.Fatal(err)
	}
	if deliveries != 0 {
		t.Error("asset defaults queued a blog announcement")
	}
}
