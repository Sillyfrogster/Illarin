package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
)

type workAnnouncement struct {
	ID             string           `json:"id"`
	AnnouncementID string           `json:"announcementId"`
	VersionID      string           `json:"versionId"`
	VersionNumber  int              `json:"versionNumber"`
	Integration    string           `json:"integration"`
	Type           string           `json:"type"`
	Removed        bool             `json:"removed"`
	State          string           `json:"state"`
	SettledReason  string           `json:"settledReason"`
	MessageID      string           `json:"messageId"`
	Run            int              `json:"run"`
	Tries          int              `json:"tries"`
	OccurredAt     time.Time        `json:"occurredAt"`
	DueAt          time.Time        `json:"dueAt"`
	SettledAt      *time.Time       `json:"settledAt"`
	Last           *announcementTry `json:"last"`
}

type workAnnouncementList struct {
	Announcements []workAnnouncement `json:"announcements"`
}

type workUpdateEvent struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`
	OccurredAt time.Time `json:"occurredAt"`
	Work       struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"work"`
	Update struct {
		ID             string `json:"id"`
		Number         int    `json:"number"`
		VersionLabel   string `json:"versionLabel"`
		Summary        string `json:"summary"`
		ContentChanged bool   `json:"contentChanged"`
		HistoryURL     string `json:"historyUrl"`
	} `json:"update"`
}

func (s integrationStack) creatorWebhook(t *testing.T, session *http.Cookie) addedIntegration {
	t.Helper()
	made := s.addUpdateDestination(t, session, "webhook", s.to.address())
	s.to.answers(echoesTheChallenge)
	verified := s.integrationRequest(t, session, http.MethodPost,
		integrationsPath+"/"+made.Integration.ID+"/verification", "")
	if verified.Code != http.StatusOK {
		t.Fatalf("verify creator webhook = %d: %s", verified.Code, verified.Body.String())
	}
	s.to.answers(nil)
	s.to.forget()
	return made
}

func (s integrationStack) publishedCharacter(t *testing.T, session *http.Cookie) apitest.StartedWork {
	t.Helper()
	started := apitest.StartCharacter(t, s.router, session)
	apitest.WriteCharacterFloor(t, s.router, session, started)
	if got := apitest.PublishWork(t, s.router, session, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	return started
}

func (s integrationStack) describe(t *testing.T, session *http.Cookie, started apitest.StartedWork, text string) {
	t.Helper()
	coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
	core := apitest.EditableBlock(coreBlock)
	core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, text))
	if got := apitest.SaveBlock(t, s.router, session, started.ID, coreBlock.ID, core); got.Code != http.StatusOK {
		t.Fatalf("save the description = %d: %s", got.Code, got.Body.String())
	}
}

func (s integrationStack) announced(t *testing.T, session *http.Cookie, workID, body string) {
	t.Helper()
	response := apitest.PublishWorkVersion(t, s.router, session, workID, body)
	if response.Code != http.StatusOK {
		t.Fatalf("publish an update = %d: %s", response.Code, response.Body.String())
	}
}

func (s integrationStack) announcements(t *testing.T, session *http.Cookie, workID string) []workAnnouncement {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+workID+"/announcement-attempts", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read announcements = %d: %s", response.Code, response.Body.String())
	}
	var listed workAnnouncementList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode announcements: %v", err)
	}
	return listed.Announcements
}

func (s integrationStack) sendAnnouncementsAt(t *testing.T, at time.Time) int {
	t.Helper()
	made, err := s.handlers.Integrations.SendDueAnnouncements(t.Context(), at)
	if err != nil {
		t.Fatalf("send due announcements: %v", err)
	}
	return made
}

func (s integrationStack) onlyAnnouncement(t *testing.T, session *http.Cookie, workID string) workAnnouncement {
	t.Helper()
	listed := s.announcements(t, session, workID)
	if len(listed) != 1 {
		t.Fatalf("the work shows %d announcements, want 1: %+v", len(listed), listed)
	}
	return listed[0]
}

