package page_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
)

func TestPublishRefusesAnIncompleteDraftAndNamesEveryMissingItem(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	response := apitest.PublishWork(t, r, session, started.ID)

	if response.Code != http.StatusConflict {
		t.Fatalf("publish an empty draft status = %d, want 409: %s",
			response.Code, response.Body.String())
	}
	var refusal apitest.PublishRefusal
	if err := json.Unmarshal(response.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("decode the refusal: %v", err)
	}
	if refusal.Error == "" {
		t.Error("the refusal says nothing")
	}
	for _, id := range []string{"name", "adult_content", "description", "greetings"} {
		item := apitest.ItemNamed(t, refusal.Readiness, id)
		if item.Met {
			t.Errorf("%s reads as met on an empty draft", id)
		}
		if item.Label == "" || item.Detail == "" {
			t.Errorf("%s = %+v, want wording a creator can act on", id, item)
		}
	}
	if apitest.ItemNamed(t, refusal.Readiness, "description").BlockID == nil {
		t.Error("a missing description links to no block")
	}
	if apitest.ItemNamed(t, refusal.Readiness, "name").BlockID != nil {
		t.Error("the name links to a block, and it is a header field")
	}

	page := apitest.FetchStartedWork(t, r, session, started.ID)
	if page.Lifecycle != "draft" {
		t.Errorf("a refused draft is now %q", page.Lifecycle)
	}
}

func TestABlurbIsNeverRequiredAndPublishingIsOneWay(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)

	published := apitest.PublishWork(t, r, session, started.ID)
	if published.Code != http.StatusOK {
		t.Fatalf("publish a complete draft status = %d, want 200: %s",
			published.Code, published.Body.String())
	}
	var page apitest.StartedWork
	if err := json.Unmarshal(published.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode the published page: %v", err)
	}
	if page.Lifecycle != "published" {
		t.Errorf("lifecycle = %q, want published", page.Lifecycle)
	}
	if page.Blurb != "" {
		t.Errorf("blurb = %q, and no blurb was written", page.Blurb)
	}
	if len(page.Readiness) != 0 {
		t.Errorf("a published work still carries a readiness list: %+v", page.Readiness)
	}

	stranger := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil))
	if stranger.Code != http.StatusOK {
		t.Errorf("a reader got %d for a published work, want 200", stranger.Code)
	}

	again := apitest.PublishWork(t, r, session, started.ID)
	if again.Code != http.StatusConflict {
		t.Errorf("publishing twice status = %d, want 409: %s", again.Code, again.Body.String())
	}
}

