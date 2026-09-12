package discord

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const capability = "https://discord.com/api/webhooks/1234567890123456789/a-long-webhook-token"

func TestACapabilityIsReadIntoItsIdAndToken(t *testing.T) {
	read, err := ReadCapability(capability)

	if err != nil {
		t.Fatalf("read capability: %v", err)
	}
	if read.ID != "1234567890123456789" {
		t.Errorf("id = %q", read.ID)
	}
	if read.Token != "a-long-webhook-token" {
		t.Errorf("token = %q", read.Token)
	}
}

func TestOnlyADiscordWebhookAddressIsACapability(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/api/webhooks/123/token",
		"http://discord.com/api/webhooks/123/token",
		"https://discord.com/api/webhooks/123",
		"https://discord.com/api/webhooks/notanid/token",
		"https://discord.com/api/webhooks/1234567890123456789/token/messages/1",
		"https://discord.com/api/v10/webhooks/1234567890123456789/token",
		"https://discord.com/api/webhooks/1234567890123456789/",
	} {
		if _, err := ReadCapability(raw); err == nil {
			t.Errorf("%q was accepted", raw)
		}
	}
}

func TestConfirmingModeAsksDiscordForTheMessage(t *testing.T) {
	read, err := ReadCapability(capability)
	if err != nil {
		t.Fatalf("read capability: %v", err)
	}

	if got := read.Confirming(); got != capability+"?wait=true" {
		t.Errorf("confirming address = %q", got)
	}
}

func TestAWebhookReadKeepsOnlySafeIdentity(t *testing.T) {
	body := []byte(`{
		"id": "1234567890123456789",
		"token": "a-long-webhook-token",
		"guild_id": "111111111111111111",
		"channel_id": "222222222222222222",
		"name": "Illarin Blog",
		"avatar": "abc",
		"user": {"id": "333", "username": "aaron"}
	}`)

	found, err := ReadWebhook(body)

	if err != nil {
		t.Fatalf("read webhook: %v", err)
	}
	want := Webhook{
		ID:        "1234567890123456789",
		GuildID:   "111111111111111111",
		ChannelID: "222222222222222222",
		Name:      "Illarin Blog",
	}
	if found != want {
		t.Errorf("webhook = %+v, want %+v", found, want)
	}
}

func TestAWebhookWithoutAGuildOrChannelIsRefused(t *testing.T) {
	for _, body := range []string{
		`{"id":"1","channel_id":"2"}`,
		`{"id":"1","guild_id":"2"}`,
		`{"guild_id":"1","channel_id":"2"}`,
		`not json`,
	} {
		if _, err := ReadWebhook([]byte(body)); err == nil {
			t.Errorf("%q was accepted", body)
		}
	}
}

func TestAMessageIdIsReadBackFromAConfirmedSend(t *testing.T) {
	if got := MessageID([]byte(`{"id":"444444444444444444","channel_id":"2"}`)); got != "444444444444444444" {
		t.Errorf("message id = %q", got)
	}
	for _, body := range []string{`{}`, `{"id":""}`, `{"id":"nope"}`, ``, `[]`} {
		if got := MessageID([]byte(body)); got != "" {
			t.Errorf("%q gave a message id of %q", body, got)
		}
	}
}

func TestARoleIsOnlyEverASnowflake(t *testing.T) {
	if err := CheckRole("111111111111111111"); err != nil {
		t.Errorf("a snowflake was refused: %v", err)
	}
	for _, raw := range []string{"", "everyone", "@everyone", "123", strings.Repeat("1", 21)} {
		if err := CheckRole(raw); err == nil {
			t.Errorf("%q was accepted as a role", raw)
		}
	}
}

type announcement struct {
	Content string `json:"content"`
	Embeds  []struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Timestamp   string `json:"timestamp"`
		Color       int    `json:"color"`
		Author      struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"author"`
		Fields []struct {
			Name   string `json:"name"`
			Value  string `json:"value"`
			Inline bool   `json:"inline"`
		} `json:"fields"`
		Image struct {
			URL string `json:"url"`
		} `json:"image"`
		Footer struct {
			Text string `json:"text"`
		} `json:"footer"`
	} `json:"embeds"`
	Mentions struct {
		Parse []string `json:"parse"`
		Roles []string `json:"roles"`
		Users []string `json:"users"`
	} `json:"allowed_mentions"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url"`
	ThreadID  string `json:"thread_id"`
}

func announced(t *testing.T, one Announcement) announcement {
	t.Helper()
	body, err := one.Body()
	if err != nil {
		t.Fatalf("compose the announcement: %v", err)
	}
	var read announcement
	if err := json.Unmarshal(body, &read); err != nil {
		t.Fatalf("decode the announcement: %v", err)
	}
	return read
}

func aRelease() Announcement {
	return Announcement{
		Title:    "Illarin 2.1 is out",
		Summary:  "Packs travel with their worldbooks now.",
		URL:      "https://blog.illarin.test/illarin-2-1",
		Image:    "https://blog.illarin.test/illarin-2-1/card.png",
		Category: "Release",
		Version:  "2.1.0",
		Author:   Author{Name: "Aaron", URL: "https://illarin.test/@aaron"},
		At:       time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC),
	}
}

