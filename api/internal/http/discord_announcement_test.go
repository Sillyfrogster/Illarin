package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

const (
	discordWebhookID = "1234567890123456789"
	discordGuildID   = "111111111111111111"
	discordChannelID = "222222222222222222"
	discordRoleID    = "333333333333333333"
	discordMessageID = "444444444444444444"
	discordToken     = "a-test-webhook-token"
)

func discordCapability() string {
	return "https://discord.com/api/webhooks/" + discordWebhookID + "/" + discordToken
}

type discordServer struct {
	server *httptest.Server

	mu      sync.Mutex
	got     []arrived
	webhook func() (int, string)
	send    func(arrived) (int, string)
}

func newDiscordServer(t *testing.T) *discordServer {
	t.Helper()
	held := &discordServer{}
	held.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err != nil {
			t.Errorf("read what arrived at Discord: %v", err)
		}
		one := arrived{Headers: r.Header.Clone(), Body: body}
		held.mu.Lock()
		if r.Method == http.MethodPost {
			held.got = append(held.got, one)
		}
		reading, sending := held.webhook, held.send
		held.mu.Unlock()
		status, said := http.StatusOK, ""
		if r.Method == http.MethodGet {
			status, said = http.StatusOK, fmt.Sprintf(
				`{"id":%q,"guild_id":%q,"channel_id":%q,"name":"Illarin Blog","token":%q}`,
				discordWebhookID, discordGuildID, discordChannelID, discordToken,
			)
			if reading != nil {
				status, said = reading()
			}
		} else {
			status, said = http.StatusOK, fmt.Sprintf(`{"id":%q}`, discordMessageID)
			if sending != nil {
				status, said = sending(one)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(said))
	}))
	t.Cleanup(held.server.Close)
	return held
}

func (d *discordServer) announcements() []arrived {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]arrived(nil), d.got...)
}