func TestTheFloorReadsElementContentRatherThanTheBlockItSitsIn(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	if got := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"Ilse","blurb":"","isNsfw":true}`); got.Code != http.StatusNoContent {
		t.Fatalf("save details status = %d: %s", got.Code, got.Body.String())
	}

	refused := apitest.PublishWork(t, r, session, started.ID)
	if refused.Code != http.StatusConflict {
		t.Fatalf("a work with empty required blocks published: %d", refused.Code)
	}

	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She keeps the books."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save description status = %d: %s", got.Code, got.Body.String())
	}
	messagesBlock := apitest.BlockNamed(t, started.Blocks, "messages")
	messages := apitest.EditableBlock(messagesBlock)
	messages.Elements[0].Content = json.RawMessage(`{"texts":[{"text":""}]}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages); got.Code != http.StatusOK {
		t.Fatalf("save an empty greeting status = %d: %s", got.Code, got.Body.String())
	}

	stillRefused := apitest.PublishWork(t, r, session, started.ID)
	if stillRefused.Code != http.StatusConflict {
		t.Fatalf("a greeting with no text published the draft: %d", stillRefused.Code)
	}
	var refusal apitest.PublishRefusal
	if err := json.Unmarshal(stillRefused.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("decode the refusal: %v", err)
	}
	if !apitest.ItemNamed(t, refusal.Readiness, "description").Met {
		t.Error("a written description does not meet the floor")
	}
	if apitest.ItemNamed(t, refusal.Readiness, "greetings").Met {
		t.Error("an empty greeting meets the floor")
	}

	messages.Elements[0].Content = json.RawMessage(`{"texts":[{"text":"Come in."}]}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages); got.Code != http.StatusOK {
		t.Fatalf("save a written greeting status = %d: %s", got.Code, got.Body.String())
	}
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
}

func TestAReadinessListStandsOnADraftForItsOwnerAlone(t *testing.T) {
	t.Parallel()
	setup, r, session, _ := harness.NewVerifiedRoutersWithService(t, 1<<20, api.DefaultDeadlines())
	started := apitest.StartCharacter(t, r, session)

	if len(started.Readiness) != 4 {
		t.Fatalf("readiness on a new draft = %+v, want four items", started.Readiness)
	}
	if got := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"Ilse","blurb":"","isNsfw":null}`); got.Code != http.StatusNoContent {
		t.Fatalf("save a name with no answer status = %d: %s", got.Code, got.Body.String())
	}
	named := apitest.FetchStartedWork(t, r, session, started.ID)
	if !apitest.ItemNamed(t, named.Readiness, "name").Met {
		t.Error("a named draft still reads as unnamed")
	}
	if apitest.ItemNamed(t, named.Readiness, "adult_content").Met {
		t.Error("an unanswered adult content question reads as answered")
	}
	if named.IsNSFW != nil {
		t.Errorf("the adult content answer = %v, want unanswered", *named.IsNSFW)
	}

	other := apitest.SignUp(t, setup, "other@example.com", "other.creator")
	stranger := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil), other))
	if stranger.Code != http.StatusNotFound {
		t.Errorf("another account read the draft: %d", stranger.Code)
	}
}

func TestADraftHasNoDownloadNoSendAndNoVisibilityToSet(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	for _, path := range []string{
		"/download/" + started.ID,
		"/download/" + started.ID + "/raw",
	} {
		response := apitest.Send(t, r, apitest.Authorized(
			httptest.NewRequest(http.MethodGet, path, nil), session))
		if response.Code != http.StatusNotFound {
			t.Errorf("GET %s status = %d, want 404 while the work is a draft",
				path, response.Code)
		}
	}

	visibility := httptest.NewRequest(http.MethodPut,
		"/v1/works/"+started.ID+"/visibility", strings.NewReader(`{"visibility":"unlisted"}`))
	visibility.Header.Set("Content-Type", "application/json")
	if got := apitest.Send(t, r, apitest.Authorized(visibility, session)); got.Code != http.StatusConflict {
		t.Errorf("set visibility on a draft status = %d, want 409: %s", got.Code, got.Body.String())
	}
}

func TestADraftsImagesAreServedOnlyAgainstTheSignatureItsPageCarries(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	added := apitest.Send(t, r, apitest.Authorized(apitest.MediaUploadRequest(
		t, started.ID, "avatar", apitest.PNG(t, 600, 400),
	), session))
	if added.Code != http.StatusCreated {
		t.Fatalf("add draft media status = %d, want 201: %s", added.Code, added.Body.String())
	}

	strangerList := apitest.Send(t, r, httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/media", nil))
	if strangerList.Code != http.StatusNotFound {
		t.Errorf("a stranger listed a draft's images: %d", strangerList.Code)
	}

	page := apitest.FetchStartedWork(t, r, session, started.ID)
	if len(page.Media) != 1 {
		t.Fatalf("draft media = %+v, want the image just added", page.Media)
	}
	if page.Preview != nil {
		t.Errorf("a draft carries an unfurl preview: %q", *page.Preview)
	}
	signed := page.Media[0].DetailURL
	path, query, found := strings.Cut(signed, "?")
	if !found || !strings.Contains(query, "signature=") {
		t.Fatalf("draft image URL = %q, want a signature", signed)
	}

	served := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, signed, nil))
	if served.Code != http.StatusOK {
		t.Fatalf("signed draft image status = %d, want 200: %s", served.Code, served.Body.String())
	}
	if got := served.Header().Get("Cache-Control"); got != "private, no-store" {
		t.Errorf("Cache-Control = %q, want a draft image kept out of caches", got)
	}

	unsigned := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, path, nil), session))
	if unsigned.Code != http.StatusNotFound {
		t.Errorf("unsigned draft image status = %d, want 404 even for the owner", unsigned.Code)
	}

	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	published := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, path, nil))
	if published.Code != http.StatusOK {
		t.Errorf("the same image URL after publishing = %d, want 200", published.Code)
	}
	if got := published.Header().Get("Cache-Control"); got != "public, max-age=31536000, immutable" {
		t.Errorf("published Cache-Control = %q", got)
	}
}

