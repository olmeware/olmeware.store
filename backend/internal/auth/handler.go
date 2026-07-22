package auth

import (
	"net/http"

	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes the auth HTTP endpoints.
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Register wires auth routes onto the mux under the given prefix.
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"/auth/register", h.register)
	mux.HandleFunc("POST "+prefix+"/auth/login", h.login)
	mux.HandleFunc("POST "+prefix+"/auth/refresh", h.refresh)
	mux.HandleFunc("POST "+prefix+"/auth/logout", h.logout)
	// GET /auth/me is protected; wired by the server with RequireAuth.
	mux.Handle("GET "+prefix+"/auth/me", h.svc.tokens.RequireAuth(http.HandlerFunc(h.me)))
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	resp, err := h.svc.Register(r.Context(), req, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, resp)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	resp, err := h.svc.Login(r.Context(), req, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := httpx.Decode(w, r, &req); err != nil {
		httpx.Error(w, err)
		return
	}
	resp, err := h.svc.Refresh(r.Context(), req.RefreshToken, r.UserAgent(), httpx.ClientIP(r))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, resp)
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	_ = httpx.Decode(w, r, &req)
	if err := h.svc.Logout(r.Context(), req.RefreshToken); err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	p, _ := FromContext(r.Context())
	user, err := h.svc.Me(r.Context(), p.UserID)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, user)
}
