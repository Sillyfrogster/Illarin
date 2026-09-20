package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/apitest"
	"github.com/Sillyfrogster/Illarin/api/internal/blog"
	"github.com/Sillyfrogster/Illarin/api/internal/integration/dispatch"
)

func (s integrationStack) activeFor(t *testing.T, name string, events []string) addedIntegration {
	t.Helper()
	s.to.answers(echoesTheChallenge)
	body, err := json.Marshal(map[string]any{
		"name": name, "address": s.to.address(), "events": events,
	})
	if err != nil {
		t.Fatalf("encode integration: %v", err)
	}
	response := apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/integrations", string(body),
	), s.admin))
	if response.Code != http.StatusCreated {
		t.Fatalf("add integration status = %d: %s", response.Code, response.Body.String())
	}
	var made addedIntegration
	if err := json.Unmarshal(response.Body.Bytes(), &made); err != nil {
		t.Fatalf("decode integration: %v", err)
	}
	if proven := s.verify(t, s.admin, made.Integration.ID); proven.Code != http.StatusOK {
		t.Fatalf("verify status = %d: %s", proven.Code, proven.Body.String())
	}
	s.to.answers(nil)
	s.to.forget()
	return made
}

func (s integrationStack) answersWith(status int) {
	s.to.answers(func(arrived) (int, string) { return status, "" })
}

func (s integrationStack) rotate(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/integrations/"+id+"/secret", "",
	), session))
}

func (s integrationStack) rotated(t *testing.T, id string) struct {
	Integration integrationRow `json:"integration"`
	Secret      string         `json:"secret"`
	OldUntil    time.Time      `json:"previousSecretUntil"`
} {
	t.Helper()
	var shown struct {
		Integration integrationRow `json:"integration"`
		Secret      string         `json:"secret"`
		OldUntil    time.Time      `json:"previousSecretUntil"`
	}
	response := s.rotate(t, s.admin, id)
	if response.Code != http.StatusOK {
		t.Fatalf("rotate status = %d: %s", response.Code, response.Body.String())
	}
	if err := json.Unmarshal(response.Body.Bytes(), &shown); err != nil {
		t.Fatalf("decode rotated secret: %v", err)
	}
	return shown
}

func (s integrationStack) allDeliveries(
	t *testing.T,
	session *http.Cookie,
	query string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/blog/announcement-attempts"+query, nil,
	), session))
}

func (s integrationStack) diagnosed(t *testing.T, query string) attemptList {
	t.Helper()
	response := s.allDeliveries(t, s.admin, query)
	if response.Code != http.StatusOK {
		t.Fatalf("list attempts status = %d: %s", response.Code, response.Body.String())
	}
	var listed attemptList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode attempts: %v", err)
	}
	return listed
}

func (s integrationStack) replay(
	t *testing.T,
	session *http.Cookie,
	id string,
) *httptest.ResponseRecorder {
	t.Helper()
	return apitest.Send(t, s.router, apitest.Authorized(jsonRequest(t,
		http.MethodPost, "/v1/blog/announcement-attempts/"+id+"/replay", "",
	), session))
}

func (s integrationStack) tries(t *testing.T, session *http.Cookie, id string) tryList {
	t.Helper()
	response := apitest.Send(t, s.router, apitest.Authorized(httptest.NewRequest(
		http.MethodGet, "/v1/blog/announcement-attempts/"+id+"/tries", nil,
	), session))
	if response.Code != http.StatusOK {
		t.Fatalf("read attempts status = %d: %s", response.Code, response.Body.String())
	}
	var listed tryList
	if err := json.Unmarshal(response.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode attempts: %v", err)
	}
	return listed
}

func (s integrationStack) onlyAttempt(t *testing.T, postID string) postAttempt {
	t.Helper()
	listed := s.attempts(t, s.editor, postID)
	if len(listed.Attempts) != 1 {
		t.Fatalf("the post shows %d attempts, want 1", len(listed.Attempts))
	}
	return listed.Attempts[0]
}