func (d *discordServer) answersReadWith(with func() (int, string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.webhook = with
}

func (d *discordServer) answersSendWith(with func(arrived) (int, string)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.send = with
}

type announcement struct {
	Content string `json:"content"`
	Embeds  []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Timestamp   string `json:"timestamp"`
		Author      struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"author"`
		Fields []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"fields"`
		Image struct {
			URL string `json:"url"`
		} `json:"image"`
		Footer struct {
			Text string `json:"text"`
		} `json:"footer"`
	} `json:"embeds"`
	Mentions struct {
		Parse []string `json:"parse"`
		Roles []string `json:"roles"`
		Users []string `json:"users"`
	} `json:"allowed_mentions"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	ThreadID  string `json:"thread_id"`
}

func (s destinationStack) sentToDiscord(t *testing.T) announcement {
	t.Helper()
	arrivals := s.discord.announcements()
	if len(arrivals) != 1 {
		t.Fatalf("Discord was sent %d announcements, want 1", len(arrivals))
	}
	var read announcement
	if err := json.Unmarshal(arrivals[0].Body, &read); err != nil {
		t.Fatalf("decode the announcement: %v", err)
	}
	return read
}

func (s destinationStack) addChannel(
	t *testing.T,
	session *http.Cookie,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return send(t, s.router, authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/channels", body,
	), session))
}

func (s destinationStack) channelWithRole(t *testing.T, role string) destination {
	t.Helper()
	body := fmt.Sprintf(`{"name":"Announcements","address":%q}`, discordCapability())
	if role != "" {
		body = fmt.Sprintf(
			`{"name":"Announcements","address":%q,"roleId":%q,"roleName":%q}`,
			discordCapability(), discordRoleID, role,
		)
	}
	response := s.addChannel(t, s.authority, body)
	if response.Code != http.StatusCreated {
		t.Fatalf("add channel status = %d: %s", response.Code, response.Body.String())
	}
	var made destination
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode channel: %v", err)
	}
	return made
}

func (s destinationStack) announcedPost(t *testing.T, choice string) (blogPost, []postDelivery) {
	t.Helper()
	post := s.readyPost(t)
	response := s.publishTo(t, s.editor, post.ID, post.Version, choosing(post, choice))
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	s.sendQueued(t)
	return post, s.deliveries(t, s.editor, post.ID).Deliveries
}

func choosing(post blogPost, choice string) string {
	return fmt.Sprintf(`{"version":%d,%s}`, post.Version, choice)
}

func TestOnlyThePublicationAuthorityConfiguresADiscordChannel(t *testing.T) {
	stack := newDestinationStack(t)
	member := stack.member(t, "writer@example.com", "outside.writer")

	response := stack.addChannel(t, member, fmt.Sprintf(
		`{"name":"Theirs","address":%q}`, discordCapability(),
	))

	if response.Code != http.StatusForbidden {
		t.Errorf("add channel status = %d, want 403", response.Code)
	}
}

func TestOnlyADiscordWebhookAddressBecomesAChannel(t *testing.T) {
	stack := newDestinationStack(t)

	for _, address := range []string{
		"https://example.com/api/webhooks/1234567890123456789/token",
		"http://discord.com/api/webhooks/1234567890123456789/token",
		"https://discord.com/api/webhooks/1234567890123456789",
		"https://discord.com/api/webhooks/nope/token",
	} {
		response := stack.addChannel(t, stack.authority, fmt.Sprintf(
			`{"name":"Announcements","address":%q}`, address,
		))
		if response.Code != http.StatusBadRequest {
			t.Errorf("add %s status = %d, want 400", address, response.Code)
		}
	}
}

func TestAChannelIsOnlyConfiguredWhenDiscordConfirmsIt(t *testing.T) {
	stack := newDestinationStack(t)
	stack.discord.answersReadWith(func() (int, string) { return http.StatusUnauthorized, `{}` })

	refused := stack.addChannel(t, stack.authority, fmt.Sprintf(
		`{"name":"Announcements","address":%q}`, discordCapability(),
	))

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("add channel status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
	if listed := stack.destinations(t, stack.authority); len(listed.Destinations) != 0 {
		t.Errorf("a refused capability left %d destinations", len(listed.Destinations))
	}
}

func TestAMissingWebhookSaysWhatToDoAboutIt(t *testing.T) {
	stack := newDestinationStack(t)
	stack.discord.answersReadWith(func() (int, string) { return http.StatusNotFound, `{}` })

	refused := stack.addChannel(t, stack.authority, fmt.Sprintf(
		`{"name":"Announcements","address":%q}`, discordCapability(),
	))

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("add channel status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
	if !strings.Contains(refused.Body.String(), "does not recognize that webhook") {
		t.Errorf("refusal = %s, want it to name the fix", refused.Body.String())
	}
}

func TestAWebhookOutsideAGuildChannelIsRefused(t *testing.T) {
	stack := newDestinationStack(t)
	stack.discord.answersReadWith(func() (int, string) {
		return http.StatusOK, `{"id":"1234567890123456789","name":"Somewhere"}`
	})

	refused := stack.addChannel(t, stack.authority, fmt.Sprintf(
		`{"name":"Announcements","address":%q}`, discordCapability(),
	))

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("add channel status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
}

func TestAChannelKeepsOnlySafeIdentityAndIsReadyToAnnounce(t *testing.T) {
	stack := newDestinationStack(t)

	made := stack.channelWithRole(t, "Blog readers")

	if made.Kind != "discord" || made.State != "active" {
		t.Errorf("channel = %s in %s, want an active discord destination", made.Kind, made.State)
	}
	if made.Channel == nil {
		t.Fatal("the channel identity is missing")
	}
	if made.Channel.GuildID != discordGuildID || made.Channel.ChannelID != discordChannelID {
		t.Errorf("channel identity = %+v", made.Channel)
	}
	if made.Channel.RoleName != "Blog readers" {
		t.Errorf("role name = %q", made.Channel.RoleName)
	}
	if len(made.Events) != 1 || made.Events[0] != "publication.post.published.v1" {
		t.Errorf("events = %v, want first publications alone", made.Events)
	}
	body, _ := json.Marshal(stack.destinations(t, stack.authority))
	if strings.Contains(string(body), discordToken) {
		t.Error("the listing carried the Discord capability token")
	}
	var sealed []byte
	err := stack.pool.QueryRow(t.Context(), `
		select address from publication_destinations where id = $1
	`, made.ID).Scan(&sealed)
	if err != nil {
		t.Fatalf("read the sealed capability: %v", err)
	}
	if strings.Contains(string(sealed), discordToken) {
		t.Error("the stored capability holds the token in the clear")
	}
}

func TestOnlyASnowflakeRoleIsApproved(t *testing.T) {
	stack := newDestinationStack(t)

	refused := stack.addChannel(t, stack.authority, fmt.Sprintf(
		`{"name":"Announcements","address":%q,"roleId":"everyone","roleName":"Everyone"}`,
		discordCapability(),
	))

	if refused.Code != http.StatusBadRequest {
		t.Errorf("add channel status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
}

func TestADiscordChannelHasNoSigningSecretToRotate(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")

	response := stack.rotate(t, stack.authority, made.ID)

	if response.Code != http.StatusBadRequest {
		t.Errorf("rotate status = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func TestAContributorSeesTheChannelNameAndItsRoleAndNothingElse(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "Blog readers")
	post := stack.readyPost(t)

	offered := stack.postChoices(t, stack.editor, post.ID)

	if len(offered.Destinations) != 1 {
		t.Fatalf("offered %d destinations, want 1", len(offered.Destinations))
	}
	one := offered.Destinations[0]
	if one.ID != made.ID || one.Kind != "discord" || one.Name != "Announcements" {
		t.Errorf("offered = %+v", one)
	}
	if one.Role != "Blog readers" {
		t.Errorf("role = %q, want the name the authority approved", one.Role)
	}
	body, _ := json.Marshal(offered)
	for _, secret := range []string{discordToken, discordRoleID, discordChannelID} {
		if strings.Contains(string(body), secret) {
			t.Errorf("the contributor's view carried %q", secret)
		}
	}
}

func TestAFirstPublicationAnnouncesWhatIllarinDecidedToSay(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")

	post, sent := stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q],"note":"Worth a read."`, made.ID,
	))

	if len(sent) != 1 || sent[0].State != "delivered" {
		t.Fatalf("deliveries = %+v, want one delivered", sent)
	}
	if sent[0].Kind != "discord" || sent[0].MessageID != discordMessageID {
		t.Errorf("delivery = %+v, want the Discord message it made", sent[0])
	}
	read := stack.sentToDiscord(t)
	if len(read.Embeds) != 1 {
		t.Fatalf("embeds = %d, want 1", len(read.Embeds))
	}
	embed := read.Embeds[0]
	if embed.Title != post.Title {
		t.Errorf("title = %q, want %q", embed.Title, post.Title)
	}
	if embed.Description != "What Illarin changed this week." {
		t.Errorf("description = %q, want the hand-written summary", embed.Description)
	}
	if embed.URL != "http://blog.localhost:3000/"+post.Slug {
		t.Errorf("url = %q, want the canonical address", embed.URL)
	}
	if embed.Author.Name == "" || !strings.Contains(embed.Author.URL, "/@") {
		t.Errorf("author = %+v, want the stored attribution", embed.Author)
	}
	if len(embed.Fields) != 1 || embed.Fields[0].Value != "Announcement" {
		t.Errorf("fields = %+v, want the category", embed.Fields)
	}
	if embed.Timestamp == "" || embed.Footer.Text != "Illarin Blog" {
		t.Errorf("embed = %+v, want a publication time and the publication's name", embed)
	}
	if read.Content != "Worth a read." {
		t.Errorf("content = %q, want the shared note", read.Content)
	}
}

