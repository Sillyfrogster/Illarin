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

type destination struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Name        string     `json:"name"`
	Host        string     `json:"host"`
	Address     string     `json:"address"`
	State       string     `json:"state"`
	Events      []string   `json:"events"`
	Channel     *channel   `json:"channel"`
	SecretSetAt time.Time  `json:"secretSetAt"`
	OldUntil    *time.Time `json:"previousSecretUntil"`
	VerifiedAt  *string    `json:"verifiedAt"`
	DisabledAt  *string    `json:"disabledAt"`
}

type channel struct {
	GuildID   string `json:"guildId"`
	ChannelID string `json:"channelId"`
	Webhook   string `json:"webhookName"`
	RoleID    string `json:"roleId"`
	RoleName  string `json:"roleName"`
}

type addedDestination struct {
	Destination destination `json:"destination"`
	Secret      string      `json:"secret"`
}

type destinationList struct {
	Destinations []destination `json:"destinations"`
}

type destinationChoice struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	State     string   `json:"state"`
	Events    []string `json:"events"`
	Role      string   `json:"role"`
	ByDefault bool     `json:"byDefault"`
}

type destinationChoiceList struct {
	Destinations []destinationChoice `json:"destinations"`
	Inherited    bool                `json:"inherited"`
}

type deliveryAttempt struct {
	Run         int       `json:"run"`
	Number      int       `json:"number"`
	Outcome     string    `json:"outcome"`
	Status      *int      `json:"status"`
	Detail      string    `json:"detail"`
	TookMs      int       `json:"tookMs"`
	AttemptedAt time.Time `json:"attemptedAt"`
}

type postDelivery struct {
	ID            string           `json:"id"`
	EventID       string           `json:"eventId"`
	EventType     string           `json:"eventType"`
	PostID        string           `json:"postId"`
	PostTitle     string           `json:"postTitle"`
	RevisionID    string           `json:"revisionId"`
	Destination   string           `json:"destination"`
	Type          string           `json:"type"`
	MessageID     string           `json:"messageId"`
	Removed       bool             `json:"removed"`
	State         string           `json:"state"`
	SettledReason string           `json:"settledReason"`
	Run           int              `json:"run"`
	Attempts      int              `json:"attempts"`
	OccurredAt    time.Time        `json:"occurredAt"`
	DueAt         time.Time        `json:"dueAt"`
	Last          *deliveryAttempt `json:"last"`
}

type deliveryList struct {
	Deliveries []postDelivery `json:"deliveries"`
}

type attemptList struct {
	Attempts []deliveryAttempt `json:"attempts"`
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

type destinationStack struct {
	publicationStack
	to      *receiver
	discord *discordServer
	editor  *http.Cookie
}

func newDestinationStack(t *testing.T) destinationStack {
	t.Helper()
	return newDestinationStackThrough(t, nil)
}

func newDestinationStackThrough(
	t *testing.T,
	resolves func(host string) ([]netip.Addr, error),
) destinationStack {
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
	session := apitest.VerifiedSignUp(t, router, outbox, "authority@example.com", "publication.authority")
	apitest.HoldsAuthority(t, pool, "publication.authority")
	stack := destinationStack{
		publicationStack: publicationStack{
			router: router, pool: pool, handlers: handlers, outbox: outbox, authority: session,
		},
		to:      to,
		discord: discord,
	}
	stack.editor = stack.admin(t, "editor@example.com", "illarin.editor")
	return stack
}

func (s destinationStack) add(
	t *testing.T,
	session *http.Cookie,
	name, address string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/destinations",
		fmt.Sprintf(`{"name":%q,"address":%q}`, name, address),
	), session))
}

func (s destinationStack) added(t *testing.T, name, address string) addedDestination {
	t.Helper()
	response := s.add(t, s.authority, name, address)
	if response.Code != http.StatusCreated {
		t.Fatalf("add destination status = %d: %s", response.Code, response.Body.String())
	}
	var made addedDestination
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode destination: %v", err)
	}
	return made
}

func (s destinationStack) verify(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/destinations/"+id+"/verification", "",
	), session))
}

func (s destinationStack) active(t *testing.T, name string) addedDestination {
	t.Helper()
	s.to.answers(echoesTheChallenge)
	made := s.added(t, name, s.to.address())
	response := s.verify(t, s.authority, made.Destination.ID)
	if response.Code != http.StatusOK {
		t.Fatalf("verify status = %d: %s", response.Code, response.Body.String())
	}
	s.to.answers(nil)
	s.to.forget()
	return made
}