func TestDraftsStandInTheOwnersOwnListingAndNowhereElse(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	owner := readProfileListing(t, r, "/v1/works?creator=verified.creator", session)
	if len(owner.Items) != 1 || owner.Items[0].OwnerState == nil ||
		*owner.Items[0].OwnerState != "draft" {
		t.Fatalf("owner listing = %+v, want the draft marked", owner.Items)
	}
	if owner.Items[0].Name != "" {
		t.Errorf("an unnamed draft carries the name %q", owner.Items[0].Name)
	}
	if owner.Items[0].IsNsfw != nil {
		t.Errorf("an unanswered draft reads as %v", *owner.Items[0].IsNsfw)
	}

	stranger := readProfileListing(t, r, "/v1/works?creator=verified.creator", nil)
	if len(stranger.Items) != 0 {
		t.Errorf("a stranger sees %+v on the profile", stranger.Items)
	}

	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	after := readProfileListing(t, r, "/v1/works?creator=verified.creator", session)
	if len(after.Items) != 1 || after.Items[0].OwnerState != nil {
		t.Errorf("published listing = %+v, want an unmarked card", after.Items)
	}
}

func TestDeletingADraftTakesTheSameRecoveryWindow(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)

	deleted := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodDelete, "/v1/works/"+started.ID, nil), session))
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete a draft status = %d, want 204: %s", deleted.Code, deleted.Body.String())
	}

	listed := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/profiles/verified.creator/deleted", nil), session))
	if listed.Code != http.StatusOK {
		t.Fatalf("deleted listing status = %d, want 200: %s", listed.Code, listed.Body.String())
	}
	var recovery struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &recovery); err != nil {
		t.Fatalf("decode deleted listing: %v", err)
	}
	if len(recovery.Items) != 1 || recovery.Items[0].ID != started.ID {
		t.Fatalf("deleted listing = %+v, want the deleted draft", recovery.Items)
	}

	restored := apitest.Send(t, r, apitest.Authorized(
		httptest.NewRequest(http.MethodPost, "/v1/works/"+started.ID+"/restore", nil), session))
	if restored.Code != http.StatusNoContent {
		t.Fatalf("restore a draft status = %d, want 204: %s", restored.Code, restored.Body.String())
	}
	if page := apitest.FetchStartedWork(t, r, session, started.ID); page.Lifecycle != "draft" {
		t.Errorf("a restored draft is now %q", page.Lifecycle)
	}
}

func TestAPublishedWorkKeepsItsAdultContentAnswer(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}

	unanswered := apitest.SaveDetails(t, r, session, started.ID, `{"name":"Ilse","blurb":"","isNsfw":null}`)
	if unanswered.Code != http.StatusBadRequest {
		t.Errorf("unanswering a published work status = %d, want 400: %s",
			unanswered.Code, unanswered.Body.String())
	}

	long := apitest.SaveDetails(t, r, session, started.ID,
		`{"name":"`+strings.Repeat("a", 201)+`","blurb":"","isNsfw":false}`)
	if long.Code != http.StatusBadRequest {
		t.Errorf("an overlong name status = %d, want 400: %s", long.Code, long.Body.String())
	}
}

