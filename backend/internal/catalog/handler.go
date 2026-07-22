package catalog

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/olmeware/backend/internal/httpx"
)

// Handler exposes the public catalog endpoints. Everything is composed
// server-side so the frontend only renders the response.
type Handler struct{ repo *Repo }

func NewHandler(repo *Repo) *Handler { return &Handler{repo: repo} }

// Register wires read-only catalog routes under prefix.
func (h *Handler) Register(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("GET "+prefix+"/catalog/products", h.listProducts)
	mux.HandleFunc("GET "+prefix+"/catalog/products/{slug}", h.getProduct)
	mux.HandleFunc("GET "+prefix+"/catalog/collections", h.listCollections)
	mux.HandleFunc("GET "+prefix+"/catalog/tech-themes", h.listThemes)
}

func (h *Handler) listProducts(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := ProductFilters{
		Garment:    q.Get("garment"),
		Stack:      q.Get("stack"),
		Size:       q.Get("size"),
		Collection: q.Get("collection"),
		Search:     q.Get("search"),
		Sort:       q.Get("sort"),
		Limit:      atoiDefault(q.Get("limit"), 60),
		Offset:     atoiDefault(q.Get("offset"), 0),
	}
	if v := q.Get("featured"); v == "true" {
		t := true
		f.Featured = &t
	}
	if v := q.Get("minPrice"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			m := n * 100
			f.MinMinor = &m
		}
	}
	if v := q.Get("maxPrice"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			m := n * 100
			f.MaxMinor = &m
		}
	}

	products, err := h.repo.ListProducts(r.Context(), f)
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"products": products, "count": len(products)})
}

func (h *Handler) getProduct(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	product, err := h.repo.GetProductBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			httpx.Error(w, httpx.NotFound("Product not found."))
			return
		}
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, product)
}

func (h *Handler) listCollections(w http.ResponseWriter, r *http.Request) {
	collections, err := h.repo.ListCollections(r.Context())
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"collections": collections})
}

func (h *Handler) listThemes(w http.ResponseWriter, r *http.Request) {
	themes, err := h.repo.ListTechThemes(r.Context(), r.URL.Query().Get("category"))
	if err != nil {
		httpx.Error(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"themes": themes})
}

func atoiDefault(s string, def int) int {
	if s == "" {
		return def
	}
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}
