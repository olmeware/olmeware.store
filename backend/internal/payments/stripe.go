package payments

import (
	"fmt"
	"strings"

	stripe "github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/paymentintent"
	"github.com/stripe/stripe-go/v79/webhook"
)

// stripeClient wraps the Stripe SDK for the operations we need.
type stripeClient struct {
	secretKey     string
	webhookSecret string
}

func newStripeClient(secretKey, webhookSecret string) *stripeClient {
	if secretKey != "" {
		stripe.Key = secretKey
	}
	return &stripeClient{secretKey: secretKey, webhookSecret: webhookSecret}
}

type stripeIntent struct {
	ID           string
	ClientSecret string
	Status       string
}

// createIntent creates a PaymentIntent. The idempotencyKey guarantees that
// repeated calls (e.g. a double-clicked pay button) return the same intent and
// never create a second charge.
func (c *stripeClient) createIntent(amountMinor int64, currency, orderID, idempotencyKey string) (*stripeIntent, error) {
	params := &stripe.PaymentIntentParams{
		Amount:   stripe.Int64(amountMinor),
		Currency: stripe.String(strings.ToLower(currency)),
		AutomaticPaymentMethods: &stripe.PaymentIntentAutomaticPaymentMethodsParams{
			Enabled: stripe.Bool(true),
		},
	}
	params.AddMetadata("order_id", orderID)
	params.IdempotencyKey = stripe.String(idempotencyKey)

	pi, err := paymentintent.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe create intent: %w", err)
	}
	return &stripeIntent{ID: pi.ID, ClientSecret: pi.ClientSecret, Status: string(pi.Status)}, nil
}

// retrieveIntent fetches an existing PaymentIntent (used to return a fresh
// client secret when a payment row already exists).
func (c *stripeClient) retrieveIntent(id string) (*stripeIntent, error) {
	pi, err := paymentintent.Get(id, nil)
	if err != nil {
		return nil, fmt.Errorf("stripe get intent: %w", err)
	}
	return &stripeIntent{ID: pi.ID, ClientSecret: pi.ClientSecret, Status: string(pi.Status)}, nil
}

// verifyWebhook validates the Stripe-Signature header against the raw body.
func (c *stripeClient) verifyWebhook(payload []byte, sigHeader string) (stripe.Event, error) {
	return webhook.ConstructEvent(payload, sigHeader, c.webhookSecret)
}
