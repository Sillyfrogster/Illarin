package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
	"github.com/Sillyfrogster/Illarin/api/internal/testdb"
)

type integrationRow struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	Name          string     `json:"name"`
	Host          string     `json:"host"`
	Address       string     `json:"address"`
	State         string     `json:"state"`
	Announcements []string   `json:"announcements"`
	Channel       *channel   `json:"channel"`
	SecretSetAt   time.Time  `json:"secretSetAt"`
	OldUntil      *time.Time `json:"previousSecretUntil"`
	VerifiedAt    *string    `json:"verifiedAt"`
	DisabledAt    *string    `json:"disabledAt"`
}

type channel struct {
	GuildID   string `json:"guildId"`
	ChannelID string `json:"channelId"`
	Webhook   string `json:"webhookName"`
	RoleID    string `json:"roleId"`
	RoleName  string `json:"roleName"`
}

type addedIntegration struct {
	Integration integrationRow `json:"integration"`
	Secret      string         `json:"secret"`
}

type integrationList struct {
	Integrations []integrationRow `json:"integrations"`
}

type integrationChoice struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Type          string   `json:"type"`
	State         string   `json:"state"`
	Announcements []string `json:"announcements"`
	Role          string   `json:"role"`
	ByDefault     bool     `json:"byDefault"`
}

type destinationChoiceList struct {
	Integrations []integrationChoice `json:"integrations"`
	Inherited    bool                `json:"inherited"`
}

type announcementTry struct {
	Run         int       `json:"run"`
	Number      int       `json:"number"`
	Outcome     string    `json:"outcome"`
	Status      *int      `json:"status"`
	Detail      string    `json:"detail"`
	TookMs      int       `json:"tookMs"`
	AttemptedAt time.Time `json:"attemptedAt"`
}

type postAttempt struct {
	ID               string           `json:"id"`
	AnnouncementID   string           `json:"announcementId"`
	AnnouncementType string           `json:"announcementType"`
	PostID           string           `json:"postId"`
	PostTitle        string           `json:"postTitle"`
	RevisionID       string           `json:"revisionId"`
	Integration      string           `json:"integration"`
	Type             string           `json:"type"`
	MessageID        string           `json:"messageId"`
	Removed          bool             `json:"removed"`
	State            string           `json:"state"`
	SettledReason    string           `json:"settledReason"`
	Run              int              `json:"run"`
	Tries            int              `json:"tries"`
	OccurredAt       time.Time        `json:"occurredAt"`
	DueAt            time.Time        `json:"dueAt"`
	Last             *announcementTry `json:"last"`
}

type attemptList struct {
	Attempts []postAttempt `json:"attempts"`
}

type tryList struct {
	Tries []announcementTry `json:"tries"`
}

type arrived struct {
	Method  string
	Path    string
	Headers http.Header
	Body    []byte
}

type receiver struct {
	server *httptest.Server
	answer func(arrived) (int, string)
	extra  http.Header

	mu  sync.Mutex
	got []arrived
}

func newReceiver(t *testing.T) *receiver {
	t.Helper()
	held := &receiver{}
	held.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			if _, err := r.Body.Read(body); err != nil && err.Error() != "EOF" {
				t.Errorf("read what arrived: %v", err)
			}
		}
		one := arrived{Headers: r.Header.Clone(), Body: body}
		held.mu.Lock()
		held.got = append(held.got, one)
		answer, extra := held.answer, held.extra
		held.mu.Unlock()
		status, said := http.StatusOK, ""
		if answer != nil {
			status, said = answer(one)
		}
		for name, values := range extra {
			w.Header().Set(name, values[0])
		}
		w.WriteHeader(status)
		w.Write([]byte(said))
	}))
	t.Cleanup(held.server.Close)
	return held
}

func (r *receiver) address() string { return r.server.URL + "/publication" }

func (r *receiver) arrivals() []arrived {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]arrived(nil), r.got...)
}

func (r *receiver) forget() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.got = nil
}

func (r *receiver) answers(with func(arrived) (int, string)) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.answer = with
}

func (r *receiver) holds(name, value string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.extra == nil {
		r.extra = http.Header{}
	}
	r.extra.Set(name, value)
}

func echoesTheChallenge(one arrived) (int, string) {
	var sent struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(one.Body, &sent); err != nil {
		return http.StatusBadRequest, ""
	}
	return http.StatusOK, sent.Challenge
}

