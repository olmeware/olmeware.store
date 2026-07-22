package payments

import "github.com/google/uuid"

// orderForPayment is the minimal order view needed to take payment.
type orderForPayment struct {
	ID          uuid.UUID
	Status      string
	TotalMinor  int64
	Currency    string
	UserID      *uuid.UUID
	Email       string
	Name        string
	OrderNumber int64
}

// StripeIntentRequest asks to create/return a PaymentIntent for an order.
type StripeIntentRequest struct {
	OrderID uuid.UUID `json:"orderId"`
}

// StripeIntentResponse is returned to the client to confirm the card payment.
type StripeIntentResponse struct {
	PaymentID       uuid.UUID `json:"paymentId"`
	PaymentIntentID string    `json:"paymentIntentId"`
	ClientSecret    string    `json:"clientSecret"`
	PublishableKey  string    `json:"publishableKey"`
	Amount          int64     `json:"amountMinor"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`
	Reused          bool      `json:"reused"`
}

// CryptoChargeRequest asks to create/return a crypto charge for an order.
type CryptoChargeRequest struct {
	OrderID uuid.UUID `json:"orderId"`
}

// CryptoChargeResponse points the client at the hosted crypto checkout.
type CryptoChargeResponse struct {
	PaymentID  uuid.UUID `json:"paymentId"`
	ChargeCode string    `json:"chargeCode"`
	HostedURL  string    `json:"hostedUrl"`
	Amount     int64     `json:"amountMinor"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	Reused     bool      `json:"reused"`
	Coins      []string  `json:"coins"`
}
