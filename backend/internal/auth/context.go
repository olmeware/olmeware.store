package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

type ctxKey int

const principalKey ctxKey = iota

// Principal is the authenticated identity attached to a request context.
type Principal struct {
	UserID uuid.UUID
	Role   string
}

// IsAdmin reports whether the principal is the store administrator.
func (p Principal) IsAdmin() bool { return p.Role == "admin" }

// FromContext returns the authenticated principal, if any.
func FromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey).(Principal)
	return p, ok
}

// RequireAuth is middleware that rejects requests without a valid access token.
func (s *TokenService) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := s.principalFromRequest(r)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireAdmin is middleware that requires an authenticated admin.
func (s *TokenService) RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, err := s.principalFromRequest(r)
		if err != nil {
			httpx.Error(w, err)
			return
		}
		if !p.IsAdmin() {
			httpx.Error(w, httpx.Forbidden("Admin access required."))
			return
		}
		ctx := context.WithValue(r.Context(), principalKey, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OptionalAuth attaches a principal when a valid token is present but never
// rejects the request. Useful for endpoints that behave differently for guests.
func (s *TokenService) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, err := s.principalFromRequest(r); err == nil {
			r = r.WithContext(context.WithValue(r.Context(), principalKey, p))
		}
		next.ServeHTTP(w, r)
	})
}

func (s *TokenService) principalFromRequest(r *http.Request) (Principal, error) {
	header := r.Header.Get("Authorization")
	if header == "" || !strings.HasPrefix(header, "Bearer ") {
		return Principal{}, httpx.Unauthorized("Missing or malformed Authorization header.")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	claims, err := s.ParseAccessToken(raw)
	if err != nil {
		return Principal{}, httpx.Unauthorized("Invalid or expired token.")
	}
	uid, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Principal{}, httpx.Unauthorized("Invalid token subject.")
	}
	return Principal{UserID: uid, Role: claims.Role}, nil
}