type throughLoopback struct {
	receiver string
	discord  string
	resolves func(host string) ([]netip.Addr, error)
}

func (t throughLoopback) Check(address string) (string, error) {
	if t.receiver != "" && strings.HasPrefix(address, t.receiver+"/") {
		parsed, err := url.Parse(address)
		if err == nil {
			return parsed.Hostname(), nil
		}
	}
	return dispatch.NewCaller(dispatch.DefaultLimits()).Check(address)
}

func (t throughLoopback) Get(ctx context.Context, address string) (dispatch.Answer, error) {
	return t.send(ctx, http.MethodGet, address, nil, nil)
}

func (t throughLoopback) Request(ctx context.Context, method, address string, body []byte) (dispatch.Answer, error) {
	return t.send(ctx, method, address, nil, body)
}

func (t throughLoopback) Post(
	ctx context.Context,
	address string,
	headers map[string]string,
	body []byte,
) (dispatch.Answer, error) {
	return t.send(ctx, http.MethodPost, address, headers, body)
}

func (t throughLoopback) dialed(address string) string {
	if t.discord == "" {
		return address
	}
	rest, held := strings.CutPrefix(address, "https://discord.com")
	if !held {
		return address
	}
	return t.discord + rest
}

func (t throughLoopback) send(
	ctx context.Context,
	method, address string,
	headers map[string]string,
	body []byte,
) (dispatch.Answer, error) {
	parsed, err := url.Parse(address)
	if err != nil {
		return dispatch.Answer{}, err
	}
	if t.resolves != nil {
		reaching := dispatch.Reaching{
			Resolve: func(context.Context, string) ([]netip.Addr, error) {
				return t.resolves(parsed.Hostname())
			},
		}
		if _, err := reaching.At(ctx, parsed.Hostname(), "443"); err != nil {
			return dispatch.Answer{}, err
		}
	}
	var carried io.Reader
	if body != nil {
		carried = strings.NewReader(string(body))
	}
	request, err := http.NewRequestWithContext(ctx, method, t.dialed(address), carried)
	if err != nil {
		return dispatch.Answer{}, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return dispatch.Answer{}, err
	}
	defer response.Body.Close()
	said, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return dispatch.Answer{}, err
	}
	return dispatch.Answer{
		Status: response.StatusCode, Body: said,
		RetryAfter: dispatch.RetryAfter(response.Header.Get("Retry-After"), time.Now()),
	}, nil
}

type integrationStack struct {
	blogStack
	to      *receiver
	discord *discordServer
	editor  *http.Cookie
}

func newIntegrationStack(t *testing.T) integrationStack {
	t.Helper()
	return newDestinationStackThrough(t, nil)
}

func newDestinationStackThrough(
	t *testing.T,
	resolves func(host string) ([]netip.Addr, error),
) integrationStack {
	t.Helper()
	pool := testdb.Connect(t)
	outbox := &apitest.VerificationOutbox{}
	to := newReceiver(t)
	discord := newDiscordServer(t)
	handlers := apitest.NewServicesWithSends(
		t, pool, 1<<20, outbox, apitest.SendSettings(),
		throughLoopback{receiver: to.server.URL, discord: discord.server.URL, resolves: resolves},
	)
	router := harness.RegisterRouter(t, handlers, api.DefaultDeadlines())
	session := apitest.VerifiedSignUp(t, router, outbox, "admin@example.com", "blog.admin")
	apitest.SetRole(t, pool, "blog.admin", "admin")
	stack := integrationStack{
		blogStack: blogStack{
			router: router, pool: pool, handlers: handlers, outbox: outbox, admin: session,
		},
		to:      to,
		discord: discord,
	}
	stack.editor = stack.contributor(t, "editor@example.com", "illarin.editor").session
	return stack
}

func (s integrationStack) add(
	t *testing.T,
	session *http.Cookie,
	name, address string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/integrations",
		fmt.Sprintf(`{"name":%q,"address":%q}`, name, address),
	), session))
}

func (s integrationStack) added(t *testing.T, name, address string) addedIntegration {
	t.Helper()
	response := s.add(t, s.admin, name, address)
	if response.Code != http.StatusCreated {
		t.Fatalf("add integration status = %d: %s", response.Code, response.Body.String())
	}
	var made addedIntegration
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode integration: %v", err)
	}
	return made
}

func (s integrationStack) verify(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/integrations/"+id+"/verification", "",
	), session))
}

