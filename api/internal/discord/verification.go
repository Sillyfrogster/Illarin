package discord

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/Sillyfrogster/Illarin/api/internal/outbound"
)

type CapabilityReader interface {
	Get(context.Context, string) (outbound.Answer, error)
}

func VerifyCapability(ctx context.Context, reader CapabilityReader, address string) (Capability, Webhook, error) {
	capability, err := ReadCapability(address)
	if err != nil {
		return Capability{}, Webhook{}, err
	}
	answer, err := reader.Get(ctx, capability.URL)
	if err != nil {
		return Capability{}, Webhook{}, errors.New("Illarin could not reach Discord.")
	}
	if answer.Status != http.StatusOK {
		return Capability{}, Webhook{}, errors.New(whyRefused(answer.Status))
	}
	found, err := ReadWebhook(answer.Body)
	if err != nil || found.ID != capability.ID {
		return Capability{}, Webhook{}, ErrNotAWebhook
	}
	return capability, found, nil
}

func whyRefused(status int) string {
	switch status {
	case http.StatusNotFound:
		return "Discord does not recognize that webhook. Check it still exists and that the whole address was copied."
	case http.StatusUnauthorized, http.StatusForbidden:
		return "Discord turned that address away. Its token is no longer good."
	default:
		return fmt.Sprintf("Discord answered %d for that webhook.", status)
	}
}
