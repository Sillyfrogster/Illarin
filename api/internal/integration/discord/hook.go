package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

const Webhooks = "https://discord.com/api/webhooks/"

var snowflake = regexp.MustCompile(`^[0-9]{17,20}$`)

var tokenShape = regexp.MustCompile(`^[A-Za-z0-9_-]{16,120}$`)

var ErrNotACapability = errors.New(
	"a Discord integration is an incoming webhook address on discord.com",
)

var ErrNotAWebhook = errors.New("Discord did not answer with a channel webhook")

type Capability struct {
	ID    string
	Token string
	URL   string
}

func ReadCapability(raw string) (Capability, error) {
	address := strings.TrimSpace(raw)
	rest, held := strings.CutPrefix(address, Webhooks)
	if !held {
		return Capability{}, ErrNotACapability
	}
	id, token, split := strings.Cut(rest, "/")
	if !split || !snowflake.MatchString(id) || !tokenShape.MatchString(token) {
		return Capability{}, ErrNotACapability
	}
	return Capability{ID: id, Token: token, URL: address}, nil
}

func (c Capability) Confirming() string { return c.URL + "?wait=true" }

type Webhook struct {
	ID        string
	GuildID   string
	ChannelID string
	Name      string
}

func ReadWebhook(body []byte) (Webhook, error) {
	var said struct {
		ID        string `json:"id"`
		GuildID   string `json:"guild_id"`
		ChannelID string `json:"channel_id"`
		Name      string `json:"name"`
	}
	if err := json.Unmarshal(body, &said); err != nil {
		return Webhook{}, ErrNotAWebhook
	}
	if !snowflake.MatchString(said.ID) ||
		!snowflake.MatchString(said.GuildID) ||
		!snowflake.MatchString(said.ChannelID) {
		return Webhook{}, ErrNotAWebhook
	}
	return Webhook{
		ID: said.ID, GuildID: said.GuildID, ChannelID: said.ChannelID, Name: said.Name,
	}, nil
}

func CheckRole(raw string) error {
	if !snowflake.MatchString(strings.TrimSpace(raw)) {
		return fmt.Errorf("a Discord role id is 17 to 20 digits")
	}
	return nil
}