func TestADeliveryWaitsTheAgreedGapBeforeEachAttempt(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)

	at := time.Now().UTC()
	for place, gap := range blog.TryDelays[1:] {
		if attempted := stack.sendQueuedAt(t, at); attempted != 1 {
			t.Fatalf("attempt %d made %d requests, want 1", place+1, attempted)
		}
		waiting := stack.onlyAttempt(t, ready.ID)
		if waiting.State != "pending" {
			t.Fatalf("after attempt %d the attempt is %q, want pending", place+1, waiting.State)
		}
		waited := waiting.DueAt.Sub(at)
		if waited < gap*9/10 || waited > gap*11/10 {
			t.Fatalf("attempt %d waits %s, want about %s", place+2, waited, gap)
		}
		if early := stack.sendQueuedAt(t, at.Add(gap*8/10)); early != 0 {
			t.Fatalf("attempt %d ran %d requests before it was due", place+2, early)
		}
		at = waiting.DueAt
	}

	if last := stack.sendQueuedAt(t, at); last != 1 {
		t.Fatalf("the last attempt made %d requests, want 1", last)
	}

	spent := stack.onlyAttempt(t, ready.ID)
	if spent.State != "failed" || spent.SettledReason != "exhausted" {
		t.Errorf("the run ended %q/%q, want failed/exhausted", spent.State, spent.SettledReason)
	}
	if spent.Tries != blog.MaxTries {
		t.Errorf("the run made %d attempts, want %d", spent.Tries, blog.MaxTries)
	}
	if again := stack.sendQueuedAt(t, at.Add(365*24*time.Hour)); again != 0 {
		t.Errorf("a spent run made %d further requests", again)
	}
}

func TestWhatAnEndpointAnswersDecidesWhetherIllarinTriesAgain(t *testing.T) {
	t.Parallel()
	for _, one := range []struct {
		status int
		state  string
		reason string
	}{
		{http.StatusOK, "delivered", "arrived"},
		{http.StatusNoContent, "delivered", "arrived"},
		{http.StatusMovedPermanently, "failed", "refused"},
		{http.StatusFound, "failed", "refused"},
		{http.StatusBadRequest, "failed", "refused"},
		{http.StatusNotFound, "failed", "refused"},
		{http.StatusRequestTimeout, "pending", ""},
		{http.StatusTooEarly, "pending", ""},
		{http.StatusTooManyRequests, "pending", ""},
		{http.StatusInternalServerError, "pending", ""},
		{http.StatusBadGateway, "pending", ""},
	} {
		t.Run(fmt.Sprint(one.status), func(t *testing.T) {
			stack := newIntegrationStack(t)
			made := stack.active(t, "Release feed")
			ready := stack.readyPost(t)
			stack.publishedTo(t, ready, made.Integration.ID, "")
			stack.answersWith(one.status)

			stack.sendQueued(t)

			settled := stack.onlyAttempt(t, ready.ID)
			if settled.State != one.state {
				t.Errorf("%d left the attempt %q, want %q", one.status, settled.State, one.state)
			}
			if settled.SettledReason != one.reason {
				t.Errorf("%d settled as %q, want %q", one.status, settled.SettledReason, one.reason)
			}
		})
	}
}

