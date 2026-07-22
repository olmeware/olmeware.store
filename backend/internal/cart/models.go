package cart

import "github.com/google/uuid"

// Item is a composed cart line for the storefront.
type Item struct {
	VariantID      uuid.UUID `json:"variantId"`
	ProductID      uuid.UUID `json:"productId"`
	Slug           string    `json:"slug"`
	Name           string    `json:"name"`
	Tech           string    `json:"tech"`
	Garment        string    `json:"garment"`
	Logo           string    `json:"logo,omitempty"`
	ColorHex       string    `json:"colorHex"`
	Size           string    `json:"size"`
	SKU            string    `json:"sku"`
	UnitPriceMinor int64     `json:"unitPriceMinor"`
	UnitPrice      string    `json:"unitPrice"`
	Quantity       int       `json:"quantity"`
	LineTotalMinor int64     `json:"lineTotalMinor"`
	LineTotal      string    `json:"lineTotal"`
	Available      int       `json:"available"`
	InStock        bool      `json:"inStock"`
}

// Cart is the composed cart response with totals.
type Cart struct {
	ID            uuid.UUID `json:"id"`
	Currency      string    `json:"currency"`
	Items         []Item    `json:"items"`
	ItemCount     int       `json:"itemCount"`
	SubtotalMinor int64     `json:"subtotalMinor"`
	Subtotal      string    `json:"subtotal"`
	// GuestToken is set (once) when a new guest cart is created, so the client
	// can persist it and address the same cart on later requests.
	GuestToken string `json:"guestToken,omitempty"`
}

// AddItemRequest adds or increments a variant in the cart.
type AddItemRequest struct {
	VariantID uuid.UUID `json:"variantId"`
	Quantity  int       `json:"quantity"`
}

// SetQtyRequest sets an absolute quantity (0 removes the line).
type SetQtyRequest struct {
	Quantity int `json:"quantity"`
}