func (s integrationStack) active(t *testing.T, name string) addedIntegration {
	t.Helper()
	s.to.answers(echoesTheChallenge)
	made := s.added(t, name, s.to.address())
	response := s.verify(t, s.admin, made.Integration.ID)
	if response.Code != http.StatusOK {
		t.Fatalf("verify status = %d: %s", response.Code, response.Body.String())
	}
	s.to.answers(nil)
	s.to.forget()
	return made
}

func (s integrationStack) integrations(t *testing.T, session *http.Cookie) integrationList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/integrations", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list integrations status = %d: %s", response.Code, response.Body.String())
	}
	var listed integrationList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode integrations: %v", err)
	}
	return listed
}

func (s integrationStack) postChoices(
	t *testing.T,
	session *http.Cookie,
	id string,
) destinationChoiceList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/blog/posts/"+id+"/integrations", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read post integrations status = %d: %s", response.Code, response.Body.String())
	}
	var listed destinationChoiceList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode post integrations: %v", err)
	}
	return listed
}

func (s integrationStack) attempts(
	t *testing.T,
	session *http.Cookie,
	id string,
) attemptList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/blog/posts/"+id+"/announcement-attempts", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read attempts status = %d: %s", response.Code, response.Body.String())
	}
	var listed attemptList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode attempts: %v", err)
	}
	return listed
}

func (s integrationStack) publishTo(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts/"+id+"/publish", body,
	), session))
}

func (s integrationStack) sendQueued(t *testing.T) int {
	t.Helper()
	return s.sendQueuedAt(t, time.Now())
}

func (s integrationStack) sendQueuedAt(t *testing.T, at time.Time) int {
	t.Helper()
	made, err := s.handlers.Blog.SendDueAttempts(t.Context(), at)
	if err != nil {
		t.Fatalf("send queued attempts: %v", err)
	}
	return made
}

func (s integrationStack) readyPost(t *testing.T) blogPost {
	t.Helper()
	draft := s.illarinDraft(t, s.editor, "Illarin keeps its own writing now")
	return s.saved(t, s.editor, draft.ID, finished(draft, nil))
}

func TestOnlyAnAdminReachesTheBlogsIntegrations(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	member := stack.member(t, "writer@example.com", "outside.writer")

	response := stack.add(t, member, "Their webhook", stack.to.address())

	if response.Code != http.StatusForbidden {
		t.Errorf("add status = %d, want 403", response.Code)
	}
	listing := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/integrations", nil), member,
	))
	if listing.Code != http.StatusForbidden {
		t.Errorf("list status = %d, want 403", listing.Code)
	}
}

func TestAnEndpointOutsideTheAddressPolicyIsRefused(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	for _, address := range []string{
		"http://hooks.example.com/publication",
		"https://hooks.example.com:8443/publication",
		"https://user:pass@hooks.example.com/publication",
		"https://hooks.example.com/publication#part",
		"https://127.0.0.1/publication",
	} {
		response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
			http.MethodPost, "/v1/blog/integrations",
			fmt.Sprintf(`{"name":"Somewhere","address":%q}`, address),
		), stack.admin))
		if response.Code != http.StatusBadRequest {
			t.Errorf("add %s status = %d, want 400", address, response.Code)
		}
	}
}

func TestASigningSecretIsShownOnceAndNeverAgain(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	made := stack.added(t, "Release feed", stack.to.address())

	if !strings.HasPrefix(made.Secret, dispatch.Prefix) {
		t.Errorf("secret = %q, want the %s mark", made.Secret, dispatch.Prefix)
	}
	listed := stack.integrations(t, stack.admin)
	if len(listed.Integrations) != 1 {
		t.Fatalf("listed %d integrations, want 1", len(listed.Integrations))
	}
	shown := listed.Integrations[0]
	body, _ := json.Marshal(listed)
	if strings.Contains(string(body), made.Secret) {
		t.Error("the listing carried the signing secret")
	}
	if strings.Contains(string(body), stack.to.address()) {
		t.Error("the listing carried the endpoint address")
	}
	if !strings.HasPrefix(shown.Address, "https://"+shown.Host) {
		t.Errorf("address = %q, want it masked to the host", shown.Address)
	}
	if shown.State != "unverified" {
		t.Errorf("state = %q, want unverified", shown.State)
	}
}

