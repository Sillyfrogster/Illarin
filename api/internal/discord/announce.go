package discord

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const Publication = "Illarin Blog"

const Stripe = 0xE2E2DD

const (
	ContentLimit     = 2000
	TitleLimit       = 256
	DescriptionLimit = 4096
	FieldLimit       = 1024
)

const (
	CategoryHeading = "Category"
	VersionHeading  = "Version"
	UpdateHeading   = "Update"
)

const Site = "Illarin"

type Author struct {
	Name string
	URL  string
}

type Announcement struct {
	Title    string
	Summary  string
	URL      string
	Image    string
	Category string
	Update   string
	Version  string
	Note     string
	Role     string
	Author   Author
	At       time.Time
	Footer   string
}

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
		Fields:      []field{},
		Footer:      footer{Text: a.footer()},
	}
	if a.Category != "" {
		shown.Fields = append(shown.Fields, field{
			Name: CategoryHeading, Value: cut(a.Category, FieldLimit), Inline: true,
		})
	}
	if a.Update != "" {
		shown.Fields = append(shown.Fields, field{
			Name: UpdateHeading, Value: cut(a.Update, FieldLimit), Inline: true,
		})
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

func (a Announcement) footer() string {
	if a.Footer == "" {
		return Publication
	}
	return a.Footer
}

func cut(said string, limit int) string {
	held := []rune(said)
	if len(held) <= limit {
		return said
	}
	return strings.TrimRight(string(held[:limit-1]), " ") + "…"
}

type message struct {
	Content  string   `json:"content"`
	Embeds   []embed  `json:"embeds"`
	Mentions mentions `json:"allowed_mentions"`
}

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
