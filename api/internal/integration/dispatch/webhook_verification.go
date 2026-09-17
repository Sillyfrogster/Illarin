package dispatch

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

	"github.com/google/uuid"
)

const SecretOverlap = 24 * time.Hour

var ErrNotProven = errors.New("the endpoint did not return the challenge")

type VerificationSender interface {
	Post(context.Context, string, map[string]string, []byte) (Answer, error)
}

func VerifyEndpoint(ctx context.Context, sender VerificationSender, event, address string, secrets []string, at time.Time) error {
	random := make([]byte, 24)
	if _, err := rand.Read(random); err != nil {
		return errors.New("Could not create a verification challenge.")
	}
	challenge := base64.RawURLEncoding.EncodeToString(random)
	id := uuid.New().String()
	body, err := json.Marshal(struct {
		ID        string    `json:"id"`
		Type      string    `json:"type"`
		Challenge string    `json:"challenge"`
		SentAt    time.Time `json:"sentAt"`
	}{id, event, challenge, at})
	if err != nil {
		return err
	}
	headers, err := Headers(secrets, id, at, body)
	if err != nil {
		return errors.New("Could not sign the verification challenge.")
	}
	answer, err := sender.Post(ctx, address, headers, body)
	if err != nil {
		return errors.New("Illarin could not reach that endpoint.")
	}
	if answer.Status < http.StatusOK || answer.Status >= http.StatusMultipleChoices {
		return fmt.Errorf("The endpoint answered %d instead of the challenge.", answer.Status)
	}
	if !proves(answer.Body, challenge) {
		return ErrNotProven
	}
	return nil
}

func proves(reply []byte, challenge string) bool {
	if len(reply) > 1<<10 {
		return false
	}
	if strings.TrimSpace(string(reply)) == challenge {
		return true
	}
	var wrapped struct {
		Challenge string `json:"challenge"`
	}
	return json.Unmarshal(reply, &wrapped) == nil && wrapped.Challenge == challenge
}
