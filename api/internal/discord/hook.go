package discord

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// Webhooks is the one prefix a Discord capability may start with.
const Webhooks = "https://discord.com/api/webhooks/"

// snowflake is the shape of every Discord id, which is the only shape Illarin
// will treat as one.
var snowflake = regexp.MustCompile(`^[0-9]{17,20}$`)

// tokenShape is what a webhook token looks like, and all Illarin checks about
// it before Discord itself is asked.
var tokenShape = regexp.MustCompile(`^[A-Za-z0-9_-]{16,120}$`)

// ErrNotACapability says an address is not a Discord incoming webhook.
var ErrNotACapability = errors.New(
	"a Discord destination is an incoming webhook address on discord.com",
)

// ErrNotAWebhook says Discord answered with something other than a webhook
// bound to a guild channel.
var ErrNotAWebhook = errors.New("Discord did not answer with a channel webhook")

// Capability is one Discord incoming webhook address, split into the parts a
// request needs. The token in it is the whole credential, so it is held under
// the same seal as any other endpoint secret.
type Capability struct {
	ID    string
	Token string
	URL   string
}

// ReadCapability answers the capability behind a supplied address, or why it
// was refused.
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

// Confirming is the address that makes Discord answer with the message it
// created, so a send is either confirmed or known to be unconfirmed.
func (c Capability) Confirming() string { return c.URL + "?wait=true" }

// Webhook is the safe identity Discord answers with when asked about a
// capability. It names where announcements land and nothing that would let a
// reader send one.
type Webhook struct {
	ID        string
	GuildID   string
	ChannelID string
	Name      string
}

// ReadWebhook answers the identity behind what Discord said about a capability.
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

// MessageID answers the id of the message Discord says it created, and an
// empty string when the answer proves nothing.
func MessageID(body []byte) string {
	var said struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(body, &said); err != nil {
		return ""
	}
	if !snowflake.MatchString(said.ID) {
		return ""
	}
	return said.ID
}

// CheckRole refuses anything that is not a Discord role id.
func CheckRole(raw string) error {
	if !snowflake.MatchString(strings.TrimSpace(raw)) {
		return fmt.Errorf("a Discord role id is 17 to 20 digits")
	}
	return nil
}
