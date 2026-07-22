package admin

import (
	"time"

	"github.com/google/uuid"
)

// ProductInput is the create/update payload for a product.
type ProductInput struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Garment        string   `json:"garment"` // shirt | sweater | hoodie | cap
	Stack          string   `json:"stack"`
	Tech           string   `json:"tech"`
	LogoSlug       string   `json:"logoSlug"` // e.g. "python" -> /logos/python.svg
	PriceMajor     int      `json:"price"`    // MXN major units
	ColorHex       string   `json:"colorHex"`
	Sizes          []string `json:"sizes"`
	CollectionSlug string   `json:"collectionSlug,omitempty"`
	Featured       bool     `json:"featured"`
	Status         string   `json:"status,omitempty"` // draft | active | archived
}

// AdminProduct is a product row as shown in the admin table. It carries the
// full presentation (all statuses) so the admin UI can render mockups.
type AdminProduct struct {
	ID           uuid.UUID `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	Garment      string    `json:"garment"`
	Stack        string    `json:"stack"`
	Tech         string    `json:"tech"`
	Logo         string    `json:"logo,omitempty"`
	ColorHex     string    `json:"colorHex"`
	Status       string    `json:"status"`
	Featured     bool      `json:"featured"`
	PriceMinor   int64     `json:"priceMinor"`
	Price        string    `json:"price"`
	Sizes        []string  `json:"sizes"`
	Images       []string  `json:"images"`
	Collections  []string  `json:"collections"`
	VariantCount int       `json:"variantCount"`
	CreatedAt    time.Time `json:"createdAt"`
}

// CollectionInput is the create/update payload for a collection.
type CollectionInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	SortOrder   int    `json:"sortOrder"`
}

// AdminCollection is a collection row for the admin panel.
type AdminCollection struct {
	ID           uuid.UUID `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	SortOrder    int       `json:"sortOrder"`
	ProductCount int       `json:"productCount"`
	CreatedAt    time.Time `json:"createdAt"`
}