func TestTheEndpointAndSecretAreSealedInTheDatabase(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)

	made := stack.added(t, "Release feed", stack.to.address())

	var address, secret []byte
	err := stack.pool.QueryRow(context.Background(), `
		select address, signing_secret from blog_integrations where id = $1
	`, made.Integration.ID).Scan(&address, &secret)
	if err != nil {
		t.Fatalf("read the sealed configuration: %v", err)
	}
	if strings.Contains(string(address), stack.to.server.URL) {
		t.Error("the stored address holds the endpoint in the clear")
	}
	if strings.Contains(string(secret), made.Secret) {
		t.Error("the stored secret holds the value in the clear")
	}
}

func TestAnEndpointIsActiveOnlyAfterItReturnsTheChallenge(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.added(t, "Release feed", stack.to.address())

	stack.to.answers(func(arrived) (int, string) { return http.StatusOK, "something else" })
	refused := stack.verify(t, stack.admin, made.Integration.ID)

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
	if stack.integrations(t, stack.admin).Integrations[0].State != "unverified" {
		t.Error("a wrong answer activated the integration")
	}

	stack.to.answers(echoesTheChallenge)
	accepted := stack.verify(t, stack.admin, made.Integration.ID)

	if accepted.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200: %s", accepted.Code, accepted.Body.String())
	}
	if stack.integrations(t, stack.admin).Integrations[0].State != "active" {
		t.Error("the integration did not become active")
	}
}

func TestTheVerificationRequestIsSigned(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	stack.to.answers(echoesTheChallenge)
	made := stack.added(t, "Release feed", stack.to.address())

	if response := stack.verify(t, stack.admin, made.Integration.ID); response.Code != 200 {
		t.Fatalf("verify status = %d: %s", response.Code, response.Body.String())
	}

	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	checkSignature(t, arrivals[0], made.Secret)
}

func TestAPublishedPostReachesTheChosenEndpoint(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[%q],"note":"Read it in ten minutes."}`,
		ready.Version, made.Integration.ID,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	before := stack.to.arrivals()
	if len(before) != 0 {
		t.Fatalf("%d requests were made inside the publish transaction", len(before))
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the worker settled %d attempts, want 1", sent)
	}
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	checkSignature(t, arrivals[0], made.Secret)
	body := string(arrivals[0].Body)
	for _, want := range []string{
		`"type":"blog.post.published.v1"`,
		`"title":"Illarin keeps its own writing now"`,
		`"summary":"What Illarin changed this week."`,
		`"note":"Read it in ten minutes."`,
		`"url":"http://localhost:3000/blog/`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the event does not carry %s: %s", want, body)
		}
	}
	if strings.Contains(body, "Illarin now keeps its own writing.") {
		t.Errorf("the event carries the post body: %s", body)
	}
	sent := stack.attempts(t, stack.editor, ready.ID)
	if len(sent.Attempts) != 1 {
		t.Fatalf("the post shows %d attempts, want 1", len(sent.Attempts))
	}
	if sent.Attempts[0].State != "delivered" {
		t.Errorf("attempt state = %q, want delivered", sent.Attempts[0].State)
	}
	if sent.Attempts[0].Last.Outcome != "delivered" {
		t.Errorf("attempt outcome = %q, want delivered", sent.Attempts[0].Last.Outcome)
	}
}

