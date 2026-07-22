package orders

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

// Service holds order business logic.
type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

// Checkout identifies who is placing the order.
type Checkout struct {
	UserID     uuid.UUID // uuid.Nil for guests
	GuestToken string
}

// Create converts the caller's active cart into a pending order.
func (s *Service) Create(ctx context.Context, c Checkout, req CreateOrderRequest) (*Order, error) {
	if err := validate(req); err != nil {
		return nil, err
	}
	if req.ShippingAddress.CountryCode == "" {
		req.ShippingAddress.CountryCode = "MX"
	}

	cartID, err := s.resolveCart(ctx, c)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.BadRequest("No active cart to check out.")
		}
		return nil, err
	}

	var userID *uuid.UUID
	if c.UserID != uuid.Nil {
		userID = &c.UserID
	}

	order, err := s.repo.CreateFromCart(ctx, cartID, userID, req)
	switch {
	case errors.Is(err, ErrEmptyCart):
		return nil, httpx.BadRequest("Your cart is empty.")
	case errors.Is(err, ErrOutOfStock):
		return nil, httpx.Conflict("One or more items are out of stock.")
	case err != nil:
		return nil, err
	}
	return order, nil
}

// Get returns an order; non-admins may only read their own.
func (s *Service) Get(ctx context.Context, id uuid.UUID, viewerID uuid.UUID, isAdmin bool) (*Order, error) {
	var owner *uuid.UUID
	if !isAdmin {
		owner = &viewerID
	}
	order, err := s.repo.GetByID(ctx, id, owner)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Order not found.")
		}
		return nil, err
	}
	return order, nil
}

// List returns the viewer's orders.
func (s *Service) List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Order, error) {
	return s.repo.ListForUser(ctx, userID, limit, offset)
}

func (s *Service) resolveCart(ctx context.Context, c Checkout) (uuid.UUID, error) {
	if c.UserID != uuid.Nil {
		return s.repo.ActiveCartIDForUser(ctx, c.UserID)
	}
	if c.GuestToken != "" {
		sum := sha256.Sum256([]byte(c.GuestToken))
		return s.repo.ActiveCartIDForGuest(ctx, hex.EncodeToString(sum[:]))
	}
	return uuid.Nil, ErrNotFound
}

func validate(req CreateOrderRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return httpx.BadRequest("Name is required.")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(req.Email)); err != nil {
		return httpx.BadRequest("A valid email is required.")
	}
	a := req.ShippingAddress
	if strings.TrimSpace(a.RecipientName) == "" || strings.TrimSpace(a.Line1) == "" ||
		strings.TrimSpace(a.City) == "" || strings.TrimSpace(a.State) == "" ||
		strings.TrimSpace(a.PostalCode) == "" {
		return httpx.BadRequest("A complete shipping address is required.")
	}
	return nil
}
