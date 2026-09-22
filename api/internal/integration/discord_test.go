package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/Sillyfrogster/Illarin/api/internal/api"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/apitest/full"
	"github.com/Sillyfrogster/Illarin/api/internal/integration"
	"github.com/Sillyfrogster/Illarin/api/internal/work"
	"github.com/jackc/pgx/v5/pgxpool"
)

var harness = full.Harness

const hook = "https://discord.com/api/webhooks/1234567890123456789/a-long-webhook-token"

func TestACreatorConnectsReadsAndClearsTheirDiscordChannel(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	router, _ := harness.NewRouterWithSenderAndPool(t, 1<<20, api.DefaultDeadlines(), outbox)
	creator := apitest.VerifiedSignUp(t, router, outbox, "creator@example.com", "discord.creator")
	other := apitest.VerifiedSignUp(t, router, outbox, "other@example.com", "discord.other")
	send := func(session *http.Cookie, method, body string) (int, string) {
		response := apitest.Send(t, router, apitest.AuthorizedJSONRequest(t, method, "/v1/account/discord-channel", body, session))
		return response.Code, response.Body.String()
	}

	if code, body := send(creator, http.MethodPut, `{"address":"https://example.com/api/webhooks/1/token"}`); code != http.StatusBadRequest {
		t.Fatalf("a non-Discord address = %d %s, want 400", code, body)
	}
	if code, body := send(creator, http.MethodPut, `{"address":"`+hook+`"}`); code != http.StatusOK {
		t.Fatalf("connect = %d %s", code, body)
	}
	_, read := send(creator, http.MethodGet, "")
	if !strings.Contains(read, `"connected":true`) || strings.Contains(read, "webhook-token") {
		t.Fatalf("the creator reads %s, want connected and no address", read)
	}
	if _, body := send(other, http.MethodGet, ""); !strings.Contains(body, `"connected":false`) {
		t.Errorf("another account reads %s", body)
	}
	if code, _ := send(creator, http.MethodDelete, ""); code != http.StatusOK {
		t.Fatalf("disconnect = %d", code)
	}
	if _, body := send(creator, http.MethodGet, ""); !strings.Contains(body, `"connected":false`) {
		t.Errorf("after disconnecting the creator reads %s", body)
	}
}

func TestOnlyAnAdminSetsTheBlogsDiscordChannel(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	router, pool := harness.NewRouterWithSenderAndPool(t, 1<<20, api.DefaultDeadlines(), outbox)
	member := apitest.VerifiedSignUp(t, router, outbox, "member@example.com", "blog.member")
	admin := apitest.VerifiedSignUp(t, router, outbox, "admin@example.com", "blog.admin")
	apitest.SetRole(t, pool, "blog.admin", "admin")

	for session, want := range map[*http.Cookie]int{member: http.StatusForbidden, admin: http.StatusOK} {
		response := apitest.Send(t, router, apitest.AuthorizedJSONRequest(
			t, http.MethodPut, "/v1/blog/discord-channel", `{"address":"`+hook+`"}`, session))
		if response.Code != want {
			t.Errorf("set the blog's channel = %d, want %d", response.Code, want)
		}
	}
	var owners int
	if err := pool.QueryRow(context.Background(),
		`select count(*) from discord_webhooks where owner_id is null`).Scan(&owners); err != nil || owners != 1 {
		t.Fatalf("blog channels = %d, error = %v", owners, err)
	}
}

