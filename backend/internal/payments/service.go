package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/google/uuid"
	stripe "github.com/stripe/stripe-go/v79"

	"github.com/olmeware/backend/internal/httpx"
)

// Viewer is the authenticated (or guest) caller placing/paying an order.
type Viewer struct {
	UserID  uuid.UUID // uuid.Nil for guests
	IsAdmin bool
}

// Service orchestrates card (Stripe) and crypto (Coinbase Commerce) payments.
type Service struct {
	repo           *Repo
	stripe         *stripeClient
	coinbase       *coinbaseClient
	publishableKey string
	frontendURL    string
	stripeOn       bool
	coinbaseOn     bool
}

// Deps configures the payment service.
type Deps struct {
	Repo            *Repo
	StripeSecret    string
	StripePublish   string
	StripeWebhook   string
	CoinbaseKey     string
	CoinbaseWebhook string
	FrontendURL     string
	StripeEnabled   bool
	CoinbaseEnabled bool
}

func NewService(d Deps) *Service {
	return &Service{
		repo:           d.Repo,
		stripe:         newStripeClient(d.StripeSecret, d.StripeWebhook),
		coinbase:       newCoinbaseClient(d.CoinbaseKey, d.CoinbaseWebhook),
		publishableKey: d.StripePublish,
		frontendURL:    d.FrontendURL,
		stripeOn:       d.StripeEnabled,
		coinbaseOn:     d.CoinbaseEnabled,
	}
}

var errUnavailable = httpx.NewError(503, "payments_unavailable", "This payment method is not configured.")

// CreateStripeIntent creates (or returns) an idempotent PaymentIntent for an order.
func (s *Service) CreateStripeIntent(ctx context.Context, orderID uuid.UUID, v Viewer) (*StripeIntentResponse, error) {
	if !s.stripeOn {
		return nil, errUnavailable
	}
	order, err := s.loadPayable(ctx, orderID, v)
	if err != nil {
		return nil, err
	}

	idem := "stripe_order_" + orderID.String()

	// Reuse an existing intent for this order if present.
	if pid, intentID, _, e := s.repo.findPayment(ctx, "stripe", idem); e == nil {
		if intentID != "" {
			if intent, ierr := s.stripe.retrieveIntent(intentID); ierr == nil {
				return &StripeIntentResponse{
					PaymentID: pid, PaymentIntentID: intent.ID, ClientSecret: intent.ClientSecret,
					PublishableKey: s.publishableKey, Amount: order.TotalMinor,
					Currency: order.Currency, Status: intent.Status, Reused: true,
				}, nil
			}
		}
	}

	intent, err := s.stripe.createIntent(order.TotalMinor, order.Currency, orderID.String(), idem)
	if err != nil {
		return nil, err
	}
	paymentID, created, err := s.repo.insertOrGetPayment(ctx, orderID, "stripe",
		order.TotalMinor, order.Currency, idem, intent.ID)
	if err != nil {
		return nil, err
	}
	if !created {
		_ = s.repo.setPaymentIntent(ctx, paymentID, intent.ID)
	}
	return &StripeIntentResponse{
		PaymentID: paymentID, PaymentIntentID: intent.ID, ClientSecret: intent.ClientSecret,
		PublishableKey: s.publishableKey, Amount: order.TotalMinor, Currency: order.Currency,
		Status: intent.Status, Reused: !created,
	}, nil
}

// HandleStripeWebhook verifies and processes a Stripe event exactly once.
func (s *Service) HandleStripeWebhook(ctx context.Context, payload []byte, sig string) error {
	event, err := s.stripe.verifyWebhook(payload, sig)
	if err != nil {
		return httpx.BadRequest("Invalid Stripe signature.")
	}
	isNew, err := s.repo.insertStripeEvent(ctx, event.ID, string(event.Type),
		event.APIVersion, event.Livemode, payload)
	if err != nil {
		return err
	}
	if !isNew {
		return nil // already processed
	}

	if err := s.applyStripeEvent(ctx, event); err != nil {
		s.repo.markStripeError(ctx, event.ID, err.Error())
		return err
	}
	s.repo.markStripeProcessed(ctx, event.ID)
	return nil
}

func (s *Service) applyStripeEvent(ctx context.Context, event stripe.Event) error {
	switch event.Type {
	case "payment_intent.succeeded":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return err
		}
		chargeID := ""
		if pi.LatestCharge != nil {
			chargeID = pi.LatestCharge.ID
		}
		log.Printf("payment: stripe intent %s succeeded", pi.ID)
		return s.repo.markSucceededByIntent(ctx, "stripe", pi.ID, chargeID)
	case "payment_intent.payment_failed":
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return err
		}
		code, msg := "", ""
		if pi.LastPaymentError != nil {
			code, msg = string(pi.LastPaymentError.Code), pi.LastPaymentError.Msg
		}
		log.Printf("payment: stripe intent %s failed: %s", pi.ID, msg)
		return s.repo.markFailedByIntent(ctx, "stripe", pi.ID, code, msg)
	default:
		return nil // ignore unrelated events
	}
}

