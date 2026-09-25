package account

import (
	"context"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
)

type SMTPSettings struct {
	Address  string
	From     string
	Username string
	Password string
}

type SMTPSender struct {
	settings SMTPSettings
	from     *mail.Address
	host     string
}

func NewSMTPSender(settings SMTPSettings) (*SMTPSender, error) {
	host, _, err := net.SplitHostPort(settings.Address)
	if err != nil {
		return nil, fmt.Errorf("SMTP_ADDR must include a host and port: %w", err)
	}
	from, err := mail.ParseAddress(settings.From)
	if err != nil {
		return nil, fmt.Errorf("SMTP_FROM is not an email address: %w", err)
	}
	return &SMTPSender{settings: settings, from: from, host: host}, nil
}

func (s *SMTPSender) SendVerification(_ context.Context, address, link string) error {
	message, err := verificationEmail(link)
	if err != nil {
		return err
	}
	return s.send(address, message)
}

func (s *SMTPSender) SendPasswordReset(_ context.Context, address, link string) error {
	message, err := passwordResetEmail(link)
	if err != nil {
		return err
	}
	return s.send(address, message)
}

func (s *SMTPSender) send(address string, message accountEmail) error {
	var auth smtp.Auth
	if s.settings.Username != "" {
		auth = smtp.PlainAuth("", s.settings.Username, s.settings.Password, s.host)
	}
	mime, err := accountEmailMIME(s.from.String(), address, message)
	if err != nil {
		return err
	}

	if err := smtp.SendMail(
		s.settings.Address,
		auth,
		s.from.Address,
		[]string{address},
		mime,
	); err != nil {
		return fmt.Errorf("send account email: %w", err)
	}
	return nil
}