func TestAPublishedUpdateAnnouncesToItsChosenDestinationsOutsideTheRequest(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	channel := stack.addUpdateDestination(t, stack.editor, "discord", discordCapability())
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "She has moved to the east shelf.")

	stack.announced(t, stack.editor, started.ID, fmt.Sprintf(
		`{"summary":"Moved her @everyone <@&%s> to the east shelf","notes":"Private reasoning stays here.",`+
			`"versionLabel":"v2","integrationIds":[%q,%q]}`,
		discordWebhookID, hook.Integration.ID, channel.Integration.ID))
	if len(stack.to.arrivals()) != 0 || len(stack.discord.announcements()) != 0 {
		t.Fatal("publication sent inside its own request")
	}
	queued := stack.announcements(t, stack.editor, started.ID)
	if len(queued) != 2 {
		t.Fatalf("queued %d announcements, want 2", len(queued))
	}
	for _, one := range queued {
		if one.State != "pending" || one.Tries != 0 || one.VersionNumber != 2 {
			t.Errorf("queued announcement = %+v", one)
		}
	}

	if made := stack.sendAnnouncementsAt(t, time.Now()); made != 2 {
		t.Fatalf("sent %d announcements, want 2", made)
	}
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the webhook received %d requests, want 1", len(arrivals))
	}
	var event workUpdateEvent
	if err := json.Unmarshal(arrivals[0].Body, &event); err != nil {
		t.Fatalf("decode the event: %v", err)
	}
	if event.Type != "work.update.published.v1" || event.Work.ID != started.ID ||
		event.Work.Type != "character" || event.Work.Name != "Ilse of the west shelf" ||
		event.Update.Number != 2 || event.Update.VersionLabel != "v2" ||
		!event.Update.ContentChanged || event.OccurredAt.IsZero() ||
		!strings.HasPrefix(event.Update.Summary, "Moved her") {
		t.Errorf("event = %+v", event)
	}
	if event.Work.URL != "http://localhost:3000/a/"+started.ID ||
		event.Update.HistoryURL != "http://localhost:3000/a/"+started.ID+"/history#version-2" {
		t.Errorf("links = %q and %q", event.Work.URL, event.Update.HistoryURL)
	}
	body := string(arrivals[0].Body)
	for _, private := range []string{"Private reasoning", "east shelf.", "notes", "diff"} {
		if strings.Contains(body, private) {
			t.Errorf("the event carries %q", private)
		}
	}
	webhookID := arrivals[0].Headers.Get(dispatch.IDHeader)
	stamp := stampOf(t, arrivals[0])
	if !dispatch.Accepts(arrivals[0].Headers.Get(dispatch.SignatureHeader), hook.Secret, webhookID, stamp, arrivals[0].Body) {
		t.Error("the announcement was not signed with the creator's secret")
	}

	posted := stack.discord.announcements()
	if len(posted) != 1 {
		t.Fatalf("Discord received %d requests, want 1", len(posted))
	}
	var message struct {
		Content  string `json:"content"`
		Mentions struct {
			Parse []string `json:"parse"`
			Roles []string `json:"roles"`
			Users []string `json:"users"`
		} `json:"allowed_mentions"`
		Embeds []struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Fields      []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"fields"`
			Footer struct {
				Text string `json:"text"`
			} `json:"footer"`
		} `json:"embeds"`
	}
	if err := json.Unmarshal(posted[0].Body, &message); err != nil {
		t.Fatalf("decode the Discord message: %v", err)
	}
	if message.Content != "" || len(message.Mentions.Parse) != 0 ||
		len(message.Mentions.Roles) != 0 || len(message.Mentions.Users) != 0 {
		t.Errorf("Discord message can mention: %+v", message)
	}
	if len(message.Embeds) != 1 || message.Embeds[0].Title != "Ilse of the west shelf" ||
		!strings.Contains(message.Embeds[0].Description, "@everyone") ||
		message.Embeds[0].URL != event.Update.HistoryURL || message.Embeds[0].Footer.Text != "Illarin" {
		t.Errorf("embed = %+v", message.Embeds)
	}
	if len(message.Embeds[0].Fields) != 1 || message.Embeds[0].Fields[0].Name != "Update" ||
		message.Embeds[0].Fields[0].Value != "v2" {
		t.Errorf("fields = %+v, want the version label alone", message.Embeds[0].Fields)
	}

	for _, one := range stack.announcements(t, stack.editor, started.ID) {
		if one.State != "delivered" || one.SettledReason != "arrived" || one.Tries != 1 {
			t.Errorf("settled announcement = %+v", one)
		}
		if one.ID != webhookID && one.Type == "webhook" {
			t.Errorf("the webhook-id %q is not the attempt %q", webhookID, one.ID)
		}
		if one.Type == "discord" && one.MessageID != discordMessageID {
			t.Errorf("Discord announcement kept message %q", one.MessageID)
		}
	}
	if again := stack.sendAnnouncementsAt(t, time.Now().Add(time.Hour)); again != 0 {
		t.Errorf("a delivered announcement was sent %d more times", again)
	}
}

func TestOnlyAPublishedUpdateAnnounces(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	started := apitest.StartCharacter(t, stack.router, stack.editor)
	apitest.WriteCharacterFloor(t, stack.router, stack.editor, started)
	chosen := stack.integrationRequest(t, stack.editor, http.MethodPut,
		"/v1/works/"+started.ID+"/integrations",
		fmt.Sprintf(`{"integrationIds":[%q]}`, hook.Integration.ID))
	if chosen.Code != http.StatusNoContent {
		t.Fatalf("remember defaults = %d", chosen.Code)
	}
	if got := apitest.PublishWork(t, stack.router, stack.editor, started.ID); got.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", got.Code, got.Body.String())
	}
	stack.describe(t, stack.editor, started, "A private save changes nothing public.")
	correction := apitest.AuthorizedJSONRequest(t, http.MethodPatch,
		"/v1/works/"+started.ID+"/versions/1/notes",
		`{"summary":"Corrected summary","notes":"Corrected context."}`, stack.editor)
	if got := apitest.Send(t, stack.router, correction); got.Code != http.StatusNoContent {
		t.Fatalf("correct notes = %d: %s", got.Code, got.Body.String())
	}

	stack.sendAnnouncementsAt(t, time.Now())
	if len(stack.announcements(t, stack.editor, started.ID)) != 0 || len(stack.to.arrivals()) != 0 {
		t.Fatal("first publication, a private save or a note correction announced")
	}

	stack.announced(t, stack.editor, started.ID, `{"summary":"Now she is public"}`)
	sent := stack.onlyAnnouncement(t, stack.editor, started.ID)
	if sent.Integration != "Creator updates" || sent.VersionNumber != 2 {
		t.Errorf("the remembered integration was not used: %+v", sent)
	}
	stack.sendAnnouncementsAt(t, time.Now())
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the remembered integration received %d requests, want 1", len(arrivals))
	}
	var event workUpdateEvent
	if err := json.Unmarshal(arrivals[0].Body, &event); err != nil {
		t.Fatal(err)
	}
	if event.Update.Summary != "Now she is public" {
		t.Errorf("event = %+v", event.Update)
	}

	correction = apitest.AuthorizedJSONRequest(t, http.MethodPatch,
		"/v1/works/"+started.ID+"/versions/2/notes",
		`{"summary":"Corrected again","notes":""}`, stack.editor)
	if got := apitest.Send(t, stack.router, correction); got.Code != http.StatusNoContent {
		t.Fatalf("correct notes = %d: %s", got.Code, got.Body.String())
	}
	stack.sendAnnouncementsAt(t, time.Now().Add(time.Hour))
	if len(stack.to.arrivals()) != 1 || len(stack.announcements(t, stack.editor, started.ID)) != 1 {
		t.Error("a note correction resent or edited the announcement")
	}
}

func TestAnExplicitSelectionIsRememberedAndAnEmptyOneAnnouncesNowhere(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	started := stack.publishedCharacter(t, stack.editor)

	stack.describe(t, stack.editor, started, "First change.")
	stack.announced(t, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"First change","integrationIds":[%q]}`, hook.Integration.ID))
	choices := stack.integrationRequest(t, stack.editor, http.MethodGet,
		"/v1/works/"+started.ID+"/integrations", "")
	var offered struct {
		Integrations []integrationChoice `json:"integrations"`
	}
	if err := json.Unmarshal(choices.Body.Bytes(), &offered); err != nil {
		t.Fatal(err)
	}
	if len(offered.Integrations) != 1 || !offered.Integrations[0].ByDefault {
		t.Fatalf("after an explicit selection the work remembers %+v", offered.Integrations)
	}

	stack.describe(t, stack.editor, started, "Second change.")
	stack.announced(t, stack.editor, started.ID, `{"summary":"Second change","integrationIds":[]}`)
	listed := stack.announcements(t, stack.editor, started.ID)
	if len(listed) != 1 || listed[0].VersionNumber != 2 {
		t.Fatalf("an empty selection queued something: %+v", listed)
	}
	choices = stack.integrationRequest(t, stack.editor, http.MethodGet,
		"/v1/works/"+started.ID+"/integrations", "")
	if err := json.Unmarshal(choices.Body.Bytes(), &offered); err != nil {
		t.Fatal(err)
	}
	if offered.Integrations[0].ByDefault {
		t.Error("an empty selection left the old selection remembered")
	}
}