// CreateCryptoCharge creates (or returns) an idempotent Coinbase Commerce charge.
func (s *Service) CreateCryptoCharge(ctx context.Context, orderID uuid.UUID, v Viewer) (*CryptoChargeResponse, error) {
	if !s.coinbaseOn {
		return nil, errUnavailable
	}
	order, err := s.loadPayable(ctx, orderID, v)
	if err != nil {
		return nil, err
	}

	idem := "coinbase_order_" + orderID.String()
	if pid, code, _, e := s.repo.findPayment(ctx, "coinbase", idem); e == nil && code != "" {
		return &CryptoChargeResponse{
			PaymentID: pid, ChargeCode: code, HostedURL: hostedURL(code),
			Amount: order.TotalMinor, Currency: order.Currency, Status: "pending",
			Reused: true, Coins: SupportedCoins,
		}, nil
	}

	amount := fmt.Sprintf("%d.%02d", order.TotalMinor/100, order.TotalMinor%100)
	desc := fmt.Sprintf("Olmeware order #%d", order.OrderNumber)
	charge, err := s.coinbase.createCharge(ctx, "Olmeware Store", desc, amount, order.Currency,
		orderID.String(), s.frontendURL+"/orders", s.frontendURL+"/cart")
	if err != nil {
		return nil, err
	}
	paymentID, created, err := s.repo.insertOrGetPayment(ctx, orderID, "coinbase",
		order.TotalMinor, order.Currency, idem, charge.Code)
	if err != nil {
		return nil, err
	}
	hosted := charge.HostedURL
	if hosted == "" {
		hosted = hostedURL(charge.Code)
	}
	return &CryptoChargeResponse{
		PaymentID: paymentID, ChargeCode: charge.Code, HostedURL: hosted,
		Amount: order.TotalMinor, Currency: order.Currency, Status: charge.Status,
		Reused: !created, Coins: SupportedCoins,
	}, nil
}

// HandleCryptoWebhook verifies and processes a Coinbase event exactly once.
func (s *Service) HandleCryptoWebhook(ctx context.Context, payload []byte, sig string) error {
	if !s.coinbase.verifyWebhook(payload, sig) {
		return httpx.BadRequest("Invalid Coinbase signature.")
	}
	event, err := parseCoinbaseEvent(payload)
	if err != nil {
		return httpx.BadRequest("Malformed Coinbase payload.")
	}
	code := event.Event.Data.Code
	isNew, err := s.repo.insertCryptoEvent(ctx, event.Event.ID, event.Event.Type, code, payload)
	if err != nil {
		return err
	}
	if !isNew {
		return nil
	}

	var perr error
	switch event.Event.Type {
	case "charge:confirmed", "charge:resolved":
		log.Printf("payment: coinbase charge %s confirmed", code)
		perr = s.repo.markSucceededByIntent(ctx, "coinbase", code, "")
	case "charge:failed":
		log.Printf("payment: coinbase charge %s failed", code)
		perr = s.repo.markFailedByIntent(ctx, "coinbase", code, event.Event.Type, "")
	}
	if perr != nil {
		s.repo.markCryptoError(ctx, event.Event.ID, perr.Error())
		return perr
	}
	s.repo.markCryptoProcessed(ctx, event.Event.ID)
	return nil
}

// loadPayable loads an order and checks it is payable by the viewer.
func (s *Service) loadPayable(ctx context.Context, orderID uuid.UUID, v Viewer) (*orderForPayment, error) {
	order, err := s.repo.getOrder(ctx, orderID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, httpx.NotFound("Order not found.")
		}
		return nil, err
	}
	// Owned orders require the owner or an admin; guest orders (no user) are
	// reachable only by their unguessable id.
	if order.UserID != nil && !v.IsAdmin && *order.UserID != v.UserID {
		return nil, httpx.Forbidden("You cannot pay for this order.")
	}
	switch order.Status {
	case "pending_payment":
		return order, nil
	case "paid", "processing", "shipped", "delivered":
		return nil, httpx.Conflict("This order is already paid.")
	default:
		return nil, httpx.Conflict("This order cannot be paid.")
	}
}

func hostedURL(code string) string { return "https://commerce.coinbase.com/charges/" + code }