func TestAPublishedPageBelowTheFloorMarksTheShortfallForItsOwner(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}

	page := apitest.FetchStartedWork(t, r, session, started.ID)
	if len(page.Readiness) != 0 {
		t.Errorf("a published page that meets the floor carries %d items", len(page.Readiness))
	}

	messagesBlock := apitest.BlockNamed(t, page.Blocks, "messages")
	messages := apitest.EditableBlock(messagesBlock)
	messages.Elements[0].Content = json.RawMessage(`{"texts":[]}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages); got.Code != http.StatusOK {
		t.Fatalf("empty the greetings status = %d, want 200: %s", got.Code, got.Body.String())
	}

	short := apitest.FetchStartedWork(t, r, session, started.ID)
	if short.Lifecycle != "published" {
		t.Errorf("the page is now %q, and a shortfall never unpublishes", short.Lifecycle)
	}
	greetings := apitest.ItemNamed(t, short.Readiness, "greetings")
	if greetings.Met {
		t.Error("the greetings requirement reads as met with no greeting")
	}
	if greetings.BlockID == nil {
		t.Error("the shortfall links to no block for the owner to go to")
	}
}

func TestAPublishedPageBelowTheFloorMarksNothingForAVisitor(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	page := apitest.FetchStartedWork(t, r, session, started.ID)
	messagesBlock := apitest.BlockNamed(t, page.Blocks, "messages")
	messages := apitest.EditableBlock(messagesBlock)
	messages.Elements[0].Content = json.RawMessage(`{"texts":[]}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, messagesBlock.ID, messages); got.Code != http.StatusOK {
		t.Fatalf("empty the greetings status = %d, want 200: %s", got.Code, got.Body.String())
	}

	response := apitest.Send(t, r, httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID, nil))
	if response.Code != http.StatusOK {
		t.Fatalf("read the page as a visitor status = %d, want 200", response.Code)
	}
	var visitor apitest.StartedWork
	if err := json.Unmarshal(response.Body.Bytes(), &visitor); err != nil {
		t.Fatalf("decode the visitor's page: %v", err)
	}
	if len(visitor.Readiness) != 0 {
		t.Errorf("a visitor reads %d readiness items, want none", len(visitor.Readiness))
	}
}

func TestAVersionPublishesOnceAndTheSameCandidateIsRefusedAfterwards(t *testing.T) {
	t.Parallel()
	r, session := harness.NewVerifiedRouter(t)
	started := apitest.StartCharacter(t, r, session)
	apitest.WriteCharacterFloor(t, r, session, started)
	if got := apitest.PublishWork(t, r, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d, want 200: %s", got.Code, got.Body.String())
	}
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(`{"text":"She has moved to the east shelf."}`)
	if got := apitest.SaveBlock(t, r, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description status = %d, want 200: %s", got.Code, got.Body.String())
	}

	response := apitest.PublishWorkVersion(t, r, session, started.ID,
		`{"summary":"Moved her to the east shelf","versionLabel":"v2"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("publish an update status = %d, want 200: %s", response.Code, response.Body.String())
	}
	var recorded struct {
		Number         int    `json:"number"`
		Summary        string `json:"summary"`
		VersionLabel   string `json:"versionLabel"`
		ContentChanged bool   `json:"contentChanged"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &recorded); err != nil {
		t.Fatalf("decode the recorded update: %v", err)
	}
	if recorded.Number != 2 || recorded.VersionLabel != "v2" || !recorded.ContentChanged {
		t.Fatalf("recorded update = %+v", recorded)
	}
	if response.Header().Get("X-Drafted-Changes-Version") == "" {
		t.Error("the update named no committed drafted-changes version")
	}

	again := apitest.PublishWorkVersion(t, r, session, started.ID, `{"summary":"Nothing new"}`)
	if again.Code != http.StatusConflict {
		t.Fatalf("republishing status = %d, want 409: %s", again.Code, again.Body.String())
	}
	var refusal struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(again.Body.Bytes(), &refusal); err != nil {
		t.Fatalf("decode the refusal: %v", err)
	}
	if refusal.Code != "no_changes" {
		t.Fatalf("refusal code = %q, want no_changes: %s", refusal.Code, again.Body.String())
	}
}
