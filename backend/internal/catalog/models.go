package catalog

import "github.com/google/uuid"

// Variant is a purchasable size option of a product.
type Variant struct {
	ID         uuid.UUID `json:"id"`
	SKU        string    `json:"sku"`
	Size       string    `json:"size"`
	ColorHex   string    `json:"colorHex"`
	ColorName  string    `json:"colorName,omitempty"`
	PriceMinor int64     `json:"priceMinor"`
	InStock    bool      `json:"inStock"`
	Available  int       `json:"available"`
}

// Product is the storefront presentation of a product, fully composed by the
// backend so the frontend only renders it.
type Product struct {
	ID          uuid.UUID `json:"id"`
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Garment     string    `json:"garment"`
	Stack       string    `json:"stack"`
	Tech        string    `json:"tech"`
	Logo        string    `json:"logo,omitempty"`
	ColorHex    string    `json:"colorHex"`
	PriceMinor  int64     `json:"priceMinor"`
	Price       string    `json:"price"`
	Currency    string    `json:"currency"`
	Featured    bool      `json:"featured"`
	Status      string    `json:"status"`
	Sizes       []string  `json:"sizes"`
	Images      []string  `json:"images"`
	Variants    []Variant `json:"variants,omitempty"`
	Collections []string  `json:"collections"`
}

// Collection groups products on the storefront.
type Collection struct {
	ID           uuid.UUID `json:"id"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	ProductCount int       `json:"productCount"`
}

// TechTheme is a design theme (logo) available for merch.
type TechTheme struct {
	ID       uuid.UUID `json:"id"`
	Slug     string    `json:"slug"`
	Name     string    `json:"name"`
	Category string    `json:"category"`
	Logo     string    `json:"logo,omitempty"`
}

// ProductFilters captures storefront catalog query parameters.
type ProductFilters struct {
	Garment    string
	Stack      string
	Size       string
	Collection string
	Search     string
	Featured   *bool
	MinMinor   *int64
	MaxMinor   *int64
	Sort       string // newest | price_asc | price_desc | name
	Limit      int
	Offset     int
}