func TestAVersionOfAPublicWorkIsQueuedAndAnUnlistedOneIsNot(t *testing.T) {
	t.Parallel()
	outbox := &apitest.VerificationOutbox{}
	router, pool := harness.NewRouterWithSenderAndPool(t, 1<<20, api.DefaultDeadlines(), outbox)
	creator := apitest.VerifiedSignUp(t, router, outbox, "creator@example.com", apitest.CreatorHandle)
	workID := apitest.PublishedCharacter(t, router, creator)
	send := func(method, path, body string) {
		t.Helper()
		if got := apitest.Send(t, router, apitest.AuthorizedJSONRequest(t, method, path, body, creator)); got.Code >= 300 {
			t.Fatalf("%s %s = %d: %s", method, path, got.Code, got.Body.String())
		}
	}
	publish := func(summary, choice string) {
		t.Helper()
		started := apitest.FetchStartedWork(t, router, creator, workID)
		coreBlock := apitest.BlockNamed(t, started.Blocks, "character_core")
		core := apitest.EditableBlock(coreBlock)
		core.Elements[0].Content = json.RawMessage(fmt.Sprintf(`{"text":%q}`, summary))
		if got := apitest.SaveBlock(t, router, creator, workID, coreBlock.ID, core); got.Code != http.StatusOK {
			t.Fatalf("save = %d: %s", got.Code, got.Body.String())
		}
		body := fmt.Sprintf(`{"summary":%q%s}`, summary, choice)
		if got := apitest.PublishWorkVersion(t, router, creator, workID, body); got.Code != http.StatusOK {
			t.Fatalf("publish %q = %d: %s", summary, got.Code, got.Body.String())
		}
	}

	send(http.MethodPut, "/v1/account/discord-channel", `{"address":"`+hook+`"}`)
	publish("Posted", "")
	publish("Left out", `,"discord":false`)
	send(http.MethodPut, "/v1/works/"+workID+"/visibility", `{"visibility":"unlisted"}`)
	publish("Unlisted", `,"discord":true`)

	var bodies []string
	rows, err := pool.Query(context.Background(), `select body::text from discord_posts`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var body string
		if err := rows.Scan(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
	}
	if len(bodies) != 1 || !strings.Contains(bodies[0], "Posted") {
		t.Fatalf("queued posts = %v, want the one public version asked for", bodies)
	}
}

type fakeDiscord struct {
	mu       sync.Mutex
	statuses []int
	bodies   []string
}

func (f *fakeDiscord) RoundTrip(request *http.Request) (*http.Response, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(request.Body)
	f.bodies = append(f.bodies, request.URL.String()+" "+string(body))
	status := f.statuses[0]
	if len(f.statuses) > 1 {
		f.statuses = f.statuses[1:]
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(""))}, nil
}

func queued(t *testing.T, svc *work.Service, pool *pgxpool.Pool, integrations *integration.Service) {
	t.Helper()
	ctx := context.Background()
	owner := apitest.Owner(t, svc, "sender.owner")
	if err := integrations.Connect(ctx, &owner, hook); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `
		insert into discord_posts (webhook_id, body)
		select id, '{"embeds":[]}' from discord_webhooks where owner_id = $1
	`, owner); err != nil {
		t.Fatal(err)
	}
}

func state(t *testing.T, pool *pgxpool.Pool) (sent, failed bool, tries int) {
	t.Helper()
	if err := pool.QueryRow(context.Background(), `
		select sent_at is not null, failed_at is not null, tries from discord_posts
	`).Scan(&sent, &failed, &tries); err != nil {
		t.Fatal(err)
	}
	return sent, failed, tries
}

func TestTheSenderPostsRetriesARecoverableRefusalAndFailsADeletedWebhook(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name     string
		statuses []int
		sent     bool
		failed   bool
	}{
		{name: "accepted", statuses: []int{http.StatusNoContent}, sent: true},
		{name: "rate limited", statuses: []int{http.StatusTooManyRequests}},
		{name: "webhook deleted", statuses: []int{http.StatusNotFound}, failed: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			svc, pool := apitest.Works(t)
			discord := &fakeDiscord{statuses: test.statuses}
			integrations := apitest.NewIntegrations(pool, &http.Client{Transport: discord})
			queued(t, svc, pool, integrations)

			tried, err := integrations.SendDue(context.Background())
			if err != nil || tried != 1 {
				t.Fatalf("tried %d, error = %v", tried, err)
			}
			sent, failed, tries := state(t, pool)
			if sent != test.sent || failed != test.failed || tries != 1 {
				t.Errorf("sent %v, failed %v, tries %d", sent, failed, tries)
			}
			if len(discord.bodies) != 1 || !strings.HasPrefix(discord.bodies[0], hook+" ") {
				t.Errorf("Discord received %v", discord.bodies)
			}
			if again, _ := integrations.SendDue(context.Background()); again != 0 {
				t.Errorf("a settled or waiting post was tried again at once")
			}
		})
	}
}