func TestAnUnlistedWorkAnnouncesOnlyWithExplicitConsent(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	started := stack.publishedCharacter(t, stack.editor)
	chosen := stack.integrationRequest(t, stack.editor, http.MethodPut,
		"/v1/works/"+started.ID+"/integrations",
		fmt.Sprintf(`{"integrationIds":[%q]}`, hook.Integration.ID))
	if chosen.Code != http.StatusNoContent {
		t.Fatalf("remember defaults = %d", chosen.Code)
	}
	unlisted := apitest.Send(t, stack.router, apitest.AuthorizedJSONRequest(t, http.MethodPut,
		"/v1/works/"+started.ID+"/visibility", `{"visibility":"unlisted"}`, stack.editor))
	if unlisted.Code != http.StatusNoContent {
		t.Fatalf("unlist = %d: %s", unlisted.Code, unlisted.Body.String())
	}

	stack.describe(t, stack.editor, started, "Quiet by default.")
	stack.announced(t, stack.editor, started.ID, `{"summary":"Quiet by default"}`)
	if len(stack.announcements(t, stack.editor, started.ID)) != 0 {
		t.Fatal("an unlisted update used remembered integrations without consent")
	}

	stack.describe(t, stack.editor, started, "Needs consent.")
	refused := apitest.PublishWorkVersion(t, stack.router, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"Needs consent","integrationIds":[%q]}`, hook.Integration.ID))
	if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "announceUnlisted") {
		t.Fatalf("selecting a integration for an unlisted work = %d: %s", refused.Code, refused.Body.String())
	}
	history := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/versions", nil), stack.editor))
	if strings.Contains(history.Body.String(), "Needs consent") {
		t.Fatal("a refused announcement left the update published")
	}

	stack.announced(t, stack.editor, started.ID, fmt.Sprintf(
		`{"summary":"Needs consent","integrationIds":[%q],"announceUnlisted":true}`, hook.Integration.ID))
	stack.sendAnnouncementsAt(t, time.Now())
	if len(stack.to.arrivals()) != 1 {
		t.Fatalf("a consented unlisted announcement made %d requests, want 1", len(stack.to.arrivals()))
	}
	var event workUpdateEvent
	if err := json.Unmarshal(stack.to.arrivals()[0].Body, &event); err != nil {
		t.Fatal(err)
	}
	if event.Work.URL != "http://localhost:3000/a/"+started.ID {
		t.Errorf("the unlisted announcement carries %q", event.Work.URL)
	}
}

func TestAnIneligibleDestinationRollsThePublicationBack(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	other := apitest.VerifiedSignUp(t, stack.router, stack.outbox, "other@example.com", "other.creator")
	theirs := stack.creatorWebhook(t, other)
	disabled := stack.creatorWebhook(t, stack.editor)
	if got := stack.integrationRequest(t, stack.editor, http.MethodDelete,
		integrationsPath+"/"+disabled.Integration.ID+"/verification", ""); got.Code != http.StatusOK {
		t.Fatalf("disable = %d", got.Code)
	}
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "Changed.")

	for _, id := range []string{theirs.Integration.ID, disabled.Integration.ID} {
		refused := apitest.PublishWorkVersion(t, stack.router, stack.editor, started.ID,
			fmt.Sprintf(`{"summary":"Changed","integrationIds":[%q]}`, id))
		if refused.Code != http.StatusBadRequest || !strings.Contains(refused.Body.String(), "integrationIds") {
			t.Fatalf("publishing to an ineligible integration = %d: %s", refused.Code, refused.Body.String())
		}
	}
	history := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/versions", nil), stack.editor))
	if strings.Contains(history.Body.String(), `"number":2`) {
		t.Fatal("a refused announcement left the update published")
	}
	var events int
	if err := stack.pool.QueryRow(context.Background(), `select count(*) from work_announcements`).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 0 {
		t.Errorf("a rolled-back publication left %d events", events)
	}
}

func TestAnnouncementsRetryOnTheSharedScheduleWithAnInjectedClock(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "Retried.")
	stack.announced(t, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"Retried","integrationIds":[%q]}`, hook.Integration.ID))
	stack.answersWith(http.StatusServiceUnavailable)

	at := time.Now().UTC()
	for place, gap := range dispatch.Delays[1:] {
		if attempted := stack.sendAnnouncementsAt(t, at); attempted != 1 {
			t.Fatalf("attempt %d made %d requests, want 1", place+1, attempted)
		}
		waiting := stack.onlyAnnouncement(t, stack.editor, started.ID)
		if waiting.State != "pending" {
			t.Fatalf("after attempt %d the announcement is %q, want pending", place+1, waiting.State)
		}
		waited := waiting.DueAt.Sub(at)
		if waited < gap*9/10 || waited > gap*11/10 {
			t.Fatalf("attempt %d waits %s, want about %s", place+2, waited, gap)
		}
		if early := stack.sendAnnouncementsAt(t, at.Add(gap*8/10)); early != 0 {
			t.Fatalf("attempt %d ran %d requests before it was due", place+2, early)
		}
		at = waiting.DueAt
	}
	if last := stack.sendAnnouncementsAt(t, at); last != 1 {
		t.Fatalf("the last attempt made %d requests, want 1", last)
	}
	spent := stack.onlyAnnouncement(t, stack.editor, started.ID)
	if spent.State != "failed" || spent.SettledReason != "exhausted" || spent.Tries != dispatch.MaxTries {
		t.Errorf("the run ended %+v, want failed/exhausted after %d attempts", spent, dispatch.MaxTries)
	}
	if again := stack.sendAnnouncementsAt(t, at.Add(365*24*time.Hour)); again != 0 {
		t.Errorf("a spent run made %d further requests", again)
	}
	for _, one := range stack.to.arrivals() {
		if one.Headers.Get(dispatch.IDHeader) != spent.ID {
			t.Fatalf("an attempt carried webhook-id %q, want the attempt %q", one.Headers.Get(dispatch.IDHeader), spent.ID)
		}
	}
	history := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/versions", nil), stack.editor))
	if !strings.Contains(history.Body.String(), "Retried") {
		t.Error("attempt failure undid the publication")
	}
}

