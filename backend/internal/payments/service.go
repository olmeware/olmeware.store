package payments

import (
	"context"
	"encoding/json"
	"errors"
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

// Service orchestrates Stripe card payments.
type Service struct {
	repo           *Repo
	stripe         *stripeClient
	publishableKey string
	stripeOn       bool
}

// Deps configures the payment service.
type Deps struct {
	Repo          *Repo
	StripeSecret  string
	StripePublish string
	StripeWebhook string
	StripeEnabled bool
}

func NewService(d Deps) *Service {
	return &Service{
		repo:           d.Repo,
		stripe:         newStripeClient(d.StripeSecret, d.StripeWebhook),
		publishableKey: d.StripePublish,
		stripeOn:       d.StripeEnabled,
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
