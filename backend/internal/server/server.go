// Package server wires configuration, database, and feature handlers into an
// http.Handler and owns the API versioning prefix.
package server

import (
	"log"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/admin"
	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/cart"
	"github.com/olmeware/backend/internal/catalog"
	"github.com/olmeware/backend/internal/config"
	"github.com/olmeware/backend/internal/httpx"
	"github.com/olmeware/backend/internal/middleware"
	"github.com/olmeware/backend/internal/orders"
	"github.com/olmeware/backend/internal/payments"
)

// APIPrefix is the version prefix every endpoint lives under.
const APIPrefix = "/api/v1"

// Server holds shared dependencies used to build the HTTP handler.
type Server struct {
	cfg  *config.Config
	db   *pgxpool.Pool
	auth *auth.Service
}

// New constructs a Server and its service graph.
func New(cfg *config.Config, db *pgxpool.Pool) *Server {
	tokens := auth.NewTokenService(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authSvc := auth.NewService(auth.NewRepo(db), tokens, cfg.BcryptCost)

	stripeMode := "disabled"
	if cfg.StripeEnabled() {
		if cfg.StripeIsTest() {
			stripeMode = "test"
		} else {
			stripeMode = "LIVE"
		}
	} else if cfg.StripeSecretKey != "" && !cfg.StripeIsTest() {
		stripeMode = "disabled (live key blocked; set STRIPE_ALLOW_LIVE=true to enable)"
	}
	log.Printf("payments: stripe=%s crypto=%v", stripeMode, cfg.CoinbaseEnabled())

	return &Server{cfg: cfg, db: db, auth: authSvc}
}

// Auth exposes the auth service (used by seeding and tests).
func (s *Server) Auth() *auth.Service { return s.auth }

// Handler builds the fully wired http.Handler with middleware applied.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// Health / readiness.
	mux.HandleFunc("GET "+APIPrefix+"/health", s.health)
	mux.HandleFunc("GET /health", s.health)

	// Feature routes.
	auth.NewHandler(s.auth).Register(mux, APIPrefix)
	catalog.NewHandler(catalog.NewRepo(s.db)).Register(mux, APIPrefix)
	cart.NewHandler(cart.NewService(cart.NewRepo(s.db)), s.auth.Tokens()).Register(mux, APIPrefix)
	orders.NewHandler(orders.NewService(orders.NewRepo(s.db)), s.auth.Tokens()).Register(mux, APIPrefix)
	admin.NewHandler(admin.NewService(admin.NewRepo(s.db)), s.auth.Tokens()).Register(mux, APIPrefix)
	payments.NewHandler(payments.NewService(payments.Deps{
		Repo:            payments.NewRepo(s.db),
		StripeSecret:    s.cfg.StripeSecretKey,
		StripePublish:   s.cfg.StripePublishableKey,
		StripeWebhook:   s.cfg.StripeWebhookSecret,
		CoinbaseKey:     s.cfg.CoinbaseCommerceKey,
		CoinbaseWebhook: s.cfg.CoinbaseCommerceWebhookSecret,
		FrontendURL:     s.cfg.FrontendURL,
		StripeEnabled:   s.cfg.StripeEnabled(),
		CoinbaseEnabled: s.cfg.CoinbaseEnabled(),
	}), s.auth.Tokens()).Register(mux, APIPrefix)

	return middleware.Chain(mux,
		middleware.Recover,
		middleware.Logger,
		middleware.CORS(s.cfg.FrontendURL),
	)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	status := "ok"
	code := http.StatusOK
	if err := s.db.Ping(r.Context()); err != nil {
		status = "degraded"
		code = http.StatusServiceUnavailable
	}
	httpx.JSON(w, code, map[string]any{
		"status":  status,
		"service": "olmeware-backend",
		"version": "v1.0.0",
	})
}
