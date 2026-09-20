package integration_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/google/uuid"
)

func TestDiscordRepairsNeedAnAdminAndDoNotRepeatOrRewriteHistory(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	channel := stack.channelWithRole(t, "Readers")
	post, attempts := stack.announcedPost(t, fmt.Sprintf(`"integrationIds":[%q]`, channel.ID))
	id := attempts[0].ID
	path := "/v1/blog/announcement-attempts/" + id + "/repair"
	var version int
	if err := stack.pool.QueryRow(t.Context(), `select working_version from posts where id = $1`, post.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	for index, action := range []string{"edit", "delete", "correction"} {
		body := fmt.Sprintf(`{"requestId":%q,"action":%q,"messageId":%q,"text":"A corrected note @everyone"}`, uuid.NewString(), action, discordMessageID)
		refused := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t, http.MethodPost, path, body), stack.editor))
		if refused.Code != http.StatusForbidden {
			t.Fatalf("contributor repair = %d", refused.Code)
		}
		for repeat := 0; repeat < 2; repeat++ {
			response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t, http.MethodPost, path, body), stack.admin))
			if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"state":"completed"`) {
				t.Fatalf("%s = %d: %s", action, response.Code, response.Body.String())
			}
		}
		arrivals := stack.discord.announcements()
		if len(arrivals) != index+2 {
			t.Fatalf("%s was repeated: %d calls", action, len(arrivals))
		}
		last := arrivals[len(arrivals)-1]
		method := map[string]string{"edit": http.MethodPatch, "delete": http.MethodDelete, "correction": http.MethodPost}[action]
		if last.Method != method {
			t.Fatalf("%s used %s", action, last.Method)
		}
		if action != "correction" && !strings.HasSuffix(last.Path, "/messages/"+discordMessageID) {
			t.Fatal("repair did not name the original message")
		}
		if action != "delete" {
			var message announcement
			if err := json.Unmarshal(last.Body, &message); err != nil {
				t.Fatal(err)
			}
			if len(message.Mentions.Parse)+len(message.Mentions.Roles)+len(message.Mentions.Users) != 0 {
				t.Fatal("repair enabled mentions")
			}
		}
	}
	stack.discord.answersSendWith(func(arrived) (int, string) { return 0, "" })
	ambiguous := fmt.Sprintf(`{"requestId":%q,"action":"correction","text":"A later correction"}`, uuid.NewString())
	for repeat := 0; repeat < 2; repeat++ {
		response := apitest.Send(t, stack.router, apitest.Authorized(jsonRequest(t, http.MethodPost, path, ambiguous), stack.admin))
		if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"state":"unconfirmed"`) {
			t.Fatalf("lost repair response = %d: %s", response.Code, response.Body.String())
		}
	}
	if len(stack.discord.announcements()) != 5 {
		t.Fatal("an unconfirmed correction was sent twice")
	}
	var after, announced, tries, audits int
	err := stack.pool.QueryRow(t.Context(), `select working_version,
		(select count(*) from blog_announcements where post_id = $1),
		(select count(*) from blog_announcement_tries where attempt_id = $2),
		(select count(*) from blog_activity_log where attempt_id = $2 and action like 'discord.%')
		from posts where id = $1`, post.ID, id).Scan(&after, &announced, &tries, &audits)
	if err != nil {
		t.Fatal(err)
	}
	if version != after || announced != 1 || tries != 1 || audits != 4 {
		t.Fatalf("repair changed history: version %d/%d, announced %d, tries %d, audits %d", version, after, announced, tries, audits)
	}
}

func TestDiscordAmbiguityStopsRetriesAndAMissingWebhookIsDisabled(t *testing.T) {
	t.Parallel()
	t.Run("abandoned worker", func(t *testing.T) {
		stack := newIntegrationStack(t)
		channel := stack.channelWithRole(t, "")
		post := stack.readyPost(t)
		response := stack.publishTo(t, stack.editor, post.ID, post.Version, choosing(post, fmt.Sprintf(`"integrationIds":[%q]`, channel.ID)))
		if response.Code != http.StatusOK {
			t.Fatal(response.Body.String())
		}
		_, err := stack.pool.Exec(t.Context(), `update blog_announcement_attempts set state = 'sending', tries = 1,
			lease_token = $1, lease_expires_at = now() - interval '1 minute'`, uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		stack.sendQueued(t)
		found := stack.attempts(t, stack.editor, post.ID).Attempts
		if found[0].State != "unconfirmed" || len(stack.discord.announcements()) != 0 {
			t.Fatal("an abandoned send was repeated")
		}
	})
	for _, status := range []int{0, http.StatusBadGateway, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			stack := newIntegrationStack(t)
			channel := stack.channelWithRole(t, "")
			stack.discord.answersSendWith(func(arrived) (int, string) { return status, `{}` })
			_, attempts := stack.announcedPost(t, fmt.Sprintf(`"integrationIds":[%q]`, channel.ID))
			want := "unconfirmed"
			if status == http.StatusNotFound {
				want = "failed"
			}
			if attempts[0].State != want {
				t.Fatalf("attempt = %s", attempts[0].State)
			}
			if stack.sendQueuedAt(t, time.Now().Add(48*time.Hour)) != 0 || len(stack.discord.announcements()) != 1 {
				t.Fatal("Discord was retried automatically")
			}
			if status == http.StatusNotFound && stack.integrations(t, stack.admin).Integrations[0].State != "disabled" {
				t.Fatal("missing webhook remained active")
			}
		})
	}
}