func TestPublishingChangesToALivePostSendsNothing(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	live := stack.publishedTo(t, stack.readyPost(t), made.Integration.ID, "")
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the first publication settled %d attempts, want 1", sent)
	}

	again := stack.publishTo(t, stack.editor, live.ID, live.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[%q]}`, live.Version, made.Integration.ID,
	))

	if again.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", again.Code, again.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 0 {
		t.Fatalf("publishing changes settled %d attempts, want 0", sent)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
}

func TestTheWebhookIdIsTheDeliveryAndTheBodyIsWhatWasSigned(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.sendQueued(t)

	sent := stack.attempts(t, stack.editor, ready.ID)
	arrivals := stack.to.arrivals()

	if got := arrivals[0].Headers.Get(dispatch.IDHeader); got != sent.Attempts[0].ID {
		t.Errorf("%s = %q, want the attempt id %q", dispatch.IDHeader, got, sent.Attempts[0].ID)
	}
}

func TestQuietPublicationSendsNothing(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, ready.ID, ready.Version,
		fmt.Sprintf(`{"version":%d,"integrationIds":[]}`, ready.Version))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 0 {
		t.Errorf("the worker settled %d attempts, want 0", sent)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 0 {
		t.Errorf("the receiver was sent %d requests, want 0", len(arrivals))
	}
	if listed := stack.attempts(t, stack.editor, ready.ID); len(listed.Attempts) != 0 {
		t.Errorf("the post shows %d attempts, want 0", len(listed.Attempts))
	}
}

func TestPublicationSurvivesAnEndpointThatRefusesEverything(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	stack.to.answers(func(arrived) (int, string) { return http.StatusInternalServerError, "" })
	ready := stack.readyPost(t)

	published := stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.sendQueued(t)

	if published.Status != "published" {
		t.Errorf("status = %q, want published", published.Status)
	}
	reading := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/posts/"+published.Slug, nil,
	))
	if reading.Code != http.StatusOK {
		t.Errorf("the post is not readable: %d", reading.Code)
	}
	sent := stack.attempts(t, stack.editor, ready.ID)
	if sent.Attempts[0].State != "pending" {
		t.Errorf("attempt state = %q, want pending for another attempt", sent.Attempts[0].State)
	}
	if *sent.Attempts[0].Last.Status != http.StatusInternalServerError {
		t.Errorf("attempt status = %d, want 500", *sent.Attempts[0].Last.Status)
	}
}

func TestADeliveryRecordCarriesNoSecretOrAddress(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	stack.to.answers(func(arrived) (int, string) { return http.StatusTeapot, "go away" })
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.sendQueued(t)

	body, err := json.Marshal(stack.attempts(t, stack.editor, ready.ID))

	if err != nil {
		t.Fatalf("encode attempts: %v", err)
	}
	for _, secret := range []string{made.Secret, stack.to.address(), "go away"} {
		if strings.Contains(string(body), secret) {
			t.Errorf("the attempt record carries %q", secret)
		}
	}
}

func TestAWriterSeesSafeIntegrationIdentitiesOnly(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	writer := stack.contributor(t, "writer@example.com", "outside.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Lumiverse 2.0 is out"}`, announcement.ID,
	))

	choices := stack.postChoices(t, writer.session, draft.ID)

	if len(choices.Integrations) != 1 {
		t.Fatalf("the writer sees %d integrations, want 1", len(choices.Integrations))
	}
	if choices.Integrations[0].Name != "Release feed" {
		t.Errorf("name = %q, want the safe identity", choices.Integrations[0].Name)
	}
	if !choices.Integrations[0].ByDefault {
		t.Error("an integration is not chosen by default")
	}
	body, _ := json.Marshal(choices)
	for _, hidden := range []string{made.Secret, stack.to.address(), stack.to.server.URL} {
		if strings.Contains(string(body), hidden) {
			t.Errorf("the writer was shown %q", hidden)
		}
	}
	listing := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/blog/integrations", nil), writer.session,
	))
	if listing.Code != http.StatusForbidden {
		t.Errorf("the writer read the configuration: %d", listing.Code)
	}
}

func TestAWriterCannotSendToAnIntegrationTheBlogDoesNotHave(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	stack.active(t, "Release feed")
	writer := stack.contributor(t, "writer@example.com", "outside.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Lumiverse 2.0 is out"}`, announcement.ID,
	))
	ready := stack.saved(t, writer.session, draft.ID, finished(draft, nil))

	response := stack.publishTo(t, writer.session, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[%q]}`, ready.Version, "00000000-0000-0000-0000-000000000001",
	))

	if response.Code != http.StatusForbidden {
		t.Errorf("publish status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestADisabledEndpointStopsReceiving(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete,
		"/v1/blog/integrations/"+made.Integration.ID+"/verification", nil,
	), stack.admin))
	if response.Code != http.StatusOK {
		t.Fatalf("disable status = %d: %s", response.Code, response.Body.String())
	}
	ready := stack.readyPost(t)

	refused := stack.publishTo(t, stack.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[%q]}`, ready.Version, made.Integration.ID,
	))

	if refused.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", refused.Code, refused.Body.String())
	}
	if listed := stack.attempts(t, stack.editor, ready.ID); len(listed.Attempts) != 0 {
		t.Errorf("a disabled integration was queued %d attempts", len(listed.Attempts))
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 0 {
		t.Errorf("a disabled integration was sent %d requests", len(arrivals))
	}
}

func TestANewAddressTakesTheEndpointBackToUnverified(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPatch, "/v1/blog/integrations/"+made.Integration.ID,
		`{"address":"https://hooks.example.com/elsewhere"}`,
	), stack.admin))

	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", response.Code, response.Body.String())
	}
	shown := stack.integrations(t, stack.admin).Integrations[0]
	if shown.State != "unverified" {
		t.Errorf("state = %q, want unverified", shown.State)
	}
	if shown.Host != "hooks.example.com" {
		t.Errorf("host = %q, want the new one", shown.Host)
	}
}

func TestTheNoteBelongsToTheTransitionAndNotToThePost(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	published := stack.publishedTo(t, ready, made.Integration.ID, "Read it in ten minutes.")

	reading := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/posts/"+published.Slug, nil,
	))
	if strings.Contains(reading.Body.String(), "Read it in ten minutes.") {
		t.Error("the note became part of the post")
	}
	var note string
	err := stack.pool.QueryRow(context.Background(), `
		select note from blog_announcements where post_id = $1
	`, ready.ID).Scan(&note)
	if err != nil {
		t.Fatalf("read the event note: %v", err)
	}
	if note != "Read it in ten minutes." {
		t.Errorf("note = %q, want the one the transition captured", note)
	}
}

func (s integrationStack) publishedTo(
	t *testing.T,
	ready blogPost,
	integrationID, note string,
) blogPost {
	t.Helper()
	response := s.publishTo(t, s.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"integrationIds":[%q],"note":%q}`, ready.Version, integrationID, note,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	return decodePost(t, response)
}

