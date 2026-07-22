package auth

import (
	"time"

	"github.com/google/uuid"
)

// User mirrors a row of the users table (password hash excluded from JSON).
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"name"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// RegisterRequest is the payload for customer registration.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the payload for logging in.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest carries the opaque refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// AuthResponse is returned on successful register/login/refresh. It mirrors the
// frontend Session shape plus the token pair.
type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresAt    int64  `json:"expiresAt"`
}