func TestAnEndpointAnsweringGoneReceivesNothingFurther(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	first := stack.readyPost(t)
	second := stack.readyPost(t)
	stack.publishedTo(t, first, made.Integration.ID, "")
	stack.publishedTo(t, second, made.Integration.ID, "")
	stack.answersWith(http.StatusGone)

	stack.sendQueued(t)

	gone := stack.onlyAttempt(t, first.ID)
	if gone.State != "failed" || gone.SettledReason != "gone" {
		t.Errorf("the attempt ended %q/%q, want failed/gone", gone.State, gone.SettledReason)
	}
	stopped := stack.onlyAttempt(t, second.ID)
	if stopped.State != "failed" || stopped.SettledReason != "disabled" {
		t.Errorf("the waiting work ended %q/%q, want failed/disabled",
			stopped.State, stopped.SettledReason)
	}
	if shown := stack.integrations(t, stack.admin).Integrations[0]; shown.State != "disabled" {
		t.Errorf("the integration is %q, want disabled", shown.State)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 1 {
		t.Errorf("the receiver was sent %d requests, want 1", len(arrivals))
	}
}

func TestAnEndpointAskingIllarinToWaitIsWaitedFor(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.to.answers(func(arrived) (int, string) { return http.StatusTooManyRequests, "" })
	held := stack.handlers.Blog

	at := time.Now().UTC()
	if _, err := held.SendDueAttempts(t.Context(), at); err != nil {
		t.Fatalf("send queued attempts: %v", err)
	}

	asked := stack.onlyAttempt(t, ready.ID)
	if asked.State != "pending" {
		t.Fatalf("the attempt is %q, want pending", asked.State)
	}
	if asked.Last.Detail != "The integration asked Illarin to retry later." {
		t.Errorf("attempt detail = %q", asked.Last.Detail)
	}
}

func TestARetryAfterHeaderIsHonoredOverTheAgreedGap(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.to.answers(func(arrived) (int, string) { return http.StatusTooManyRequests, "" })
	stack.to.holds("Retry-After", "600")

	at := time.Now().UTC()
	stack.sendQueuedAt(t, at)

	asked := stack.onlyAttempt(t, ready.ID)
	if waited := asked.DueAt.Sub(at); waited < 9*time.Minute {
		t.Errorf("the next attempt waits %s, want the 10 minutes it asked for", waited)
	}
}

func TestOneDeliveryKeepsItsWebhookIdAndSignsEachAttemptAfresh(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)

	at := time.Now().UTC()
	stack.sendQueuedAt(t, at)
	stack.sendQueuedAt(t, stack.onlyAttempt(t, ready.ID).DueAt)

	arrivals := stack.to.arrivals()
	if len(arrivals) != 2 {
		t.Fatalf("the receiver was sent %d requests, want 2", len(arrivals))
	}
	work := stack.onlyAttempt(t, ready.ID)
	for _, one := range arrivals {
		if got := one.Headers.Get(dispatch.IDHeader); got != work.ID {
			t.Errorf("%s = %q, want the attempt id %q", dispatch.IDHeader, got, work.ID)
		}
		checkSignature(t, one, made.Secret)
	}
	first, second := arrivals[0].Headers, arrivals[1].Headers
	if first.Get(dispatch.TimestampHeader) == second.Get(dispatch.TimestampHeader) {
		t.Error("the second attempt reused the first attempt's timestamp")
	}
	if first.Get(dispatch.SignatureHeader) == second.Get(dispatch.SignatureHeader) {
		t.Error("the second attempt reused the first attempt's signature")
	}
	if string(arrivals[0].Body) != string(arrivals[1].Body) {
		t.Error("the two attempts carried different bodies")
	}
}

func TestARotatedSecretSignsUnderBothUntilTheOverlapEnds(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")

	turned := stack.rotated(t, made.Integration.ID)

	if turned.Secret == made.Secret {
		t.Fatal("the rotation handed back the same secret")
	}
	if !turned.OldUntil.After(time.Now()) {
		t.Errorf("the overlap ends at %s, which is not ahead", turned.OldUntil)
	}
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.sendQueued(t)

	arrivals := stack.to.arrivals()
	if len(arrivals) != 1 {
		t.Fatalf("the receiver was sent %d requests, want 1", len(arrivals))
	}
	header := arrivals[0].Headers.Get(dispatch.SignatureHeader)
	id := arrivals[0].Headers.Get(dispatch.IDHeader)
	sent := stampOf(t, arrivals[0])
	for name, secret := range map[string]string{"new": turned.Secret, "old": made.Secret} {
		if !dispatch.Accepts(header, secret, id, sent, arrivals[0].Body) {
			t.Errorf("the request carries no signature the %s secret accepts", name)
		}
	}

	forgotten, err := stack.handlers.Blog.ForgetOldSecrets(
		t.Context(), time.Now().Add(blog.SecretOverlap+time.Hour),
	)
	if err != nil {
		t.Fatalf("forget the rotated secrets: %v", err)
	}
	if forgotten != 1 {
		t.Fatalf("the cleanup forgot %d secrets, want 1", forgotten)
	}
	stack.to.forget()
	after := stack.readyPost(t)
	stack.publishedTo(t, after, made.Integration.ID, "")
	stack.sendQueued(t)

	later := stack.to.arrivals()
	if len(later) != 1 {
		t.Fatalf("the receiver was sent %d requests after the overlap, want 1", len(later))
	}
	header = later[0].Headers.Get(dispatch.SignatureHeader)
	id = later[0].Headers.Get(dispatch.IDHeader)
	sent = stampOf(t, later[0])
	if !dispatch.Accepts(header, turned.Secret, id, sent, later[0].Body) {
		t.Error("the request carries no signature the new secret accepts")
	}
	if dispatch.Accepts(header, made.Secret, id, sent, later[0].Body) {
		t.Error("the old secret still produces an accepted signature")
	}
	if shown := stack.integrations(t, stack.admin).Integrations[0]; shown.OldUntil != nil {
		t.Error("the integration still names an overlap")
	}
}

