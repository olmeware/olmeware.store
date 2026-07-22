package admin

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes admin-only endpoints (product & collection management).
type Handler struct {
	svc    *Service
	tokens *auth.TokenService
}

func NewHandler(svc *Service, tokens *auth.TokenService) *Handler {
	return &Handler{svc: svc, tokens: tokens}
}

// Register wires admin routes, each guarded by RequireAdmin.
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	admin := h.tokens.RequireAdmin
	p := prefix + "/admin"
	mux.Handle("GET "+p+"/products", admin(http.HandlerFunc(h.listProducts)))
	mux.Handle("POST "+p+"/products", admin(http.HandlerFunc(h.createProduct)))
	mux.Handle("PUT "+p+"/products/{id}", admin(http.HandlerFunc(h.updateProduct)))
	mux.Handle("PATCH "+p+"/products/{id}/status", admin(http.HandlerFunc(h.setProductStatus)))
	mux.Handle("DELETE "+p+"/products/{id}", admin(http.HandlerFunc(h.deleteProduct)))
	mux.Handle("GET "+p+"/collections", admin(http.HandlerFunc(h.listCollections)))
	mux.Handle("POST "+p+"/collections", admin(http.HandlerFunc(h.createCollection)))
	mux.Handle("PUT "+p+"/collections/{id}", admin(http.HandlerFunc(h.updateCollection)))
	mux.Handle("DELETE "+p+"/collections/{id}", admin(http.HandlerFunc(h.deleteCollection)))
}

func (h *Handler) adminID(r *http.Request) uuid.UUID {
	p, _ := auth.FromContext(r.Context())
	return p.UserID
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListProducts(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"products": list})
}

func (h *Handler) createProduct(w http.ResponseWriter, r *http.Request) {
	var in ProductInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	id, err := h.svc.CreateProduct(r.Context(), in, h.adminID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *Handler) updateProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in ProductInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.UpdateProduct(r.Context(), id, in, h.adminID(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) setProductStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var body struct {
		Status string `json:"status"`
	}
	if err := httpx.Decode(w, r, &body); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.SetProductStatus(r.Context(), id, body.Status, h.adminID(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) deleteProduct(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteProduct(r.Context(), id, h.adminID(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) listCollections(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListCollections(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"collections": list})
}

func (h *Handler) createCollection(w http.ResponseWriter, r *http.Request) {
	var in CollectionInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	id, err := h.svc.CreateCollection(r.Context(), in, h.adminID(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

func (h *Handler) updateCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in CollectionInput
	if err := httpx.Decode(w, r, &in); err != nil {
		httpx.Error(w, err)
		return
	}
	if err := h.svc.UpdateCollection(r.Context(), id, in, h.adminID(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) deleteCollection(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	if err := h.svc.DeleteCollection(r.Context(), id, h.adminID(r)); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpx.Error(w, httpx.BadRequest("Invalid id."))
		return uuid.Nil, false
	}
	return id, true
}
