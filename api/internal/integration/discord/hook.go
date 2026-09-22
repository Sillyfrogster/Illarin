package discord

import (
	"errors"
	"regexp"
	"strings"
)

const Webhooks = "https://discord.com/api/webhooks/"

var snowflake = regexp.MustCompile(`^[0-9]{17,20}$`)

var tokenShape = regexp.MustCompile(`^[A-Za-z0-9_-]{16,120}$`)

var ErrNotACapability = errors.New(
	"a Discord integration is an incoming webhook address on discord.com",
)

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
