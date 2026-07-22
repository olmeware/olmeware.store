package payments

import (
	"io"
	"net/http"

	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes payment endpoints.
type Handler struct {
	svc    *Service
	tokens *auth.TokenService
}

func NewHandler(svc *Service, tokens *auth.TokenService) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// Register wires payment routes. Intent/charge creation allows guests
// (OptionalAuth); webhooks are unauthenticated but signature-verified.
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	p := prefix + "/payments"
	mux.HandleFunc("GET "+p+"/config", h.config)
	mux.Handle("POST "+p+"/stripe/intent", h.tokens.OptionalAuth(http.HandlerFunc(h.stripeIntent)))
	mux.HandleFunc("POST "+p+"/stripe/webhook", h.stripeWebhook)
	mux.Handle("POST "+p+"/crypto/charge", h.tokens.OptionalAuth(http.HandlerFunc(h.cryptoCharge)))
	mux.HandleFunc("POST "+p+"/crypto/webhook", h.cryptoWebhook)
}

func (h *Handler) viewer(r *http.Request) Viewer {
	var v Viewer
	if p, ok := auth.FromContext(r.Context()); ok {
		v.UserID = p.UserID
		v.IsAdmin = p.IsAdmin()
	}
	return v
}

func (h *Handler) config(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"stripe":   map[string]any{"enabled": h.svc.stripeOn, "publishableKey": h.svc.publishableKey},
		"crypto":   map[string]any{"enabled": h.svc.coinbaseOn, "coins": SupportedCoins},
		"currency": "MXN",
	})
}

func (h *Handler) stripeIntent(w http.ResponseWriter, r *http.Request) {
	var req StripeIntentRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	resp, err := h.svc.CreateStripeIntent(r.Context(), req.OrderID, h.viewer(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) cryptoCharge(w http.ResponseWriter, r *http.Request) {
	var req CryptoChargeRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	resp, err := h.svc.CreateCryptoCharge(r.Context(), req.OrderID, h.viewer(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) stripeWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := readBody(w, r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.HandleStripeWebhook(r.Context(), payload, r.Header.Get("Stripe-Signature")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"received": true})
}

func (h *Handler) cryptoWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := readBody(w, r)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.HandleCryptoWebhook(r.Context(), payload, r.Header.Get("X-CC-Webhook-Signature")); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"received": true})
}

// readBody returns the raw request body (needed for webhook signature checks).
func readBody(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	body := http.MaxBytesReader(w, r.Body, 1<<20)
	payload, err := io.ReadAll(body)
	if err != nil {
		return nil, httpx.BadRequest("Could not read request body.")
	}
	return payload, nil
}
