package admin

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/httpx"
)

var (
	hexRe         = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
	validGarments = map[string]bool{"shirt": true, "sweater": true, "hoodie": true, "cap": true}
)

// Service holds admin business logic and validation.
type Service struct{ repo *Repo }

func NewService(repo *Repo) *Service { return &Service{repo: repo} }

func (s *Service) ListProducts(ctx context.Context) ([]AdminProduct, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) CreateProduct(ctx context.Context, in ProductInput, adminID uuid.UUID) (uuid.UUID, error) {
	if err := validateProduct(in); err != nil {
		return uuid.Nil, err
	}
	return s.repo.CreateProduct(ctx, in, adminID)
}

func (s *Service) UpdateProduct(ctx context.Context, id uuid.UUID, in ProductInput, adminID uuid.UUID) error {
	if err := validateProduct(in); err != nil {
		return err
	}
	return mapNotFound(s.repo.UpdateProduct(ctx, id, in, adminID), "Product not found.")
}

func (s *Service) SetProductStatus(ctx context.Context, id uuid.UUID, status string, adminID uuid.UUID) error {
	return mapNotFound(s.repo.SetProductStatus(ctx, id, status, adminID), "Product not found.")
}

func (s *Service) DeleteProduct(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	return mapNotFound(s.repo.SoftDeleteProduct(ctx, id, adminID), "Product not found.")
}

func (s *Service) ListCollections(ctx context.Context) ([]AdminCollection, error) {
	return s.repo.ListCollections(ctx)
}

func (s *Service) CreateCollection(ctx context.Context, in CollectionInput, adminID uuid.UUID) (uuid.UUID, error) {
	if strings.TrimSpace(in.Name) == "" {
		return uuid.Nil, httpx.BadRequest("Collection name is required.")
	}
	return s.repo.CreateCollection(ctx, in, adminID)
}

func (s *Service) UpdateCollection(ctx context.Context, id uuid.UUID, in CollectionInput, adminID uuid.UUID) error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.BadRequest("Collection name is required.")
	}
	return mapNotFound(s.repo.UpdateCollection(ctx, id, in, adminID), "Collection not found.")
}

func (s *Service) DeleteCollection(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	return mapNotFound(s.repo.SoftDeleteCollection(ctx, id, adminID), "Collection not found.")
}

func validateProduct(in ProductInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return httpx.BadRequest("Product name is required.")
	}
	if !validGarments[in.Garment] {
		return httpx.BadRequest("Garment must be one of shirt, sweater, hoodie, cap.")
	}
	if !hexRe.MatchString(in.ColorHex) {
		return httpx.BadRequest("colorHex must be a #RRGGBB value.")
	}
	if in.PriceMajor < 0 {
		return httpx.BadRequest("Price must be non-negative.")
	}
	if len(in.Sizes) == 0 {
		return httpx.BadRequest("At least one size is required.")
	}
	return nil
}

func mapNotFound(err error, msg string) error {
	if errors.Is(err, ErrNotFound) {
		return httpx.NotFound(msg)
	}
	return err
}
