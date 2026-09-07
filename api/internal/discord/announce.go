package discord

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Publication is the name every announcement is signed with.
const Publication = "Illarin Blog"

// Stripe is the colour down the side of every announcement, which is Illarin's
// own ink read against Discord's dark ground.
const Stripe = 0xE2E2DD

// What Discord accepts in each part of a message.
const (
	ContentLimit     = 2000
	TitleLimit       = 256
	DescriptionLimit = 4096
	FieldLimit       = 1024
)

// The headings on the two facts an announcement states beside the summary.
const (
	CategoryHeading = "Category"
	VersionHeading  = "Version"
)

// Author is the stored attribution one announcement carries.
type Author struct {
	Name string
	URL  string
}

// Announcement is everything Illarin says about a post on Discord. A
// contributor supplies the note and nothing else here, because the shape below
// is Illarin's rather than theirs.
type Announcement struct {
	Title    string
	Summary  string
	URL      string
	Image    string
	Category string
	Version  string
	Note     string
	Role     string
	Author   Author
	At       time.Time
}

// Body writes the exact bytes one announcement is sent as.
func (a Announcement) Body() ([]byte, error) {
	body, err := json.Marshal(message{
		Content:  a.content(),
		Embeds:   []embed{a.embed()},
		Mentions: mentions{Parse: []string{}, Roles: a.roles(), Users: []string{}},
	})
	if err != nil {
		return nil, fmt.Errorf("write the Discord announcement: %w", err)
	}
	return body, nil
}

func (a Announcement) content() string {
	said := strings.TrimSpace(a.Note)
	if a.Role != "" {
		said = strings.TrimSpace("<@&" + a.Role + "> " + said)
	}
	return cut(said, ContentLimit)
}

func (a Announcement) roles() []string {
	if a.Role == "" {
		return []string{}
	}
	return []string{a.Role}
}

func (a Announcement) embed() embed {
	shown := embed{
		Title:       cut(a.Title, TitleLimit),
		Description: cut(a.Summary, DescriptionLimit),
		URL:         a.URL,
		Color:       Stripe,
		Timestamp:   a.At.UTC().Format(time.RFC3339),
		Fields:      []field{{Name: CategoryHeading, Value: cut(a.Category, FieldLimit), Inline: true}},
		Footer:      footer{Text: Publication},
	}
	if a.Version != "" {
		shown.Fields = append(shown.Fields, field{
			Name: VersionHeading, Value: cut(a.Version, FieldLimit), Inline: true,
		})
	}
	if a.Author.Name != "" {
		shown.Author = &author{Name: cut(a.Author.Name, TitleLimit), URL: a.Author.URL}
	}
	if a.Image != "" {
		shown.Image = &picture{URL: a.Image}
	}
	return shown
}

// cut holds one part of a message inside what Discord accepts, counted the way
// Discord counts it.
func cut(said string, limit int) string {
	held := []rune(said)
	if len(held) <= limit {
		return said
	}
	return strings.TrimRight(string(held[:limit-1]), " ") + "…"
}

// message is the whole request body. It names no username, avatar or thread,
// so Discord keeps the identity the authority configured the webhook with.
type message struct {
	Content  string   `json:"content"`
	Embeds   []embed  `json:"embeds"`
	Mentions mentions `json:"allowed_mentions"`
}

// mentions is closed by construction. Parsing is off for every message, and
// the only role that can be named is the one the authority approved.
type mentions struct {
	Parse []string `json:"parse"`
	Roles []string `json:"roles"`
	Users []string `json:"users"`
}

type embed struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	URL         string   `json:"url"`
	Color       int      `json:"color"`
	Timestamp   string   `json:"timestamp"`
	Author      *author  `json:"author,omitempty"`
	Fields      []field  `json:"fields"`
	Image       *picture `json:"image,omitempty"`
	Footer      footer   `json:"footer"`
}

type author struct {
	Name string `json:"name"`
	URL  string `json:"url,omitempty"`
}

type field struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type picture struct {
	URL string `json:"url"`
}

type footer struct {
	Text string `json:"text"`
}