func TestEveryAttemptRechecksTheWorkAndTheDestination(t *testing.T) {
	t.Parallel()
	type revoke func(t *testing.T, stack integrationStack, started apitest.StartedWork, hook addedIntegration)
	for name, one := range map[string]struct {
		reason string
		act    revoke
	}{
		"withheld": {"withheld", func(t *testing.T, stack integrationStack, started apitest.StartedWork, _ addedIntegration) {
			withheld := apitest.Send(t, stack.router, apitest.AuthorizedJSONRequest(t, http.MethodPut,
				"/v1/works/"+started.ID+"/withhold", `{"reason":"Under review"}`, stack.editor))
			if withheld.Code != http.StatusNoContent {
				t.Fatalf("withhold = %d: %s", withheld.Code, withheld.Body.String())
			}
		}},
		"unlisted": {"unlisted", func(t *testing.T, stack integrationStack, started apitest.StartedWork, _ addedIntegration) {
			unlisted := apitest.Send(t, stack.router, apitest.AuthorizedJSONRequest(t, http.MethodPut,
				"/v1/works/"+started.ID+"/visibility", `{"visibility":"unlisted"}`, stack.editor))
			if unlisted.Code != http.StatusNoContent {
				t.Fatalf("unlist = %d: %s", unlisted.Code, unlisted.Body.String())
			}
		}},
		"deleted": {"deleted", func(t *testing.T, stack integrationStack, started apitest.StartedWork, _ addedIntegration) {
			deleted := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
				http.MethodDelete, "/v1/works/"+started.ID, nil), stack.editor))
			if deleted.Code != http.StatusNoContent {
				t.Fatalf("delete = %d: %s", deleted.Code, deleted.Body.String())
			}
		}},
		"withdrawn": {"withdrawn", func(t *testing.T, stack integrationStack, started apitest.StartedWork, _ addedIntegration) {
			stack.describe(t, stack.editor, started, "A replacement so the old one can go.")
			stack.announced(t, stack.editor, started.ID, `{"summary":"Replacement","integrationIds":[]}`)
			withdraw := apitest.AuthorizedJSONRequest(t, http.MethodPost,
				"/v1/works/"+started.ID+"/versions/2/withdraw",
				`{"explanation":"This version gave incorrect guidance."}`, stack.editor)
			if got := apitest.Send(t, stack.router, withdraw); got.Code != http.StatusNoContent {
				t.Fatalf("withdraw = %d: %s", got.Code, got.Body.String())
			}
		}},
		"disabled": {"disabled", func(t *testing.T, stack integrationStack, _ apitest.StartedWork, hook addedIntegration) {
			got := stack.integrationRequest(t, stack.editor, http.MethodDelete,
				integrationsPath+"/"+hook.Integration.ID+"/verification", "")
			if got.Code != http.StatusOK {
				t.Fatalf("disable = %d: %s", got.Code, got.Body.String())
			}
		}},
		"removed": {"removed", func(t *testing.T, stack integrationStack, _ apitest.StartedWork, hook addedIntegration) {
			got := stack.integrationRequest(t, stack.editor, http.MethodDelete,
				integrationsPath+"/"+hook.Integration.ID, "")
			if got.Code != http.StatusNoContent {
				t.Fatalf("remove = %d: %s", got.Code, got.Body.String())
			}
		}},
	} {
		t.Run(name, func(t *testing.T) {
			stack := newIntegrationStack(t)
			hook := stack.creatorWebhook(t, stack.editor)
			started := stack.publishedCharacter(t, stack.editor)
			stack.describe(t, stack.editor, started, "Changed before revocation.")
			stack.announced(t, stack.editor, started.ID,
				fmt.Sprintf(`{"summary":"Changed","integrationIds":[%q]}`, hook.Integration.ID))
			stack.answersWith(http.StatusServiceUnavailable)
			at := time.Now().UTC()
			if first := stack.sendAnnouncementsAt(t, at); first != 1 {
				t.Fatalf("the first attempt made %d requests", first)
			}
			stack.to.forget()
			stack.answersWith(http.StatusOK)

			one.act(t, stack, started, hook)

			stack.sendAnnouncementsAt(t, at.Add(24*time.Hour))
			if len(stack.to.arrivals()) != 0 {
				t.Fatalf("an announcement was sent after %s", name)
			}
			var state, reason string
			err := stack.pool.QueryRow(context.Background(), `
				select state, coalesce(settled_reason, '') from work_announcement_attempts
				 where announcement_id = (select id from work_announcements order by occurred_at limit 1)
			`).Scan(&state, &reason)
			if err != nil {
				t.Fatalf("read the cancelled announcement: %v", err)
			}
			if state != "failed" || reason != one.reason {
				t.Errorf("after %s the announcement is %s/%s, want failed/%s", name, state, reason, one.reason)
			}
		})
	}
}

