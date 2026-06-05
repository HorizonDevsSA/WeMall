package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/rs/zerolog"
)

type SMTPProvider struct {
	host     string
	port     string
	user     string
	password string
	from     string
	appName  string
	appURL   string
	logger   zerolog.Logger
}

func NewSMTPProvider(host, port, user, password, appName, appURL string, logger zerolog.Logger) *SMTPProvider {
	return &SMTPProvider{
		host:     host,
		port:     port,
		user:     user,
		password: password,
		from:     user,
		appName:  appName,
		appURL:   appURL,
		logger:   logger,
	}
}

func (p *SMTPProvider) SendEmail(to, subject, htmlBody string) error {
	// If credentials are empty, run in stub mode (log to console)
	if p.user == "" || p.password == "" {
		p.logger.Info().
			Str("to", to).
			Str("subject", subject).
			Msg("[STUB MODE] Email simulated successfully (no credentials configured)")
		return nil
	}

	headers := map[string]string{
		"From":         fmt.Sprintf("%s <noreply@wemall.co.zw>", p.appName),
		"To":           to,
		"Subject":      subject,
		"MIME-Version": "1.0",
		"Content-Type": `text/html; charset="UTF-8"`,
		"X-Mailer":     p.appName + " Mailer v1.0",
	}

	var msg bytes.Buffer
	for k, v := range headers {
		msg.WriteString(k + ": " + v + "\r\n")
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	auth := smtp.PlainAuth("", p.user, p.password, p.host)

	// Use Implicit TLS if port is 465, or explicit TLS (STARTTLS) if port is 587
	if p.port == "465" {
		return p.sendImplicitTLS(to, msg.Bytes(), auth)
	}

	return p.sendSTARTTLS(to, msg.Bytes(), auth)
}

func (p *SMTPProvider) sendImplicitTLS(to string, msg []byte, auth smtp.Auth) error {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         p.host,
	}

	addr := p.host + ":" + p.port
	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("smtp implicit tls dial: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, p.host)
	if err != nil {
		return fmt.Errorf("smtp client creation: %w", err)
	}
	defer client.Close()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(p.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data open: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp data write: %w", err)
	}
	return w.Close()
}

func (p *SMTPProvider) sendSTARTTLS(to string, msg []byte, auth smtp.Auth) error {
	addr := p.host + ":" + p.port
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("smtp dial: %w", err)
	}
	defer client.Close()

	tlsCfg := &tls.Config{
		ServerName: p.host,
	}
	if err = client.StartTLS(tlsCfg); err != nil {
		// Fallback without STARTTLS if not enforced, but let's log it.
		p.logger.Warn().Err(err).Msg("STARTTLS failed, attempting plain auth")
	}

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err = client.Mail(p.from); err != nil {
		return fmt.Errorf("smtp mail from: %w", err)
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp rcpt to: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp data open: %w", err)
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("smtp data write: %w", err)
	}
	return w.Close()
}
