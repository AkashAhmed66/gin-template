// Package mail sends transactional email via SMTP. The master switch
// (cfg.Enabled) defaults to false so misconfigured local runs don't silently
// swallow notifications — see .env.example.
package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/config"
	"go.uber.org/zap"
)

// Message is a rendered email ready to send.
type Message struct {
	To      []string
	Subject string
	HTML    string
	Text    string // fallback for non-HTML clients
}

// Service is the public mail interface used by application code.
type Service interface {
	Send(ctx context.Context, msg Message) error
	Render(name string, data any) (string, error)
	Enabled() bool
}

type smtpService struct {
	cfg       config.MailConfig
	log       *zap.Logger
	templates *template.Template
	once      sync.Once
}

// New returns a Service configured for SMTP. Templates are loaded lazily from
// templatesDir on the first Render call (or send call that needs rendering).
func New(cfg config.MailConfig, templatesDir string, log *zap.Logger) Service {
	s := &smtpService{cfg: cfg, log: log}
	s.loadTemplates(templatesDir)
	return s
}

// Enabled returns the master switch state.
func (s *smtpService) Enabled() bool { return s.cfg.Enabled }

// Send transmits msg via SMTP. No-op (with a log line) when Enabled() is false
// so dev workflows don't hang on missing creds.
func (s *smtpService) Send(ctx context.Context, msg Message) error {
	if !s.cfg.Enabled {
		s.log.Info("mail disabled — skip send",
			zap.Strings("to", msg.To),
			zap.String("subject", msg.Subject),
		)
		return nil
	}

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	from := fmt.Sprintf("%s <%s>", s.cfg.FromName, s.cfg.From)

	var body bytes.Buffer
	writeHeader(&body, "From", from)
	writeHeader(&body, "To", strings.Join(msg.To, ", "))
	writeHeader(&body, "Subject", msg.Subject)
	writeHeader(&body, "MIME-Version", "1.0")
	writeHeader(&body, "Content-Type", "text/html; charset=UTF-8")
	writeHeader(&body, "Date", time.Now().UTC().Format(time.RFC1123Z))
	body.WriteString("\r\n")
	if msg.HTML != "" {
		body.WriteString(msg.HTML)
	} else {
		body.WriteString(msg.Text)
	}

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	if s.cfg.TLS {
		return sendSTARTTLS(addr, s.cfg.Host, auth, s.cfg.From, msg.To, body.Bytes())
	}
	return smtp.SendMail(addr, auth, s.cfg.From, msg.To, body.Bytes())
}

// Render returns the rendered HTML body for the named template.
func (s *smtpService) Render(name string, data any) (string, error) {
	if s.templates == nil {
		return "", fmt.Errorf("no templates loaded")
	}
	var buf bytes.Buffer
	if err := s.templates.ExecuteTemplate(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (s *smtpService) loadTemplates(dir string) {
	if dir == "" {
		return
	}
	pattern := filepath.Join(dir, "*.html")
	t, err := template.New("").Funcs(template.FuncMap{
		"safeHTML": func(v string) template.HTML { return template.HTML(v) },
	}).ParseGlob(pattern)
	if err != nil {
		// Logged once; service still works for callers that pass raw HTML.
		s.log.Warn("mail templates load failed",
			zap.String("dir", dir),
			zap.Error(err),
		)
		return
	}
	s.templates = t
}

func writeHeader(b *bytes.Buffer, name, value string) {
	b.WriteString(name)
	b.WriteString(": ")
	b.WriteString(value)
	b.WriteString("\r\n")
}

func sendSTARTTLS(addr, host string, auth smtp.Auth, from string, to []string, msg []byte) error {
	c, err := smtp.Dial(addr)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()
	if err := c.Hello("localhost"); err != nil {
		return err
	}
	if ok, _ := c.Extension("STARTTLS"); ok {
		cfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		if err := c.StartTLS(cfg); err != nil {
			return err
		}
	}
	if auth != nil {
		if ok, _ := c.Extension("AUTH"); ok {
			if err := c.Auth(auth); err != nil {
				return err
			}
		}
	}
	if err := c.Mail(from); err != nil {
		return err
	}
	for _, r := range to {
		if err := c.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := c.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return c.Quit()
}