func TestAnUnconfirmedDiscordAnnouncementStaysVisibleAndIsNotSentAgain(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	channel := stack.addUpdateDestination(t, stack.editor, "discord", discordCapability())
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "Unconfirmed.")
	stack.announced(t, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"Unconfirmed","integrationIds":[%q]}`, channel.Integration.ID))
	stack.discord.answersSendWith(func(arrived) (int, string) { return http.StatusNoContent, "" })

	stack.sendAnnouncementsAt(t, time.Now())
	held := stack.onlyAnnouncement(t, stack.editor, started.ID)
	if held.State != "unconfirmed" || held.SettledReason != "unconfirmed" || held.MessageID != "" {
		t.Fatalf("an unconfirmed send shows as %+v", held)
	}
	if again := stack.sendAnnouncementsAt(t, time.Now().Add(48*time.Hour)); again != 0 ||
		len(stack.discord.announcements()) != 1 {
		t.Error("an unconfirmed announcement was sent again")
	}
}

func TestAnnouncementStatusIsTheOwnersAloneAndCarriesNoSecrets(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "Status.")
	stack.announced(t, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"Status","integrationIds":[%q]}`, hook.Integration.ID))
	stack.answersWith(http.StatusTeapot)
	stack.sendAnnouncementsAt(t, time.Now())

	for _, session := range []*http.Cookie{stack.authority, nil} {
		request := httptest.NewRequest(http.MethodGet, "/v1/works/"+started.ID+"/announcement-attempts", nil)
		if session != nil {
			request = apitest.Authorized(request, session)
		}
		response := apitest.Send(t, stack.router, request)
		if response.Code != http.StatusNotFound && response.Code != http.StatusUnauthorized {
			t.Errorf("another account read announcements with %d", response.Code)
		}
	}
	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/works/"+started.ID+"/announcement-attempts", nil), stack.editor))
	body := response.Body.String()
	if strings.Contains(body, hook.Secret) || strings.Contains(body, stack.to.address()) {
		t.Error("announcement status carries credentials or the endpoint address")
	}
	held := stack.onlyAnnouncement(t, stack.editor, started.ID)
	if held.State != "failed" || held.SettledReason != "refused" || held.Last == nil ||
		held.Last.Status == nil || *held.Last.Status != http.StatusTeapot || held.Last.Detail == "" {
		t.Errorf("a refused announcement shows as %+v", held)
	}
}

