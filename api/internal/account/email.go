package account

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"net/textproto"
)

type accountEmail struct {
	subject string
	text    string
	html    string
}

var accountEmailTemplate = template.Must(template.New("account email").Parse(`<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>{{.Subject}}</title></head>
<body style="margin:0;padding:0;background:#f5f3f7;color:#17141b;font-family:Arial,Helvetica,sans-serif;">
<div style="display:none;max-height:0;overflow:hidden;opacity:0;">{{.Summary}}</div>
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="background:#f5f3f7;"><tr><td align="center" style="padding:40px 16px 56px;">
  <table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="max-width:560px;">
    <tr><td style="padding:0 4px 24px;color:#17141b;font-size:24px;font-weight:700;letter-spacing:-0.06em;line-height:1.2;">Illarin<span style="color:#6d28d9;">.</span></td></tr>
    <tr><td style="background:#ffffff;border:1px solid #e8e3ed;border-radius:16px;padding:44px 40px 40px;">
      <h1 style="margin:0 0 18px;font-size:32px;font-weight:700;letter-spacing:-0.045em;line-height:1.16;">{{.Title}}</h1>
      <p style="margin:0;color:#554e5b;font-size:16px;line-height:1.6;">{{.Summary}}</p>
      <table role="presentation" cellpadding="0" cellspacing="0" border="0" style="margin:32px 0;"><tr><td bgcolor="#6d28d9" style="border-radius:8px;"><a href="{{.Link}}" style="display:inline-block;padding:15px 22px;color:#ffffff;font-size:16px;font-weight:700;line-height:1.2;text-decoration:none;">{{.Action}}</a></td></tr></table>
      <p style="margin:0;color:#554e5b;font-size:14px;line-height:1.6;">{{.Note}}</p>
      <div style="height:1px;background:#e8e3ed;margin:32px 0 24px;"></div>
      <p style="margin:0 0 8px;color:#554e5b;font-size:13px;line-height:1.5;">Button not working? Open this link:</p>
      <a href="{{.Link}}" style="color:#6d28d9;font-size:13px;line-height:1.5;overflow-wrap:anywhere;word-break:break-all;">{{.Link}}</a>
    </td></tr>
    <tr><td style="padding:22px 4px 0;color:#716a79;font-size:12px;line-height:1.5;">Sent by Illarin for your account.</td></tr>
  </table>
</td></tr></table>
</body></html>`))

func verificationEmail(link string) (accountEmail, error) {
	return renderAccountEmail("Verify your Illarin email", "Verify your email", "Use this link to verify the email address for your Illarin account.", "Verify email", "This link expires in 24 hours. If you didn't request it, you can ignore this email.", link)
}

func passwordResetEmail(link string) (accountEmail, error) {
	return renderAccountEmail("Reset your Illarin password", "Reset your password", "Use this link to set a new password for your Illarin account.", "Reset password", "This link expires in 1 hour. If you didn't request it, you can ignore this email.", link)
}

func renderAccountEmail(subject, title, summary, action, note, link string) (accountEmail, error) {
	var html bytes.Buffer
	if err := accountEmailTemplate.Execute(&html, struct {
		Subject, Title, Summary, Action, Note, Link string
	}{subject, title, summary, action, note, link}); err != nil {
		return accountEmail{}, fmt.Errorf("render account email: %w", err)
	}
	return accountEmail{
		subject: subject,
		text:    fmt.Sprintf("%s\r\n\r\n%s\r\n\r\n%s\r\n\r\n%s\r\n", title, summary, link, note),
		html:    html.String(),
	}, nil
}

func accountEmailMIME(from, address string, message accountEmail) ([]byte, error) {
	sender, err := mail.ParseAddress(from)
	if err != nil {
		return nil, fmt.Errorf("invalid account email sender: %w", err)
	}
	to, err := mail.ParseAddress(address)
	if err != nil || to.Address != address {
		return nil, fmt.Errorf("invalid account email recipient")
	}
	var body bytes.Buffer
	parts := multipart.NewWriter(&body)
	for _, part := range []struct{ contentType, content string }{
		{"text/plain; charset=UTF-8", message.text},
		{"text/html; charset=UTF-8", message.html},
	} {
		header := textproto.MIMEHeader{
			"Content-Type":              {part.contentType},
			"Content-Transfer-Encoding": {"quoted-printable"},
		}
		writer, err := parts.CreatePart(header)
		if err != nil {
			return nil, fmt.Errorf("create account email part: %w", err)
		}
		encoded := quotedprintable.NewWriter(writer)
		if _, err := io.WriteString(encoded, part.content); err != nil {
			return nil, fmt.Errorf("write account email part: %w", err)
		}
		if err := encoded.Close(); err != nil {
			return nil, fmt.Errorf("finish account email part: %w", err)
		}
	}
	if err := parts.Close(); err != nil {
		return nil, fmt.Errorf("finish account email: %w", err)
	}
	header := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: multipart/alternative; boundary=%q\r\n\r\n", sender.String(), to.String(), message.subject, parts.Boundary())
	return append([]byte(header), body.Bytes()...), nil
}