func (s destinationStack) destinations(t *testing.T, session *http.Cookie) destinationList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/destinations", nil), session,
	))
	if response.Code != http.StatusOK {
		t.Fatalf("list destinations status = %d: %s", response.Code, response.Body.String())
	}
	var listed destinationList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode destinations: %v", err)
	}
	return listed
}

func (s destinationStack) allowOnApp(t *testing.T, appID, destinationID string) {
	t.Helper()
	_, err := s.pool.Exec(context.Background(), `
		insert into publication_app_destinations (app_id, destination_id, by_default)
		values ($1, $2, true)
	`, appID, destinationID)
	if err != nil {
		t.Fatalf("allow on app: %v", err)
	}
}

func (s destinationStack) postChoices(
	t *testing.T,
	session *http.Cookie,
	id string,
) destinationChoiceList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+id+"/destinations", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read post destinations status = %d: %s", response.Code, response.Body.String())
	}
	var listed destinationChoiceList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode post destinations: %v", err)
	}
	return listed
}

func (s destinationStack) deliveries(
	t *testing.T,
	session *http.Cookie,
	id string,
) deliveryList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+id+"/deliveries", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read deliveries status = %d: %s", response.Code, response.Body.String())
	}
	var listed deliveryList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode deliveries: %v", err)
	}
	return listed
}

func (s destinationStack) publishTo(
	t *testing.T,
	session *http.Cookie,
	id string,
	version int,
	body string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+id+"/publish", body,
	), session))
}

func (s destinationStack) sendQueued(t *testing.T) int {
	t.Helper()
	return s.sendQueuedAt(t, time.Now())
}

func (s destinationStack) sendQueuedAt(t *testing.T, at time.Time) int {
	t.Helper()
	made, err := s.handlers.Publications.SendDueDeliveries(t.Context(), at)
	if err != nil {
		t.Fatalf("send queued deliveries: %v", err)
	}
	return made
}

func (s destinationStack) readyPost(t *testing.T) blogPost {
	t.Helper()
	draft := s.illarinDraft(t, s.editor, "Illarin keeps its own writing now")
	return s.saved(t, s.editor, draft.ID, finished(draft, nil))
}

func TestOnlyThePublicationAuthorityReachesDestinations(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	member := stack.member(t, "writer@example.com", "outside.writer")

	response := stack.add(t, member, "Their webhook", stack.to.address())

	if response.Code != http.StatusForbidden {
		t.Errorf("add status = %d, want 403", response.Code)
	}
	listing := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/destinations", nil), member,
	))
	if listing.Code != http.StatusForbidden {
		t.Errorf("list status = %d, want 403", listing.Code)
	}
}

func TestAnEndpointOutsideTheAddressPolicyIsRefused(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)

	for _, address := range []string{
		"http://hooks.example.com/publication",
		"https://hooks.example.com:8443/publication",
		"https://user:pass@hooks.example.com/publication",
		"https://hooks.example.com/publication#part",
		"https://127.0.0.1/publication",
	} {
		response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
			http.MethodPost, "/v1/publication/destinations",
			fmt.Sprintf(`{"name":"Somewhere","address":%q}`, address),
		), stack.authority))
		if response.Code != http.StatusBadRequest {
			t.Errorf("add %s status = %d, want 400", address, response.Code)
		}
	}
}

func TestASigningSecretIsShownOnceAndNeverAgain(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)

	made := stack.added(t, "Release feed", stack.to.address())

	if !strings.HasPrefix(made.Secret, dispatch.Prefix) {
		t.Errorf("secret = %q, want the %s mark", made.Secret, dispatch.Prefix)
	}
	listed := stack.destinations(t, stack.authority)
	if len(listed.Destinations) != 1 {
		t.Fatalf("listed %d destinations, want 1", len(listed.Destinations))
	}
	shown := listed.Destinations[0]
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
	stack := newDestinationStack(t)

	made := stack.added(t, "Release feed", stack.to.address())

	var address, secret []byte
	err := stack.pool.QueryRow(context.Background(), `
		select address, signing_secret from publication_destinations where id = $1
	`, made.Destination.ID).Scan(&address, &secret)
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
	stack := newDestinationStack(t)
	made := stack.added(t, "Release feed", stack.to.address())

	stack.to.answers(func(arrived) (int, string) { return http.StatusOK, "something else" })
	refused := stack.verify(t, stack.authority, made.Destination.ID)

	if refused.Code != http.StatusBadRequest {
		t.Fatalf("verify status = %d, want 400: %s", refused.Code, refused.Body.String())
	}
	if stack.destinations(t, stack.authority).Destinations[0].State != "unverified" {
		t.Error("a wrong answer activated the destination")
	}

	stack.to.answers(echoesTheChallenge)
	accepted := stack.verify(t, stack.authority, made.Destination.ID)

	if accepted.Code != http.StatusOK {
		t.Fatalf("verify status = %d, want 200: %s", accepted.Code, accepted.Body.String())
	}
	if stack.destinations(t, stack.authority).Destinations[0].State != "active" {
		t.Error("the destination did not become active")
	}
}

