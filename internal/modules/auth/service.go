package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/AkashAhmed66/gin-template/internal/common/errs"
	"github.com/AkashAhmed66/gin-template/internal/common/mail"
	"github.com/AkashAhmed66/gin-template/internal/common/security"
	"github.com/AkashAhmed66/gin-template/internal/config"
	"github.com/AkashAhmed66/gin-template/internal/modules/permission"
	"github.com/AkashAhmed66/gin-template/internal/modules/role"
	"github.com/AkashAhmed66/gin-template/internal/modules/session"
	"github.com/AkashAhmed66/gin-template/internal/modules/user"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Service is the application-facing auth API.
type Service interface {
	Register(ctx context.Context, req RegisterRequest, ua, ip string) (*AuthResponse, error)
	Login(ctx context.Context, req LoginRequest, ua, ip string) (*AuthResponse, error)
	Refresh(ctx context.Context, refreshToken, ua, ip string) (*AuthResponse, error)
	Logout(ctx context.Context, sessionID uint64) error
	LogoutAll(ctx context.Context, userID uint64) error
	ListMySessions(ctx context.Context, userID, currentSID uint64) ([]session.Response, error)
	RevokeMySession(ctx context.Context, userID, sessionID uint64) error
	Impersonate(ctx context.Context, callerID, targetID uint64, ua, ip string) (*AuthResponse, error)
	ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req ResetPasswordRequest) error
	CleanupExpiredResetTokens(ctx context.Context) (int64, error)
}

type service struct {
	db       *gorm.DB
	users    user.Repository
	roles    role.Repository
	sessions session.Service
	jwt      *security.JwtService
	mail     mail.Service
	mailCfg  config.MailConfig
	log      *zap.Logger
}

// NewService wires the auth service. The roles dependency is used to assign
// the default USER role on register.
func NewService(
	db *gorm.DB,
	users user.Repository,
	roles role.Repository,
	sessions session.Service,
	jwt *security.JwtService,
	mailSvc mail.Service,
	mailCfg config.MailConfig,
	log *zap.Logger,
) Service {
	return &service{
		db: db, users: users, roles: roles, sessions: sessions,
		jwt: jwt, mail: mailSvc, mailCfg: mailCfg, log: log,
	}
}

func (s *service) Register(ctx context.Context, req RegisterRequest, ua, ip string) (*AuthResponse, error) {
	if taken, err := s.users.ExistsByUsername(ctx, req.Username); err != nil {
		return nil, err
	} else if taken {
		return nil, errs.Duplicate("Username already taken")
	}
	if taken, err := s.users.ExistsByEmail(ctx, req.Email); err != nil {
		return nil, err
	} else if taken {
		return nil, errs.Duplicate("Email already in use")
	}

	hash, err := security.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	u := &user.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hash,
		FullName:     req.FullName,
		Enabled:      true,
	}
	defaultRole, err := s.roles.GetByName(ctx, role.NameUser)
	if err != nil {
		return nil, err
	}
	if defaultRole != nil {
		u.Roles = []role.Role{*defaultRole}
	}
	if err := s.users.Create(ctx, u); err != nil {
		return nil, err
	}
	full, err := s.users.GetByID(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	if full == nil {
		return nil, errs.Internal("User not found after creation")
	}

	go s.sendWelcome(*full)
	return s.issueTokens(ctx, *full, 0, req.DeviceName, ua, ip)
}

func (s *service) Login(ctx context.Context, req LoginRequest, ua, ip string) (*AuthResponse, error) {
	u, err := s.users.GetByUsernameOrEmail(ctx, req.UsernameOrEmail)
	if err != nil {
		return nil, err
	}
	if u == nil || !security.CheckPassword(u.PasswordHash, req.Password) {
		return nil, errs.Unauthorized("Invalid credentials")
	}
	if !u.Enabled {
		return nil, errs.Unauthorized("Account is deactivated")
	}
	return s.issueTokens(ctx, *u, 0, req.DeviceName, ua, ip)
}

func (s *service) Refresh(ctx context.Context, refreshToken, ua, ip string) (*AuthResponse, error) {
	claims, err := s.jwt.Parse(refreshToken)
	if err != nil {
		return nil, errs.Unauthorized("Invalid refresh token")
	}
	if claims.Type != security.TokenTypeRefresh {
		return nil, errs.Unauthorized("Refresh token required")
	}
	if claims.SessionID != 0 {
		if err := s.sessions.Validate(claims.SessionID, claims.UserID); err != nil {
			return nil, errs.Unauthorized("Session no longer active")
		}
	}
	u, err := s.users.GetByID(ctx, claims.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errs.Unauthorized("User no longer exists")
	}
	if !u.Enabled {
		return nil, errs.Unauthorized("Account is deactivated")
	}

	// Rotate: issue new tokens against the same session row so device identity persists.
	authorities := user.AuthoritiesOf(*u)
	access, exp, err := s.jwt.IssueAccess(u.ID, u.Username, claims.SessionID, claims.ImpersonatorID, authorities)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.jwt.IssueRefresh(u.ID, u.Username, claims.SessionID, claims.ImpersonatorID, authorities)
	if err != nil {
		return nil, err
	}
	s.sessions.Touch(claims.SessionID)
	_ = ua
	_ = ip
	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresAt:    exp,
		UserID:       u.ID,
		Username:     u.Username,
		SessionID:    claims.SessionID,
		Authorities:  authorities,
	}, nil
}

func (s *service) Logout(ctx context.Context, sessionID uint64) error {
	return s.sessions.Revoke(ctx, sessionID, "user-logout")
}

