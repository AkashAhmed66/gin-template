package auth

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/AkashAhmed66/gin-template/internal/common/mail"
	"github.com/AkashAhmed66/gin-template/internal/common/queue"
	"go.uber.org/zap"
)

// Task type constants. Use these on both the enqueue and handle side.
const (
	TaskPasswordResetEmail = "auth:password-reset-email"
)

// PasswordResetEmailPayload is the JSON-encoded payload for a password reset
// email task. Keep all data the handler needs in here — handlers may run on a
// different instance with no access to the original request's variables.
type PasswordResetEmailPayload struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	ResetURL string `json:"resetUrl"`
	TTL      string `json:"ttl"`
}

// RegisterQueueHandlers wires this module's task handlers onto the global
// queue Manager. Called once at bootstrap.
func RegisterQueueHandlers(q *queue.Manager, mailer mail.Service, log *zap.Logger) {
	if q == nil {
		return
	}
	q.Handle(TaskPasswordResetEmail, func(ctx context.Context, body []byte) error {
		var p PasswordResetEmailPayload
		if err := json.Unmarshal(body, &p); err != nil {
			return fmt.Errorf("decode password-reset payload: %w", err)
		}
		return sendPasswordResetEmail(ctx, mailer, p)
	})
}

// sendPasswordResetEmail renders + sends the email. Called from the queue
// handler when the queue is enabled, and inline from the service when it's
// disabled (so dev without Redis still works).
func sendPasswordResetEmail(ctx context.Context, mailer mail.Service, p PasswordResetEmailPayload) error {
	body, err := mailer.Render("password-reset.html", map[string]any{
		"Username": p.Username,
		"ResetURL": p.ResetURL,
		"TTL":      p.TTL,
	})
	if err != nil {
		body = fmt.Sprintf("Reset your password: %s\n(link expires in %s)", p.ResetURL, p.TTL)
	}
	return mailer.Send(ctx, mail.Message{
		To:      []string{p.Email},
		Subject: "Reset your password",
		HTML:    body,
	})
}