func TestAnAnnouncementNeverCarriesContributorSuppliedDiscordFields(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")

	stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q],"username":"Someone","embeds":[],"threadId":"1"`,
		made.ID,
	))

	read := stack.sentToDiscord(t)
	if read.Username != "" || read.AvatarURL != "" || read.ThreadID != "" {
		t.Errorf("the announcement carried an identity or thread override: %+v", read)
	}
}

func TestMentionsAreClosedUnlessTheApprovedRoleIsChosen(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "Blog readers")

	stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q],"note":"@everyone <@&999999999999999999>"`, made.ID,
	))

	read := stack.sentToDiscord(t)
	if len(read.Mentions.Parse) != 0 {
		t.Errorf("parse = %v, want nothing parsed", read.Mentions.Parse)
	}
	if len(read.Mentions.Roles) != 0 || len(read.Mentions.Users) != 0 {
		t.Errorf("mentions = %+v, want nobody named", read.Mentions)
	}
}

func TestChoosingTheRoleMentionsThatOneAndOnlyThatOne(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "Blog readers")

	stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q],"roleDestinationIds":[%q]`, made.ID, made.ID,
	))

	read := stack.sentToDiscord(t)
	if len(read.Mentions.Roles) != 1 || read.Mentions.Roles[0] != discordRoleID {
		t.Errorf("roles = %v, want the one approved role", read.Mentions.Roles)
	}
	if !strings.Contains(read.Content, "<@&"+discordRoleID+">") {
		t.Errorf("content = %q, want the role named", read.Content)
	}
}

func TestARoleCannotBeChosenWhereTheAuthorityApprovedNone(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")
	post := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, post.ID, post.Version, choosing(post, fmt.Sprintf(
		`"destinationIds":[%q],"roleDestinationIds":[%q]`, made.ID, made.ID,
	)))

	if response.Code != http.StatusForbidden {
		t.Errorf("publish status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestARoleCannotBeChosenOnADestinationNothingIsSentTo(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "Blog readers")
	post := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, post.ID, post.Version, choosing(post, fmt.Sprintf(
		`"destinationIds":[],"roleDestinationIds":[%q]`, made.ID,
	)))

	if response.Code != http.StatusForbidden {
		t.Errorf("publish status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestQuietPublicationAnnouncesNothingOnDiscord(t *testing.T) {
	stack := newDestinationStack(t)
	stack.channelWithRole(t, "")

	stack.announcedPost(t, `"destinationIds":[]`)

	if arrivals := stack.discord.announcements(); len(arrivals) != 0 {
		t.Errorf("Discord was sent %d announcements, want 0", len(arrivals))
	}
}

func TestAnAcceptedAnnouncementDiscordNeverConfirmsIsNotCalledDelivered(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")
	stack.discord.answersSendWith(func(arrived) (int, string) { return http.StatusNoContent, "" })

	_, sent := stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q]`, made.ID,
	))

	if len(sent) != 1 {
		t.Fatalf("deliveries = %d, want 1", len(sent))
	}
	if sent[0].State != "unconfirmed" || sent[0].SettledReason != "unconfirmed" {
		t.Errorf("delivery = %+v, want it recorded as unconfirmed", sent[0])
	}
	if sent[0].MessageID != "" {
		t.Errorf("message id = %q, want none", sent[0].MessageID)
	}
	if sent[0].Last == nil || sent[0].Last.Outcome != "unconfirmed" {
		t.Errorf("attempt = %+v, want an unconfirmed outcome", sent[0].Last)
	}
	if again := stack.sendQueued(t); again != 0 {
		t.Errorf("%d unconfirmed announcements were tried again", again)
	}
}

