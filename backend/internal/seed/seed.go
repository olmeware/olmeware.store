// Package seed idempotently populates the store with the bootstrap admin and
// the initial catalog (collections, tech themes, products, variants, inventory)
// mirroring the frontend's lib/seed.ts. Safe to run repeatedly.
package seed

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/auth"
	"github.com/olmeware/backend/internal/config"
)

// Run applies the full seed within a single transaction.
func Run(ctx context.Context, db *pgxpool.Pool, cfg *config.Config) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	adminID, err := seedAdmin(ctx, tx, cfg)
	if err != nil {
		return fmt.Errorf("seed admin: %w", err)
	}
	if err := seedCollectionsData(ctx, tx); err != nil {
		return fmt.Errorf("seed collections: %w", err)
	}
	if err := seedCatalog(ctx, tx, adminID); err != nil {
		return fmt.Errorf("seed catalog: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("seed: admin + %d collections + %d products applied", len(seedCollections), len(seedProducts))
	return nil
}

func seedAdmin(ctx context.Context, tx pgx.Tx, cfg *config.Config) (string, error) {
	hash, err := auth.HashPassword(cfg.AdminPassword, cfg.BcryptCost)
	if err != nil {
		return "", err
	}
	// Insert once; on repeat runs keep the existing admin (don't clobber a
	// rotated password) but ensure the role is admin.
	const q = `
		insert into users (email, password_hash, full_name, role, status)
		values (lower(btrim($1)), $2, $3, 'admin', 'active')
		on conflict (lower(email)) where (deleted_at is null)
		do update set role = 'admin'
		returning id`
	var id string
	err = tx.QueryRow(ctx, q, cfg.AdminEmail, hash, cfg.AdminName).Scan(&id)
	return id, err
}

func seedCollectionsData(ctx context.Context, tx pgx.Tx) error {
	const q = `
		insert into collections (name, slug, description, status, sort_order)
		values ($1, $2, $3, 'active', $4)
		on conflict (slug) where (deleted_at is null)
		do update set name = excluded.name, description = excluded.description,
			sort_order = excluded.sort_order`
	for _, c := range seedCollections {
		if _, err := tx.Exec(ctx, q, c.Name, c.Slug, c.Description, c.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func seedCatalog(ctx context.Context, tx pgx.Tx, adminID string) error {
	for _, p := range seedProducts {
		themeID, err := upsertTheme(ctx, tx, p)
		if err != nil {
			return fmt.Errorf("theme %s: %w", p.Tech, err)
		}
		productID, err := upsertProduct(ctx, tx, p, themeID, adminID)
		if err != nil {
			return fmt.Errorf("product %s: %w", p.Slug, err)
		}
		if err := upsertVariants(ctx, tx, productID, p); err != nil {
			return fmt.Errorf("variants %s: %w", p.Slug, err)
		}
		if p.Collection != "" {
			if err := linkCollection(ctx, tx, productID, p.Collection); err != nil {
				return fmt.Errorf("link %s: %w", p.Slug, err)
			}
		}
	}
	return nil
}

func upsertTheme(ctx context.Context, tx pgx.Tx, p seedProduct) (string, error) {
	const q = `
		insert into tech_themes (name, slug, category, logo_path, active)
		values ($1, $2, $3, $4, true)
		on conflict (slug) do update set name = excluded.name,
			category = excluded.category, logo_path = excluded.logo_path
		returning id`
	var id string
	err := tx.QueryRow(ctx, q, p.Tech, p.LogoSlug, categoryForStack(p.Stack),
		"/logos/"+p.LogoSlug+".svg").Scan(&id)
	return id, err
}

func upsertProduct(ctx context.Context, tx pgx.Tx, p seedProduct, themeID, adminID string) (string, error) {
	const q = `
		insert into products (name, slug, description, garment, stack, tech_theme_id,
			tech_label, status, featured, default_color_hex, base_price_minor, currency,
			created_by, updated_by, published_at)
		values ($1, $2, $3, $4, $5, $6, $7, 'active', $8, $9, $10, 'MXN', $11, $11, now())
		on conflict (slug) where (deleted_at is null)
		do update set name = excluded.name, description = excluded.description,
			garment = excluded.garment, stack = excluded.stack,
			tech_theme_id = excluded.tech_theme_id, tech_label = excluded.tech_label,
			featured = excluded.featured, default_color_hex = excluded.default_color_hex,
			base_price_minor = excluded.base_price_minor, updated_by = excluded.updated_by
		returning id`
	var id string
	err := tx.QueryRow(ctx, q, p.Name, p.Slug, p.Description, p.Garment, p.Stack,
		themeID, p.Tech, p.Featured, p.ColorHex, int64(p.PriceMajor)*100, adminID).Scan(&id)
	return id, err
}

func upsertVariants(ctx context.Context, tx pgx.Tx, productID string, p seedProduct) error {
	const variantQ = `
		insert into product_variants (product_id, sku, size, color_name, color_hex, active)
		values ($1, $2, $3, $4, $5, true)
		on conflict (product_id, size, color_hex) where (deleted_at is null)
		do update set sku = excluded.sku, active = true
		returning id`
	const invQ = `
		insert into inventory (variant_id, on_hand, reorder_level)
		values ($1, 100, 10)
		on conflict (variant_id) do nothing`
	for _, size := range p.Sizes {
		sku := fmt.Sprintf("%s-%s", strings.ToUpper(strings.ReplaceAll(p.Slug, "-", "")), size)
		var variantID string
		if err := tx.QueryRow(ctx, variantQ, productID, sku, size,
			colorName(p.ColorHex), p.ColorHex).Scan(&variantID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, invQ, variantID); err != nil {
			return err
		}
	}
	return nil
}

func linkCollection(ctx context.Context, tx pgx.Tx, productID, collectionSlug string) error {
	const q = `
		insert into product_collections (product_id, collection_id)
		select $1, c.id from collections c where c.slug = $2 and c.deleted_at is null
		on conflict (product_id, collection_id) do nothing`
	_, err := tx.Exec(ctx, q, productID, collectionSlug)
	return err
}

func categoryForStack(stack string) string {
	switch stack {
	case "languages":
		return "languages"
	case "frontend":
		return "frontend"
	case "backend":
		return "backend"
	case "ai-ml":
		return "ai-machine-learning"
	case "devops":
		return "devops-infrastructure"
	default:
		return "tools"
	}
}

func colorName(hex string) string {
	switch strings.ToLower(hex) {
	case black:
		return "Black"
	case white:
		return "White"
	default:
		return ""
	}
}
