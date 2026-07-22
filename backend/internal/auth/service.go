package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

// Service holds the auth business logic.
type Service struct {
	repo       *Repo
	tokens     *TokenService
	bcryptCost int
}

func NewService(repo *Repo, tokens *TokenService, bcryptCost int) *Service {
	return &Service{repo: repo, tokens: tokens, bcryptCost: bcryptCost}
}

func (s *Service) Tokens() *TokenService { return s.tokens }

// Register creates a new customer account and returns a session.
func (s *Service) Register(ctx context.Context, req RegisterRequest, ua, ip string) (*AuthResponse, error) {
	name := strings.TrimSpace(req.Name)
	email := normalizeEmail(req.Email)
	if name == "" {
		return nil, httpx.BadRequest("Name is required.")
	}
	if !validEmail(email) {
		return nil, httpx.BadRequest("A valid email is required.")
	}
	if len(req.Password) < 8 {
		return nil, httpx.BadRequest("Password must be at least 8 characters.")
	}

	exists, err := s.repo.EmailExists(ctx, email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, httpx.Conflict("An account with this email already exists.")
	}

	hash, err := HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}
	user, err := s.repo.CreateUser(ctx, email, hash, name, "customer")
	if err != nil {
		return nil, err
	}
	return s.issue(ctx, user, ua, ip)
}

// Login authenticates by email/password and returns a session.
func (s *Service) Login(ctx context.Context, req LoginRequest, ua, ip string) (*AuthResponse, error) {
	email := normalizeEmail(req.Email)
	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.Unauthorized("Invalid email or password.")
		}
		return nil, err
	}
	if user.Status != "active" {
		return nil, httpx.Forbidden("This account is disabled.")
	}
	if !CheckPassword(user.PasswordHash, req.Password) {
		return nil, httpx.Unauthorized("Invalid email or password.")
	}
	_ = s.repo.TouchLogin(ctx, user.ID)
	return s.issue(ctx, user, ua, ip)
}

// Refresh rotates the refresh token and returns a fresh session.
func (s *Service) Refresh(ctx context.Context, refreshToken, ua, ip string) (*AuthResponse, error) {
	if refreshToken == "" {
		return nil, httpx.Unauthorized("Missing refresh token.")
	}
	hash := HashToken(refreshToken)
	user, err := s.repo.SessionUser(ctx, hash)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.Unauthorized("Invalid or expired session.")
		}
		return nil, err
	}
	// Rotate: revoke the presented token, then mint a new pair.
	_ = s.repo.RevokeSession(ctx, hash)
	return s.issue(ctx, user, ua, ip)
}

// Logout revokes the given refresh token.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.repo.RevokeSession(ctx, HashToken(refreshToken))
}

// Me returns the current user by id.
func (s *Service) Me(ctx context.Context, id uuid.UUID) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("User not found.")
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) issue(ctx context.Context, user *User, ua, ip string) (*AuthResponse, error) {
	access, exp, err := s.tokens.IssueAccessToken(user.ID, user.Role)
	if err != nil {
		return nil, err
	}
	raw, hash, err := NewRefreshToken()
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateSession(ctx, user.ID, hash, ua, ip, time.Now().Add(s.tokens.RefreshTTL())); err != nil {
		return nil, err
	}
	return &AuthResponse{User: user, AccessToken: access, RefreshToken: raw, ExpiresAt: exp.Unix()}, nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validEmail(e string) bool {
	if len(e) < 4 {
		return false
	}
	_, err := mail.ParseAddress(e)
	return err == nil
}