func TestAPostAnnouncesOnDiscordOnceAcrossItsWholeLife(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")
	chosen := fmt.Sprintf(`"destinationIds":[%q]`, made.ID)

	post, sent := stack.announcedPost(t, chosen)
	if len(sent) != 1 {
		t.Fatalf("the first publication made %d deliveries, want 1", len(sent))
	}

	live := stack.working(t, stack.editor, post.ID)
	changed := stack.saved(t, stack.editor, post.ID, finished(live, map[string]any{
		"version": live.Version, "summary": "What Illarin changed this week, corrected.",
	}))
	republished := stack.publishTo(t, stack.editor, post.ID, changed.Version,
		choosing(changed, chosen))
	if republished.Code != http.StatusOK {
		t.Fatalf("publish changes status = %d: %s", republished.Code, republished.Body.String())
	}
	stack.sendQueued(t)

	after := stack.deliveries(t, stack.editor, post.ID).Deliveries
	if len(after) != 1 {
		t.Errorf("the post made %d deliveries, want the one announcement", len(after))
	}
	if arrivals := stack.discord.announcements(); len(arrivals) != 1 {
		t.Errorf("Discord was sent %d announcements, want 1", len(arrivals))
	}
}

func TestAWithdrawnAndRepublishedPostAnnouncesNothingFurther(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")
	chosen := fmt.Sprintf(`"destinationIds":[%q]`, made.ID)

	post, _ := stack.announcedPost(t, chosen)
	live := stack.working(t, stack.editor, post.ID)
	taken := stack.withdraw(t, stack.editor, post.ID, fmt.Sprintf(
		`{"version":%d,"reason":"A correction is coming.",%s}`, live.Version, chosen,
	))
	if taken.Code != http.StatusOK {
		t.Fatalf("withdraw status = %d: %s", taken.Code, taken.Body.String())
	}
	down := decodePost(t, taken)
	back := stack.republish(t, stack.editor, post.ID, fmt.Sprintf(
		`{"version":%d,"revisionId":%q,%s}`, down.Version, down.PublicRevision, chosen,
	))
	if back.Code != http.StatusOK {
		t.Fatalf("republish status = %d: %s", back.Code, back.Body.String())
	}
	stack.sendQueued(t)

	if arrivals := stack.discord.announcements(); len(arrivals) != 1 {
		t.Errorf("Discord was sent %d announcements, want the historical one", len(arrivals))
	}
}

func TestADiscordAnnouncementNeverDelaysPublication(t *testing.T) {
	stack := newDestinationStack(t)
	made := stack.channelWithRole(t, "")
	stack.discord.answersSendWith(func(arrived) (int, string) {
		return http.StatusInternalServerError, `{}`
	})

	post, sent := stack.announcedPost(t, fmt.Sprintf(
		`"destinationIds":[%q]`, made.ID,
	))

	if len(sent) != 1 || sent[0].State == "delivered" {
		t.Fatalf("deliveries = %+v, want one that has not arrived", sent)
	}
	live := stack.working(t, stack.editor, post.ID)
	if live.Status != "published" {
		t.Errorf("the post is %q, want it published anyway", live.Status)
	}
}