func TestTheVerificationRequestIsSigned(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	stack.to.answers(echoesTheChallenge)
	made := stack.added(t, "Release feed", stack.to.address())

	if response := stack.verify(t, stack.authority, made.Destination.ID); response.Code != 200 {
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
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"destinationIds":[%q],"note":"Read it in ten minutes."}`,
		ready.Version, made.Destination.ID,
	))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	before := stack.to.arrivals()
	if len(before) != 0 {
		t.Fatalf("%d requests were made inside the publish transaction", len(before))
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the worker settled %d deliveries, want 1", sent)
	}
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	checkSignature(t, arrivals[0], made.Secret)
	body := string(arrivals[0].Body)
	for _, want := range []string{
		`"type":"publication.post.published.v1"`,
		`"title":"Illarin keeps its own writing now"`,
		`"summary":"What Illarin changed this week."`,
		`"note":"Read it in ten minutes."`,
		`"url":"http://blog.localhost:3000/`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("the event does not carry %s: %s", want, body)
		}
	}
	if strings.Contains(body, "Illarin now keeps its own writing.") {
		t.Errorf("the event carries the post body: %s", body)
	}
	sent := stack.deliveries(t, stack.editor, ready.ID)
	if len(sent.Deliveries) != 1 {
		t.Fatalf("the post shows %d deliveries, want 1", len(sent.Deliveries))
	}
	if sent.Deliveries[0].State != "delivered" {
		t.Errorf("delivery state = %q, want delivered", sent.Deliveries[0].State)
	}
	if sent.Deliveries[0].Last.Outcome != "delivered" {
		t.Errorf("attempt outcome = %q, want delivered", sent.Deliveries[0].Last.Outcome)
	}
}

func TestPublishingChangesToALivePostSendsNothing(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	live := stack.publishedTo(t, stack.readyPost(t), made.Destination.ID, "")
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the first publication settled %d deliveries, want 1", sent)
	}

	again := stack.publishTo(t, stack.editor, live.ID, live.Version, fmt.Sprintf(
		`{"version":%d,"destinationIds":[%q]}`, live.Version, made.Destination.ID,
	))

	if again.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", again.Code, again.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 0 {
		t.Fatalf("publishing changes settled %d deliveries, want 0", sent)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
}

func TestTheWebhookIdIsTheDeliveryAndTheBodyIsWhatWasSigned(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Destination.ID, "")
	stack.sendQueued(t)

	sent := stack.deliveries(t, stack.editor, ready.ID)
	arrivals := stack.to.arrivals()

	if got := arrivals[0].Headers.Get(dispatch.IDHeader); got != sent.Deliveries[0].ID {
		t.Errorf("%s = %q, want the delivery id %q", dispatch.IDHeader, got, sent.Deliveries[0].ID)
	}
}

