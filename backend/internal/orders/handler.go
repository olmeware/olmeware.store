package orders

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes order endpoints.
type Handler struct {
	svc    *Service
	tokens *auth.TokenService
}

func NewHandler(svc *Service, tokens *auth.TokenService) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// Register wires order routes. Checkout allows guests (OptionalAuth); reading
// orders requires authentication.
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	mux.Handle("POST "+prefix+"/orders", h.tokens.OptionalAuth(http.HandlerFunc(h.create)))
	mux.Handle("GET "+prefix+"/orders", h.tokens.RequireAuth(http.HandlerFunc(h.list)))
	mux.Handle("GET "+prefix+"/orders/{id}", h.tokens.RequireAuth(http.HandlerFunc(h.get)))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrderRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	c := Checkout{GuestToken: r.Header.Get("X-Guest-Token")}
	if p, ok := auth.FromContext(r.Context()); ok {
		c.UserID = p.UserID
	}
	order, err := h.svc.Create(r.Context(), c, req)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, order)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.FromContext(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	list, err := h.svc.List(r.Context(), p.UserID, limit, offset)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": list})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Invalid order id."))
		return
	}
	p, _ := auth.FromContext(r.Context())
	order, err := h.svc.Get(r.Context(), id, p.UserID, p.IsAdmin())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, order)
}