func (s *service) LogoutAll(ctx context.Context, userID uint64) error {
	return s.sessions.RevokeAllForUser(ctx, userID, "user-logout-all")
}

func (s *service) ListMySessions(ctx context.Context, userID, currentSID uint64) ([]session.Response, error) {
	return s.sessions.ListForUser(ctx, userID, currentSID)
}

func (s *service) RevokeMySession(ctx context.Context, userID, sessionID uint64) error {
	row, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		return err
	}
	if row.UserID != userID {
		// Don't leak existence to non-owners.
		return errs.NotFound("Session", sessionID)
	}
	return s.sessions.Revoke(ctx, sessionID, "user-revoked-device")
}

func (s *service) Impersonate(ctx context.Context, callerID, targetID uint64, ua, ip string) (*AuthResponse, error) {
	if callerID == targetID {
		return nil, errs.BadRequest("Cannot impersonate yourself")
	}
	target, err := s.users.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	if target == nil {
		return nil, errs.NotFound("User", targetID)
	}
	if !target.Enabled {
		return nil, errs.BadRequest("Cannot impersonate a deactivated account")
	}
	return s.issueTokens(ctx, *target, callerID, fmt.Sprintf("impersonation-by-%d", callerID), ua, ip)
}

func (s *service) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	// Always return success-shaped to the controller — never leak email existence.
	u, err := s.users.GetByUsernameOrEmail(ctx, req.Email)
	if err != nil || u == nil {
		return nil
	}
	if !u.Enabled {
		return nil
	}
	rawToken, hash, err := generateResetToken()
	if err != nil {
		s.log.Warn("password reset token gen failed", zap.Error(err))
		return nil
	}
	row := &PasswordResetToken{
		UserID:    u.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(s.mailCfg.PasswordResetTTL),
		CreatedAt: time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		s.log.Warn("password reset token save failed", zap.Error(err))
		return nil
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.mailCfg.AppURL, rawToken)
	body, err := s.mail.Render("password-reset.html", map[string]any{
		"Username": u.Username,
		"ResetURL": resetURL,
		"TTL":      s.mailCfg.PasswordResetTTL.String(),
	})
	if err != nil {
		body = fmt.Sprintf("Reset your password: %s\n(link expires in %s)", resetURL, s.mailCfg.PasswordResetTTL)
	}
	_ = s.mail.Send(ctx, mail.Message{
		To:      []string{u.Email},
		Subject: "Reset your password",
		HTML:    body,
	})
	return nil
}

func (s *service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	hashed := hashToken(req.Token)
	var row PasswordResetToken
	err := s.db.WithContext(ctx).Where("token_hash = ?", hashed).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errs.BadRequest("Invalid or expired reset token")
		}
		return err
	}
	if row.ConsumedAt != nil {
		return errs.BadRequest("Token has already been used")
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return errs.BadRequest("Token has expired")
	}
	u, err := s.users.GetByID(ctx, row.UserID)
	if err != nil {
		return err
	}
	if u == nil {
		return errs.BadRequest("User no longer exists")
	}
	hash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	if err := s.users.Update(ctx, u); err != nil {
		return err
	}
	now := time.Now().UTC()
	row.ConsumedAt = &now
	if err := s.db.WithContext(ctx).Save(&row).Error; err != nil {
		return err
	}
	_ = s.users.IncrementTokenVersion(ctx, u.ID)
	_ = s.sessions.RevokeAllForUser(ctx, u.ID, "password-reset")
	return nil
}

func (s *service) CleanupExpiredResetTokens(ctx context.Context) (int64, error) {
	res := s.db.WithContext(ctx).
		Where("expires_at <= ? OR consumed_at IS NOT NULL", time.Now().UTC()).
		Delete(&PasswordResetToken{})
	return res.RowsAffected, res.Error
}

// issueTokens persists a session row and signs a matching access + refresh pair.
func (s *service) issueTokens(ctx context.Context, u user.User, impersonatorID uint64, deviceName, ua, ip string) (*AuthResponse, error) {
	authorities := user.AuthoritiesOf(u)
	refreshExpiry := time.Now().UTC().Add(s.jwt.RefreshTTL())

	sess, err := s.sessions.Create(ctx, u.ID, impersonatorID, refreshExpiry, deviceName, ua, ip)
	if err != nil {
		return nil, err
	}

	access, accessExp, err := s.jwt.IssueAccess(u.ID, u.Username, sess.ID, impersonatorID, authorities)
	if err != nil {
		return nil, err
	}
	refresh, _, err := s.jwt.IssueRefresh(u.ID, u.Username, sess.ID, impersonatorID, authorities)
	if err != nil {
		return nil, err
	}
	return &AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    "Bearer",
		ExpiresAt:    accessExp,
		UserID:       u.ID,
		Username:     u.Username,
		SessionID:    sess.ID,
		Authorities:  authorities,
	}, nil
}

func (s *service) sendWelcome(u user.User) {
	if !s.mail.Enabled() {
		return
	}
	body, err := s.mail.Render("welcome.html", map[string]any{
		"Username": u.Username,
		"AppURL":   s.mailCfg.AppURL,
	})
	if err != nil {
		body = fmt.Sprintf("Welcome %s! Visit %s to get started.", u.Username, s.mailCfg.AppURL)
	}
	_ = s.mail.Send(context.Background(), mail.Message{
		To:      []string{u.Email},
		Subject: "Welcome",
		HTML:    body,
	})
}

func generateResetToken() (raw string, hashed string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// Ensure permission package import is used (helps editors that auto-tidy).
var _ = permission.UserRead
