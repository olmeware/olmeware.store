package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds every runtime setting the backend needs. Values come from the
// environment (loaded from .env.local in development).
type Config struct {
	Port        string
	DatabaseURL string
	FrontendURL string

	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	BcryptCost      int

	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	AllowLiveStripe      bool

	// Coinbase Commerce (crypto: BTC / ETH / SOL).
	CoinbaseCommerceKey           string
	CoinbaseCommerceWebhookSecret string

	// Bootstrap admin (the single store administrator).
	AdminEmail    string
	AdminPassword string
	AdminName     string
}

// Load reads .env.local (if present) and builds a Config, applying sensible
// defaults. It fails fast when a required secret is missing.
func Load() (*Config, error) {
	// Best-effort: a missing .env.local is fine when real env vars are set.
	_ = godotenv.Load(".env.local")

	cfg := &Config{
		Port:            getenv("PORT", "8000"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		FrontendURL:     strings.TrimRight(getenv("FRONTEND_URL", "http://localhost:3000"), "/"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  getdur("ACCESS_TOKEN_TTL", 15*time.Minute),
		RefreshTokenTTL: getdur("REFRESH_TOKEN_TTL", 30*24*time.Hour),
		BcryptCost:      12,

		// Prefer explicit TEST keys when present, so development never picks up
		// live keys by accident.
		StripeSecretKey:      firstenv("STRIPE_TEST_SECRET_KEY", "STRIPE_SECRET_KEY"),
		StripePublishableKey: firstenv("STRIPE_TEST_PUBLISHABLE_KEY", "STRIPE_PUBLISHABLE_KEY", "STRIPE_PUBLISHABLE-KEY"),
		StripeWebhookSecret:  firstenv("STRIPE_TEST_WEBHOOK_SECRET", "STRIPE_WEBHOOK_SECRET"),
		AllowLiveStripe:      os.Getenv("STRIPE_ALLOW_LIVE") == "true",

		CoinbaseCommerceKey:           os.Getenv("COINBASE_COMMERCE_KEY"),
		CoinbaseCommerceWebhookSecret: os.Getenv("COINBASE_COMMERCE_WEBHOOK_SECRET"),

		AdminEmail:    getenv("ADMIN_EMAIL", "admin@olmeware.store"),
		AdminPassword: getenv("ADMIN_PASSWORD", "admin123"),
		AdminName:     getenv("ADMIN_NAME", "Olmeware Admin"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		// Dev fallback so the server boots; override in production.
		cfg.JWTSecret = "dev-insecure-jwt-secret-change-me"
	}
	return cfg, nil
}

// StripeIsTest reports whether the configured Stripe secret key is a test key.
func (c *Config) StripeIsTest() bool { return strings.HasPrefix(c.StripeSecretKey, "sk_test") }

// StripeEnabled reports whether payment calls to Stripe can be made. Live keys
// are refused unless STRIPE_ALLOW_LIVE=true, preventing accidental real charges.
func (c *Config) StripeEnabled() bool {
	if c.StripeSecretKey == "" {
		return false
	}
	return c.StripeIsTest() || c.AllowLiveStripe
}

// CoinbaseEnabled reports whether crypto charges can be created.
func (c *Config) CoinbaseEnabled() bool { return c.CoinbaseCommerceKey != "" }

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstenv(keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return ""
}

func getdur(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
