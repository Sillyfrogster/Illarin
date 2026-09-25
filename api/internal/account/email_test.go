package account

import (
	"bytes"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAccountEmailHasEscapedHTMLAndPlainText(t *testing.T) {
	link := `https://illarin.test/verify?token=one&next="<script>`
	message, err := verificationEmail(link)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := accountEmailMIME("Illarin <mail@illarin.test>", "creator@example.com", message)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := mail.ReadMessage(bytes.NewReader(encoded))
	if err != nil {
		t.Fatal(err)
	}
	_, params, err := mime.ParseMediaType(parsed.Header.Get("Content-Type"))
	if err != nil {
		t.Fatal(err)
	}
	parts := multipart.NewReader(parsed.Body, params["boundary"])
	var contents []string
	for {
		part, err := parts.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(quotedprintable.NewReader(part))
		if err != nil {
			t.Fatal(err)
		}
		contents = append(contents, string(body))
	}
	if len(contents) != 2 || !strings.Contains(contents[0], link) ||
		!strings.Contains(contents[1], `href="https://illarin.test/verify?token=one&amp;next=`) ||
		strings.Contains(contents[1], "<script>") {
		t.Fatalf("account email lost its plain link or escaped HTML link: plain=%q html=%q", contents[0], contents[1])
	}
}

func TestEmailPreview(t *testing.T) {
	directory := os.Getenv("EMAIL_PREVIEW_DIR")
	if directory == "" {
		t.Skip("email preview not requested")
	}
	for name, render := range map[string]func(string) (accountEmail, error){
		"verification":   verificationEmail,
		"password-reset": passwordResetEmail,
	} {
		message, err := render("https://illarin.example/account/example-link")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name+".html"), []byte(message.html), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