func TestOnlyAnAdminRotatesASigningSecret(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")

	response := stack.rotate(t, stack.editor, made.Integration.ID)

	if response.Code != http.StatusForbidden {
		t.Errorf("rotate status = %d, want 403", response.Code)
	}
}

func TestAHostThatLeavesPublicSpaceIsRefusedOnTheNextAttempt(t *testing.T) {
	t.Parallel()
	public := []netip.Addr{netip.MustParseAddr("93.184.216.34")}
	private := []netip.Addr{netip.MustParseAddr("10.0.0.9")}
	resolved := public
	stack := newDestinationStackThrough(t, func(string) ([]netip.Addr, error) {
		return resolved, nil
	})
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	resolved = private

	stack.sendQueued(t)

	refused := stack.onlyAttempt(t, ready.ID)
	if refused.Last.Outcome != "unreachable" {
		t.Errorf("attempt outcome = %q, want unreachable", refused.Last.Outcome)
	}
	if arrivals := stack.to.arrivals(); len(arrivals) != 0 {
		t.Errorf("the receiver was sent %d requests, want 0", len(arrivals))
	}

	resolved = public
	stack.sendQueuedAt(t, refused.DueAt)

	if arrivals := stack.to.arrivals(); len(arrivals) != 1 {
		t.Errorf("the receiver was sent %d requests once the host came back", len(arrivals))
	}
}

func TestAnExhaustedDeliveryReplaysWithoutErasingWhatItTried(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)
	spendTheRun(t, stack, ready.ID)
	spent := stack.onlyAttempt(t, ready.ID)

	stack.answersWith(http.StatusOK)
	response := stack.replay(t, stack.admin, spent.ID)

	if response.Code != http.StatusOK {
		t.Fatalf("replay status = %d: %s", response.Code, response.Body.String())
	}
	queued := stack.onlyAttempt(t, ready.ID)
	if queued.State != "pending" {
		t.Errorf("the replayed attempt is %q, want pending", queued.State)
	}
	if queued.Run != spent.Run+1 {
		t.Errorf("the replay opened run %d, want %d", queued.Run, spent.Run+1)
	}
	if queued.AnnouncementID != spent.AnnouncementID {
		t.Error("the replay changed the event the attempt carries")
	}
	stack.sendQueued(t)

	arrived := stack.onlyAttempt(t, ready.ID)
	if arrived.State != "delivered" || arrived.SettledReason != "arrived" {
		t.Errorf("the replay ended %q/%q, want delivered/arrived",
			arrived.State, arrived.SettledReason)
	}
	made1 := stack.tries(t, stack.admin, spent.ID)
	if len(made1.Tries) != blog.MaxTries+1 {
		t.Fatalf("the attempt shows %d attempts, want %d",
			len(made1.Tries), blog.MaxTries+1)
	}
	last := made1.Tries[len(made1.Tries)-1]
	if last.Run != spent.Run+1 || last.Outcome != "delivered" {
		t.Errorf("the last attempt is run %d and %q, want run %d delivered",
			last.Run, last.Outcome, spent.Run+1)
	}
	if made1.Tries[0].Run != spent.Run {
		t.Errorf("the first attempt moved to run %d", made1.Tries[0].Run)
	}
}

func TestAContributorReadsItsOwnDeliveriesAndReplaysNothing(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)
	spendTheRun(t, stack, ready.ID)
	spent := stack.onlyAttempt(t, ready.ID)

	if response := stack.replay(t, stack.editor, spent.ID); response.Code != http.StatusForbidden {
		t.Errorf("replay status = %d, want 403", response.Code)
	}
	if listing := stack.allDeliveries(t, stack.editor, ""); listing.Code != http.StatusForbidden {
		t.Errorf("list status = %d, want 403", listing.Code)
	}
	own := stack.attempts(t, stack.editor, ready.ID)
	if len(own.Attempts) != 1 {
		t.Fatalf("the contributor sees %d of its own attempts, want 1", len(own.Attempts))
	}
	body, _ := json.Marshal(own)
	for _, hidden := range []string{made.Secret, stack.to.address()} {
		if strings.Contains(string(body), hidden) {
			t.Errorf("the contributor was shown %q", hidden)
		}
	}
}