func TestTheAnnouncementCarriesWhatIllarinDecidedToSay(t *testing.T) {
	read := announced(t, aRelease())

	if len(read.Embeds) != 1 {
		t.Fatalf("embeds = %d, want 1", len(read.Embeds))
	}
	embed := read.Embeds[0]
	if embed.Title != "Illarin 2.1 is out" {
		t.Errorf("title = %q", embed.Title)
	}
	if embed.Description != "Packs travel with their worldbooks now." {
		t.Errorf("description = %q", embed.Description)
	}
	if embed.URL != "https://blog.illarin.test/illarin-2-1" {
		t.Errorf("url = %q", embed.URL)
	}
	if embed.Timestamp != "2026-09-06T12:00:00Z" {
		t.Errorf("timestamp = %q", embed.Timestamp)
	}
	if embed.Image.URL != "https://blog.illarin.test/illarin-2-1/card.png" {
		t.Errorf("image = %q", embed.Image.URL)
	}
	if embed.Author.Name != "Aaron" || embed.Author.URL != "https://illarin.test/@aaron" {
		t.Errorf("author = %+v", embed.Author)
	}
	if embed.Footer.Text != Publication {
		t.Errorf("footer = %q", embed.Footer.Text)
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("fields = %d, want a category and a version", len(embed.Fields))
	}
	if embed.Fields[0].Value != "Release" || embed.Fields[1].Value != "2.1.0" {
		t.Errorf("fields = %+v", embed.Fields)
	}
}

func TestAnAnnouncementWithoutAVersionOrPictureLeavesThemOut(t *testing.T) {
	one := aRelease()
	one.Version = ""
	one.Image = ""

	read := announced(t, one)

	if len(read.Embeds[0].Fields) != 1 {
		t.Errorf("fields = %+v, want the category alone", read.Embeds[0].Fields)
	}
	if read.Embeds[0].Image.URL != "" {
		t.Errorf("image = %q", read.Embeds[0].Image.URL)
	}
}

func TestTheNoteIsTheOnlyTextAContributorPutsInTheMessage(t *testing.T) {
	one := aRelease()
	one.Note = "Worth a read if you keep worldbooks."

	read := announced(t, one)

	if read.Content != "Worth a read if you keep worldbooks." {
		t.Errorf("content = %q", read.Content)
	}
}

func TestNoAnnouncementParsesAMentionOutOfItsText(t *testing.T) {
	one := aRelease()
	one.Note = "@everyone <@&999999999999999999> <@111111111111111111>"

	read := announced(t, one)

	if len(read.Mentions.Parse) != 0 {
		t.Errorf("parse = %v, want nothing parsed", read.Mentions.Parse)
	}
	if len(read.Mentions.Roles) != 0 || len(read.Mentions.Users) != 0 {
		t.Errorf("mentions = %+v, want nobody named", read.Mentions)
	}
}

func TestOnlyTheConfiguredRoleIsEverMentioned(t *testing.T) {
	one := aRelease()
	one.Note = "<@&999999999999999999>"
	one.Role = "111111111111111111"

	read := announced(t, one)

	if len(read.Mentions.Roles) != 1 || read.Mentions.Roles[0] != "111111111111111111" {
		t.Errorf("roles = %v", read.Mentions.Roles)
	}
	if !strings.HasPrefix(read.Content, "<@&111111111111111111>") {
		t.Errorf("content = %q, want the role named at the front", read.Content)
	}
	if len(read.Mentions.Parse) != 0 {
		t.Errorf("parse = %v, want nothing parsed", read.Mentions.Parse)
	}
}

func TestAnAnnouncementNeverCarriesAnIdentityOrThreadOverride(t *testing.T) {
	read := announced(t, aRelease())

	if read.Username != "" || read.AvatarURL != "" || read.ThreadID != "" {
		t.Errorf("the announcement overrode Discord's own identity: %+v", read)
	}
}

func TestAnAnnouncementIsCutToWhatDiscordAccepts(t *testing.T) {
	one := aRelease()
	one.Title = strings.Repeat("t", TitleLimit+50)
	one.Summary = strings.Repeat("s", DescriptionLimit+50)
	one.Note = strings.Repeat("n", ContentLimit+50)
	one.Category = strings.Repeat("c", FieldLimit+50)
	one.Version = strings.Repeat("v", FieldLimit+50)
	one.Update = strings.Repeat("u", FieldLimit+50)
	one.Footer = strings.Repeat("f", FooterLimit+50)
	one.Author.Name = strings.Repeat("🐦", TitleLimit)

	read := announced(t, one)

	if len([]rune(read.Embeds[0].Title)) > TitleLimit {
		t.Errorf("title is %d runes", len([]rune(read.Embeds[0].Title)))
	}
	if len([]rune(read.Embeds[0].Description)) > DescriptionLimit {
		t.Errorf("description is %d runes", len([]rune(read.Embeds[0].Description)))
	}
	if len([]rune(read.Content)) > ContentLimit {
		t.Errorf("content is %d runes", len([]rune(read.Content)))
	}
	shown := read.Embeds[0]
	total := textLength(shown.Title) + textLength(shown.Description) + textLength(shown.Author.Name) + textLength(shown.Footer.Text)
	for _, field := range shown.Fields {
		total += textLength(field.Name) + textLength(field.Value)
	}
	if total > EmbedLimit {
		t.Errorf("embed is %d characters", total)
	}
	if textLength(cut(strings.Repeat("🐦", ContentLimit), ContentLimit)) > ContentLimit {
		t.Fatal("emoji content exceeds Discord's limit")
	}
}
