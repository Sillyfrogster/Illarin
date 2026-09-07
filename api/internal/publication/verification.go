package publication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Sillyfrogster/Illarin/api/internal/webhook"
	"github.com/google/uuid"
)

// EventVerification is the one event a destination receives before it is
// allowed to receive anything real.
const EventVerification = "publication.endpoint.verification.v1"

// challengeBytes is the length of the value an endpoint has to hand back.
const challengeBytes = 24

// maxChallengeReply is the most of a reply Illarin reads while looking for the
// challenge in it.
const maxChallengeReply = 1 << 10

// ErrNotProven says an endpoint did not return the challenge it was sent.
var ErrNotProven = errors.New("the endpoint did not return the challenge")

// verification is the whole body sent to prove an endpoint is under the
// control of whoever configured it.
type verification struct {
	ID        uuid.UUID `json:"id"`
	Type      string    `json:"type"`
	Challenge string    `json:"challenge"`
	SentAt    time.Time `json:"sentAt"`
}

// VerifyDestination sends a signed challenge and activates the destination
// only when the exact value comes back.
func (s *Service) VerifyDestination(
	ctx context.Context,
	actor uuid.UUID,
	id uuid.UUID,
) (Destination, error) {
	current, err := s.Destination(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	if current.Kind == KindDiscord {
		return s.provenByDiscord(ctx, actor, id)
	}
	address, secrets, err := s.endpointOf(ctx, id)
	if err != nil {
		return Destination{}, err
	}
	challenge, err := newChallenge()
	if err != nil {
		return Destination{}, err
	}
	sent := s.now().UTC()
	body, err := json.Marshal(verification{
		ID: uuid.New(), Type: EventVerification, Challenge: challenge, SentAt: sent,
	})
	if err != nil {
		return Destination{}, fmt.Errorf("write the verification event: %w", err)
	}
	headers, err := webhook.Headers(secrets, uuid.New().String(), sent, body)
	if err != nil {
		return Destination{}, err
	}
	answer, err := s.sender.Post(ctx, address, headers, body)
	if err != nil {
		return Destination{}, FieldError{
			Field: "address", Message: "Illarin could not reach that endpoint.", cause: err,
		}
	}
	if answer.Status < http.StatusOK || answer.Status >= http.StatusMultipleChoices {
		return Destination{}, FieldError{
			Field:   "address",
			Message: fmt.Sprintf("The endpoint answered %d instead of the challenge.", answer.Status),
			cause:   ErrNotProven,
		}
	}
	if !proves(answer.Body, challenge) {
		return Destination{}, FieldError{
			Field:   "address",
			Message: "The endpoint did not return the challenge it was sent.",
			cause:   ErrNotProven,
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Destination{}, fmt.Errorf("begin destination verification: %w", err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
		update publication_destinations
		   set state = $2, verified_at = now(), disabled_at = null, updated_at = now()
		 where id = $1
	`, id, DestinationActive)
	if err != nil {
		return Destination{}, fmt.Errorf("activate the destination: %w", err)
	}
	err = recordPublicationAudit(ctx, tx, change{
		Actor: actor, Action: "destination.verified", DestinationID: &id,
	})
	if err != nil {
		return Destination{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Destination{}, fmt.Errorf("commit destination verification: %w", err)
	}
	return s.Destination(ctx, id)
}

// proves answers whether a bounded reply is the challenge, whether the endpoint
// echoed it plainly or wrapped it in the object the documentation describes.
func proves(reply []byte, challenge string) bool {
	if len(reply) > maxChallengeReply {
		return false
	}
	if strings.TrimSpace(string(reply)) == challenge {
		return true
	}
	var wrapped struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(reply, &wrapped); err != nil {
		return false
	}
	return wrapped.Challenge == challenge
}

func newChallenge() (string, error) {
	body := make([]byte, challengeBytes)
	if _, err := rand.Read(body); err != nil {
		return "", fmt.Errorf("make a challenge: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}