func checkSignature(t *testing.T, one arrived, secret string) {
	t.Helper()
	id := one.Headers.Get(dispatch.IDHeader)
	stamp := one.Headers.Get(dispatch.TimestampHeader)
	if id == "" || stamp == "" {
		t.Fatalf("the request carries no %s or %s", dispatch.IDHeader, dispatch.TimestampHeader)
	}
	seconds, err := time.Parse(time.RFC3339, "1970-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	var unix int64
	if _, err := fmt.Sscan(stamp, &unix); err != nil {
		t.Fatalf("%s = %q is not a unix time", dispatch.TimestampHeader, stamp)
	}
	want, err := dispatch.Sign(secret, id, seconds.Add(time.Duration(unix)*time.Second), one.Body)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	if got := one.Headers.Get(dispatch.SignatureHeader); got != want {
		t.Errorf("%s = %q, want %q", dispatch.SignatureHeader, got, want)
	}
}

func TestAScheduledPublicationKeepsTheChoiceItWasGiven(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	at := time.Now().Add(time.Hour)

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/posts/"+ready.ID+"/schedule", fmt.Sprintf(
			`{"version":%d,"at":%q,"integrationIds":[%q],"note":"Out at noon."}`,
			ready.Version, at.Format(time.RFC3339), made.Integration.ID,
		),
	), stack.editor))

	if response.Code != http.StatusCreated {
		t.Fatalf("schedule status = %d: %s", response.Code, response.Body.String())
	}
	settled, err := stack.handlers.Blog.PublishDueSchedules(t.Context(), at.Add(time.Minute))
	if err != nil {
		t.Fatalf("publish due schedules: %v", err)
	}
	if settled != 1 {
		t.Fatalf("the scheduler settled %d editions, want 1", settled)
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the worker settled %d attempts, want 1", sent)
	}
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	if !strings.Contains(string(arrivals[0].Body), `"note":"Out at noon."`) {
		t.Errorf("the event lost the note the schedule captured: %s", arrivals[0].Body)
	}
}

func TestPublishingWithNoChoiceAnnouncesToEveryIntegration(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	stack.active(t, "Release feed")
	writer := stack.contributor(t, "writer@example.com", "outside.writer")
	announcement := stack.categoryBySlug(t, "announcement")
	draft := stack.started(t, writer.session, fmt.Sprintf(
		`{"categoryId":%q,"title":"Lumiverse 2.0 is out"}`, announcement.ID,
	))
	ready := stack.saved(t, writer.session, draft.ID, finished(draft, nil))

	response := stack.publishTo(t, writer.session, ready.ID, ready.Version,
		fmt.Sprintf(`{"version":%d}`, ready.Version))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Errorf("the worker settled %d attempts, want one per integration", sent)
	}
}

func TestAContributorNeverSeesAnotherPostsAnnouncementAttempts(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	outsider := stack.member(t, "writer@example.com", "outside.writer")

	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/blog/posts/"+ready.ID+"/announcement-attempts", nil,
	), outsider))

	if response.Code != http.StatusForbidden {
		t.Errorf("read attempts status = %d, want 403", response.Code)
	}
}