func TestQuietPublicationSendsNothing(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	response := stack.publishTo(t, stack.editor, ready.ID, ready.Version,
		fmt.Sprintf(`{"version":%d,"destinationIds":[]}`, ready.Version))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 0 {
		t.Errorf("the worker settled %d deliveries, want 0", sent)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 0 {
		t.Errorf("the receiver was sent %d requests, want 0", len(arrivals))
	}
	if listed := stack.deliveries(t, stack.editor, ready.ID); len(listed.Deliveries) != 0 {
		t.Errorf("the post shows %d deliveries, want 0", len(listed.Deliveries))
	}
}

func TestPublicationSurvivesAnEndpointThatRefusesEverything(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	stack.to.answers(func(arrived) (int, string) { return http.StatusInternalServerError, "" })
	ready := stack.readyPost(t)

	published := stack.publishedTo(t, ready, made.Destination.ID, "")
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
	sent := stack.deliveries(t, stack.editor, ready.ID)
	if sent.Deliveries[0].State != "pending" {
		t.Errorf("delivery state = %q, want pending for another attempt", sent.Deliveries[0].State)
	}
	if *sent.Deliveries[0].Last.Status != http.StatusInternalServerError {
		t.Errorf("attempt status = %d, want 500", *sent.Deliveries[0].Last.Status)
	}
}

func TestADeliveryRecordCarriesNoSecretOrAddress(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	stack.to.answers(func(arrived) (int, string) { return http.StatusTeapot, "go away" })
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Destination.ID, "")
	stack.sendQueued(t)

	body, err := json.Marshal(stack.deliveries(t, stack.editor, ready.ID))

	if err != nil {
		t.Fatalf("encode deliveries: %v", err)
	}
	for _, secret := range []string{made.Secret, stack.to.address(), "go away"} {
		if strings.Contains(string(body), secret) {
			t.Errorf("the delivery record carries %q", secret)
		}
	}
}

func TestAContributorSeesSafeDestinationIdentitiesOnly(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	writer := stack.member(t, "writer@example.com", "outside.writer")
	app := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	grant := stack.approved(t, "outside.writer", app.ID, []string{announcement.ID}, announcement.ID)
	stack.allowOnApp(t, app.ID, made.Destination.ID)
	draft := stack.started(t, writer, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Lumiverse 2.0 is out"}`,
		grant.ID, announcement.ID,
	))

	choices := stack.postChoices(t, writer, draft.ID)

	if len(choices.Destinations) != 1 {
		t.Fatalf("the contributor sees %d destinations, want 1", len(choices.Destinations))
	}
	if choices.Destinations[0].Name != "Release feed" {
		t.Errorf("name = %q, want the safe identity", choices.Destinations[0].Name)
	}
	if !choices.Destinations[0].ByDefault {
		t.Error("the app default did not reach the contributor")
	}
	body, _ := json.Marshal(choices)
	for _, hidden := range []string{made.Secret, stack.to.address(), stack.to.server.URL} {
		if strings.Contains(string(body), hidden) {
			t.Errorf("the contributor was shown %q", hidden)
		}
	}
	listing := apitest.Send(t, stack.router, apitest.Authorized(
		httptest.NewRequest(http.MethodGet, "/v1/publication/destinations", nil), writer,
	))
	if listing.Code != http.StatusForbidden {
		t.Errorf("the contributor read the configuration: %d", listing.Code)
	}
}

func TestAContributorCannotSendOutsideWhatTheGrantAllows(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	writer := stack.member(t, "writer@example.com", "outside.writer")
	app := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	grant := stack.approved(t, "outside.writer", app.ID, []string{announcement.ID}, announcement.ID)
	draft := stack.started(t, writer, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Lumiverse 2.0 is out"}`, grant.ID, announcement.ID,
	))
	ready := stack.saved(t, writer, draft.ID, finished(draft, nil))

	response := stack.publishTo(t, writer, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"destinationIds":[%q]}`, ready.Version, made.Destination.ID,
	))

	if response.Code != http.StatusForbidden {
		t.Errorf("publish status = %d, want 403: %s", response.Code, response.Body.String())
	}
}

func TestADisabledEndpointStopsReceiving(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodDelete,
		"/v1/publication/destinations/"+made.Destination.ID+"/verification", nil,
	), stack.authority))
	if response.Code != http.StatusOK {
		t.Fatalf("disable status = %d: %s", response.Code, response.Body.String())
	}
	ready := stack.readyPost(t)

	refused := stack.publishTo(t, stack.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"destinationIds":[%q]}`, ready.Version, made.Destination.ID,
	))

	if refused.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", refused.Code, refused.Body.String())
	}
	if listed := stack.deliveries(t, stack.editor, ready.ID); len(listed.Deliveries) != 0 {
		t.Errorf("a disabled destination was queued %d deliveries", len(listed.Deliveries))
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 0 {
		t.Errorf("a disabled destination was sent %d requests", len(arrivals))
	}
}