func TestARotatedSecretSignsAnnouncementsTwiceDuringTheOverlap(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	hook := stack.creatorWebhook(t, stack.editor)
	rotated := stack.integrationRequest(t, stack.editor, http.MethodPost,
		integrationsPath+"/"+hook.Integration.ID+"/secret", "")
	var fresh addedIntegration
	if rotated.Code != http.StatusOK || json.Unmarshal(rotated.Body.Bytes(), &fresh) != nil {
		t.Fatalf("rotate = %d", rotated.Code)
	}
	started := stack.publishedCharacter(t, stack.editor)
	stack.describe(t, stack.editor, started, "Rotated.")
	stack.announced(t, stack.editor, started.ID,
		fmt.Sprintf(`{"summary":"Rotated","integrationIds":[%q]}`, hook.Integration.ID))
	stack.sendAnnouncementsAt(t, time.Now())

	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("received %d requests, want 1", len(arrivals))
	}
	id, stamp := arrivals[0].Headers.Get(dispatch.IDHeader), stampOf(t, arrivals[0])
	signature := arrivals[0].Headers.Get(dispatch.SignatureHeader)
	if !dispatch.Accepts(signature, hook.Secret, id, stamp, arrivals[0].Body) ||
		!dispatch.Accepts(signature, fresh.Secret, id, stamp, arrivals[0].Body) {
		t.Error("the announcement is not accepted by both the old and the new secret")
	}
}
