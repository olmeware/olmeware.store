package cart

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes cart endpoints for guests and authenticated users.
type Handler struct {
	svc    *Service
	tokens *auth.TokenService
}

func NewHandler(svc *Service, tokens *auth.TokenService) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// Register wires cart routes. OptionalAuth attaches a user when a token is
// present but still allows guest carts (addressed by X-Guest-Token).
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	opt := h.tokens.OptionalAuth
	mux.Handle("GET "+prefix+"/cart", opt(http.HandlerFunc(h.get)))
	mux.Handle("POST "+prefix+"/cart/items", opt(http.HandlerFunc(h.add)))
	mux.Handle("PUT "+prefix+"/cart/items/{variantId}", opt(http.HandlerFunc(h.setQty)))
	mux.Handle("DELETE "+prefix+"/cart/items/{variantId}", opt(http.HandlerFunc(h.remove)))
	mux.Handle("DELETE "+prefix+"/cart", opt(http.HandlerFunc(h.clear)))
}

func (h *Handler) owner(r *http.Request) Owner {
	o := Owner{GuestToken: r.Header.Get("X-Guest-Token")}
	if p, ok := auth.FromContext(r.Context()); ok {
		o.UserID = p.UserID
	}
	return o
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	cart, err := h.svc.Get(r.Context(), h.owner(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cart)
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	var req AddItemRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	cart, err := h.svc.Add(r.Context(), h.owner(r), req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cart)
}

func (h *Handler) setQty(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(r.PathValue("variantId"))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Invalid variant id."))
		return
	}
	var req SetQtyRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	cart, err := h.svc.SetQty(r.Context(), h.owner(r), variantID, req.Quantity)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cart)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	variantID, err := uuid.Parse(r.PathValue("variantId"))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Invalid variant id."))
		return
	}
	cart, err := h.svc.Remove(r.Context(), h.owner(r), variantID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cart)
}

func (h *Handler) clear(w http.ResponseWriter, r *http.Request) {
	cart, err := h.svc.Clear(r.Context(), h.owner(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, cart)
}
