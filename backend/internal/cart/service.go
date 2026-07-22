package cart

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

// Owner identifies who a cart belongs to: an authenticated user or a guest
// addressed by an opaque token.
type Owner struct {
	UserID     uuid.UUID // uuid.Nil for guests
	GuestToken string    // raw token from the X-Guest-Token header (may be empty)
}

func (o Owner) isUser() bool { return o.UserID != uuid.Nil }

func (o Owner) guestHash() string {
	if o.GuestToken == "" {
		return ""
	}
	return hashToken(o.GuestToken)
}

// Service holds cart business logic.
type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// Get returns the owner's active cart, or an empty cart if none exists yet.
func (s *Service) Get(ctx context.Context, o Owner) (*Cart, error) {
	id, err := s.resolveExisting(ctx, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &Cart{Currency: "MXN", Items: []Item{}}, nil
		}
		return nil, err
	}
	return s.repo.Load(ctx, id)
}

// Add adds (increments) a variant, creating the cart if needed. A new guest
// token is returned on the cart when one had to be minted.
func (s *Service) Add(ctx context.Context, o Owner, req AddItemRequest) (*Cart, error) {
	if req.VariantID == uuid.Nil {
		return nil, httpx.BadRequest("variantId is required.")
	}
	qty := req.Quantity
	if qty == 0 {
		qty = 1
	}
	if qty < 0 {
		return nil, httpx.BadRequest("quantity must be positive.")
	}

	available, _, err := s.repo.VariantStock(ctx, req.VariantID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("That product option is unavailable.")
		}
		return nil, err
	}

	cartID, newToken, err := s.resolveOrCreate(ctx, o)
	if err != nil {
		return nil, err
	}

	// Enforce stock against the resulting quantity.
	existing, err := s.repo.Load(ctx, cartID)
	if err != nil {
		return nil, err
	}
	current := 0
	for _, it := range existing.Items {
		if it.VariantID == req.VariantID {
			current = it.Quantity
		}
	}
	if current+qty > available {
		return nil, httpx.Conflict("Not enough stock for that quantity.")
	}

	if err := s.repo.AddOrIncrement(ctx, cartID, req.VariantID, qty); err != nil {
		return nil, err
	}
	cart, err := s.repo.Load(ctx, cartID)
	if err != nil {
		return nil, err
	}
	cart.GuestToken = newToken
	return cart, nil
}

// SetQty sets an absolute quantity for a line (0 removes it).
func (s *Service) SetQty(ctx context.Context, o Owner, variantID uuid.UUID, qty int) (*Cart, error) {
	id, err := s.resolveExisting(ctx, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("No active cart.")
		}
		return nil, err
	}
	if qty > 0 {
		available, _, err := s.repo.VariantStock(ctx, variantID)
		if err == nil && qty > available {
			return nil, httpx.Conflict("Not enough stock for that quantity.")
		}
	}
	if err := s.repo.SetQuantity(ctx, id, variantID, qty); err != nil {
		return nil, err
	}
	return s.repo.Load(ctx, id)
}

// Remove deletes a line from the cart.
func (s *Service) Remove(ctx context.Context, o Owner, variantID uuid.UUID) (*Cart, error) {
	id, err := s.resolveExisting(ctx, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("No active cart.")
		}
		return nil, err
	}
	if err := s.repo.RemoveItem(ctx, id, variantID); err != nil {
		return nil, err
	}
	return s.repo.Load(ctx, id)
}

// Clear empties the cart.
func (s *Service) Clear(ctx context.Context, o Owner) (*Cart, error) {
	id, err := s.resolveExisting(ctx, o)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return &Cart{Currency: "MXN", Items: []Item{}}, nil
		}
		return nil, err
	}
	if err := s.repo.Clear(ctx, id); err != nil {
		return nil, err
	}
	return s.repo.Load(ctx, id)
}

func (s *Service) resolveExisting(ctx context.Context, o Owner) (uuid.UUID, error) {
	if o.isUser() {
		return s.repo.activeCartIDForUser(ctx, o.UserID)
	}
	if h := o.guestHash(); h != "" {
		return s.repo.activeCartIDForGuest(ctx, h)
	}
	return uuid.Nil, ErrNotFound
}

func (s *Service) resolveOrCreate(ctx context.Context, o Owner) (id uuid.UUID, newGuestToken string, err error) {
	if o.isUser() {
		id, err = s.repo.activeCartIDForUser(ctx, o.UserID)
		if errors.Is(err, ErrNotFound) {
			id, err = s.repo.createUserCart(ctx, o.UserID)
		}
		return id, "", err
	}
	if h := o.guestHash(); h != "" {
		id, err = s.repo.activeCartIDForGuest(ctx, h)
		if errors.Is(err, ErrNotFound) {
			id, err = s.repo.createGuestCart(ctx, h)
		}
		return id, "", err
	}
	// Mint a fresh guest token + cart.
	token, err := newGuestTokenValue()
	if err != nil {
		return uuid.Nil, "", err
	}
	id, err = s.repo.createGuestCart(ctx, hashToken(token))
	if err != nil {
		return uuid.Nil, "", err
	}
	return id, token, nil
}

func newGuestTokenValue() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
