package orders

import (
	"time"

	"github.com/google/uuid"
)

// Address is a shipping or billing address captured at checkout.
type Address struct {
	RecipientName string `json:"recipientName"`
	Phone         string `json:"phone,omitempty"`
	Line1         string `json:"line1"`
	Line2         string `json:"line2,omitempty"`
	Neighborhood  string `json:"neighborhood,omitempty"`
	City          string `json:"city"`
	State         string `json:"state"`
	PostalCode    string `json:"postalCode"`
	CountryCode   string `json:"countryCode"`
}

// CreateOrderRequest is the checkout payload.
type CreateOrderRequest struct {
	Email           string   `json:"email"`
	Name            string   `json:"name"`
	Phone           string   `json:"phone,omitempty"`
	ShippingAddress Address  `json:"shippingAddress"`
	BillingAddress  *Address `json:"billingAddress,omitempty"`
	Note            string   `json:"note,omitempty"`
}

// OrderItem is a line captured on the order.
type OrderItem struct {
	ID             uuid.UUID `json:"id"`
	ProductID      uuid.UUID `json:"productId"`
	SKU            string    `json:"sku"`
	ProductName    string    `json:"productName"`
	Garment        string    `json:"garment"`
	Tech           string    `json:"tech"`
	Size           string    `json:"size"`
	ColorHex       string    `json:"colorHex"`
	Logo           string    `json:"logo,omitempty"`
	UnitPriceMinor int64     `json:"unitPriceMinor"`
	UnitPrice      string    `json:"unitPrice"`
	Quantity       int       `json:"quantity"`
	LineTotalMinor int64     `json:"lineTotalMinor"`
	LineTotal      string    `json:"lineTotal"`
}

// Order is the composed order response.
type Order struct {
	ID            uuid.UUID   `json:"id"`
	OrderNumber   int64       `json:"orderNumber"`
	Status        string      `json:"status"`
	CustomerEmail string      `json:"customerEmail"`
	CustomerName  string      `json:"customerName"`
	Currency      string      `json:"currency"`
	SubtotalMinor int64       `json:"subtotalMinor"`
	Subtotal      string      `json:"subtotal"`
	ShippingMinor int64       `json:"shippingMinor"`
	TaxMinor      int64       `json:"taxMinor"`
	DiscountMinor int64       `json:"discountMinor"`
	TotalMinor    int64       `json:"totalMinor"`
	Total         string      `json:"total"`
	Items         []OrderItem `json:"items"`
	CreatedAt     time.Time   `json:"createdAt"`
}