func TestANewAddressTakesTheEndpointBackToUnverified(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPatch, "/v1/publication/destinations/"+made.Destination.ID,
		`{"address":"https://hooks.example.com/elsewhere"}`,
	), stack.authority))

	if response.Code != http.StatusOK {
		t.Fatalf("update status = %d: %s", response.Code, response.Body.String())
	}
	shown := stack.destinations(t, stack.authority).Destinations[0]
	if shown.State != "unverified" {
		t.Errorf("state = %q, want unverified", shown.State)
	}
	if shown.Host != "hooks.example.com" {
		t.Errorf("host = %q, want the new one", shown.Host)
	}
}

func TestTheNoteBelongsToTheTransitionAndNotToThePost(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)

	published := stack.publishedTo(t, ready, made.Destination.ID, "Read it in ten minutes.")

	reading := apitest.Send(t, stack.router, httptest.NewRequest(
		http.MethodGet, "/v1/posts/"+published.Slug, nil,
	))
	if strings.Contains(reading.Body.String(), "Read it in ten minutes.") {
		t.Error("the note became part of the post")
	}
	var note string
	err := stack.pool.QueryRow(context.Background(), `
		select note from publication_events where post_id = $1
	`, ready.ID).Scan(&note)
	if err != nil {
		t.Fatalf("read the event note: %v", err)
	}
	if note != "Read it in ten minutes." {
		t.Errorf("note = %q, want the one the transition captured", note)
	}
}

func (s destinationStack) publishedTo(
	t *testing.T,
	ready blogPost,
	destinationID, note string,
) blogPost {
	t.Helper()
	response := s.publishTo(t, s.editor, ready.ID, ready.Version, fmt.Sprintf(
		`{"version":%d,"destinationIds":[%q],"note":%q}`, ready.Version, destinationID, note,
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
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	at := time.Now().Add(time.Hour)

	response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/publication/posts/"+ready.ID+"/schedule", fmt.Sprintf(
			`{"version":%d,"at":%q,"destinationIds":[%q],"note":"Out at noon."}`,
			ready.Version, at.Format(time.RFC3339), made.Destination.ID,
		),
	), stack.editor))

	if response.Code != http.StatusCreated {
		t.Fatalf("schedule status = %d: %s", response.Code, response.Body.String())
	}
	settled, err := stack.handlers.Publications.PublishDueSchedules(t.Context(), at.Add(time.Minute))
	if err != nil {
		t.Fatalf("publish due schedules: %v", err)
	}
	if settled != 1 {
		t.Fatalf("the scheduler settled %d editions, want 1", settled)
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Fatalf("the worker settled %d deliveries, want 1", sent)
	}
	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	if !strings.Contains(string(arrivals[0].Body), `"note":"Out at noon."`) {
		t.Errorf("the event lost the note the schedule captured: %s", arrivals[0].Body)
	}
}

func TestAPublicationWithNoChoiceTakesTheDefaults(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	writer := stack.member(t, "writer@example.com", "outside.writer")
	app := stack.configureApp(t, "lumiverse", "Lumiverse", "https://lumiverse.example")
	announcement := stack.categoryBySlug(t, "announcement")
	grant := stack.approved(t, "outside.writer", app.ID, []string{announcement.ID}, announcement.ID)
	stack.allowOnApp(t, app.ID, made.Destination.ID)
	draft := stack.started(t, writer, fmt.Sprintf(
		`{"grantId":%q,"categoryId":%q,"title":"Lumiverse 2.0 is out"}`, grant.ID, announcement.ID,
	))
	ready := stack.saved(t, writer, draft.ID, finished(draft, nil))

	response := stack.publishTo(t, writer, ready.ID, ready.Version,
		fmt.Sprintf(`{"version":%d}`, ready.Version))

	if response.Code != http.StatusOK {
		t.Fatalf("publish status = %d: %s", response.Code, response.Body.String())
	}
	if sent := stack.sendQueued(t); sent != 1 {
		t.Errorf("the worker settled %d deliveries, want the app default", sent)
	}
}

func TestAContributorNeverSeesAnotherPostsDeliveries(t *testing.T) {
	t.Parallel()
	stack := newDestinationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Destination.ID, "")
	outsider := stack.member(t, "writer@example.com", "outside.writer")

	response := apitest.Send(t, stack.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/publication/posts/"+ready.ID+"/deliveries", nil,
	), outsider))

	if response.Code != http.StatusForbidden {
		t.Errorf("read deliveries status = %d, want 403", response.Code)
	}
}
