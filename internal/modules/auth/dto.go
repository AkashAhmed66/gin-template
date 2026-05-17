// Package auth carries the public authentication endpoints — register, login,
// refresh, logout, logout-all, sessions, impersonate, forgot/reset password.
package auth

import "time"

type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Email      string `json:"email" binding:"required,email,max=255"`
	Password   string `json:"password" binding:"required,min=8,max=128"`
	FullName   string `json:"fullName" binding:"max=200"`
	DeviceName string `json:"deviceName" binding:"max=200"`
}

type LoginRequest struct {
	UsernameOrEmail string `json:"usernameOrEmail" binding:"required"`
	Password        string `json:"password" binding:"required"`
	DeviceName      string `json:"deviceName" binding:"max=200"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email,max=255"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8,max=128"`
}

type AuthResponse struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	TokenType    string    `json:"tokenType"`
	ExpiresAt    time.Time `json:"expiresAt"`
	UserID       uint64    `json:"userId"`
	Username     string    `json:"username"`
	SessionID    uint64    `json:"sessionId"`
	Authorities  []string  `json:"authorities"`
}