func TestAnUnsettledDeliveryIsNotReplayed(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)
	stack.sendQueued(t)
	waiting := stack.onlyAttempt(t, ready.ID)

	response := stack.replay(t, stack.admin, waiting.ID)

	if response.Code != http.StatusBadRequest {
		t.Errorf("replay status = %d, want 400: %s", response.Code, response.Body.String())
	}
}

func TestExhaustedWorkStaysVisibleToTheAdmin(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	stack.answersWith(http.StatusServiceUnavailable)
	spendTheRun(t, stack, ready.ID)

	listed := stack.diagnosed(t, "?state=failed")

	if len(listed.Attempts) != 1 {
		t.Fatalf("the admin sees %d exhausted attempts, want 1", len(listed.Attempts))
	}
	shown := listed.Attempts[0]
	if shown.SettledReason != "exhausted" {
		t.Errorf("settled reason = %q, want exhausted", shown.SettledReason)
	}
	if shown.PostTitle != "Illarin keeps its own writing now" {
		t.Errorf("post title = %q, want the one the edition carries", shown.PostTitle)
	}
	if len(stack.diagnosed(t, "?state=delivered").Attempts) != 0 {
		t.Error("the delivered listing carried exhausted work")
	}
	body, _ := json.Marshal(listed)
	for _, hidden := range []string{made.Secret, stack.to.address()} {
		if strings.Contains(string(body), hidden) {
			t.Errorf("the listing carried %q", hidden)
		}
	}
}

func TestAnInterruptedAttemptIsTakenOverWithoutSpendingItsPlace(t *testing.T) {
	t.Parallel()
	stack := newIntegrationStack(t)
	made := stack.active(t, "Release feed")
	ready := stack.readyPost(t)
	stack.publishedTo(t, ready, made.Integration.ID, "")
	work := stack.onlyAttempt(t, ready.ID)
	stack.answersWith(http.StatusServiceUnavailable)

	interrupt(t, stack, work.ID)
	at := time.Now().UTC()
	if taken := stack.sendQueuedAt(t, at); taken != 1 {
		t.Fatalf("the worker took over %d attempts, want 1", taken)
	}

	waiting := stack.onlyAttempt(t, ready.ID)
	if waiting.State != "pending" {
		t.Fatalf("the attempt is %q, want pending", waiting.State)
	}
	if waiting.Tries != 2 {
		t.Errorf("the attempt counts %d attempts, want 2", waiting.Tries)
	}
	if waiting.Last.Number != 2 {
		t.Errorf("the recorded attempt is number %d, want 2", waiting.Last.Number)
	}
	gap := blog.TryDelays[1]
	if waited := waiting.DueAt.Sub(at); waited < gap*9/10 || waited > gap*11/10 {
		t.Errorf("the next attempt waits %s, want the first gap of %s", waited, gap)
	}
}

func interrupt(t *testing.T, stack integrationStack, attemptID string) {
	t.Helper()
	_, err := stack.pool.Exec(context.Background(), `
		update blog_announcement_attempts
		   set state = 'sending', tries = tries + 1,
		       lease_token = gen_random_uuid(), lease_expires_at = now() - interval '1 minute'
		 where id = $1
	`, attemptID)
	if err != nil {
		t.Fatalf("interrupt the attempt: %v", err)
	}
}

func spendTheRun(t *testing.T, stack integrationStack, postID string) {
	t.Helper()
	at := time.Now().UTC()
	for range blog.MaxTries {
		if made := stack.sendQueuedAt(t, at); made != 1 {
			t.Fatalf("an attempt made %d requests, want 1", made)
		}
		at = stack.onlyAttempt(t, postID).DueAt
	}
}

func stampOf(t *testing.T, one arrived) time.Time {
	t.Helper()
	var unix int64
	if _, err := fmt.Sscan(one.Headers.Get(dispatch.TimestampHeader), &unix); err != nil {
		t.Fatalf("%s is not a unix time: %v", dispatch.TimestampHeader, err)
	}
	return time.Unix(unix, 0).UTC()
}
