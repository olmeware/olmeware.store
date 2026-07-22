package catalog

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/money"
)

// ErrNotFound is returned when a product/collection lookup yields nothing.
var ErrNotFound = errors.New("not found")

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

const sizeRank = `array_position(array['XS','S','M','L','XL','XXL']::text[], v.size)`

// listSelect is the shared product projection for storefront listings.
var listSelect = `
	select p.id, p.slug, p.name, p.description, p.garment, p.stack, p.tech_label,
		tt.logo_path, p.default_color_hex, p.base_price_minor, p.currency,
		p.featured, p.status,
		coalesce((
			select array_agg(v.size order by ` + sizeRank + `)
			from product_variants v
			where v.product_id = p.id and v.active and v.deleted_at is null
		), '{}') as sizes,
		coalesce((
			select array_agg(c.slug)
			from product_collections pc
			join collections c on c.id = pc.collection_id and c.deleted_at is null
			where pc.product_id = p.id
		), '{}') as collections
	from products p
	left join tech_themes tt on tt.id = p.tech_theme_id`

func scanProduct(row pgx.Row) (*Product, error) {
	var p Product
	var logo *string
	if err := row.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Garment, &p.Stack,
		&p.Tech, &logo, &p.ColorHex, &p.PriceMinor, &p.Currency, &p.Featured, &p.Status,
		&p.Sizes, &p.Collections); err != nil {
		return nil, err
	}
	if logo != nil {
		p.Logo = *logo
	}
	p.Price = money.FormatMXN(p.PriceMinor)
	p.Images = []string{}
	return &p, nil
}

// ListProducts returns storefront products matching the filters.
func (r *Repo) ListProducts(ctx context.Context, f ProductFilters) ([]Product, error) {
	var where []string
	var args []any
	add := func(cond string, val any) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, len(args)))
	}

	where = append(where, "p.deleted_at is null", "p.status = 'active'")
	if f.Garment != "" {
		add("p.garment = $%d", f.Garment)
	}
	if f.Stack != "" {
		add("p.stack = $%d", f.Stack)
	}
	if f.Search != "" {
		args = append(args, f.Search)
		n := len(args)
		where = append(where, fmt.Sprintf(
			"(p.name ilike '%%' || $%d || '%%' or p.tech_label ilike '%%' || $%d || '%%')", n, n))
	}
	if f.Featured != nil {
		add("p.featured = $%d", *f.Featured)
	}
	if f.MinMinor != nil {
		add("p.base_price_minor >= $%d", *f.MinMinor)
	}
	if f.MaxMinor != nil {
		add("p.base_price_minor <= $%d", *f.MaxMinor)
	}
	if f.Size != "" {
		add("exists (select 1 from product_variants v where v.product_id = p.id and v.active and v.deleted_at is null and v.size = $%d)", f.Size)
	}
	if f.Collection != "" {
		add("exists (select 1 from product_collections pc join collections c on c.id = pc.collection_id where pc.product_id = p.id and c.slug = $%d and c.deleted_at is null)", f.Collection)
	}

	query := listSelect + "\nwhere " + strings.Join(where, " and ") + "\norder by " + orderBy(f.Sort)

	limit := f.Limit
	if limit <= 0 || limit > 100 {
		limit = 60
	}
	args = append(args, limit)
	query += fmt.Sprintf("\nlimit $%d", len(args))
	args = append(args, f.Offset)
	query += fmt.Sprintf(" offset $%d", len(args))

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	products := []Product{}
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		products = append(products, *p)
	}
	return products, rows.Err()
}

// GetProductBySlug returns a single product with its variants and availability.
func (r *Repo) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	query := listSelect + "\nwhere p.deleted_at is null and p.slug = $1"
	p, err := scanProduct(r.db.QueryRow(ctx, query, slug))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	variants, err := r.variantsForProduct(ctx, p.ID.String())
	if err != nil {
		return nil, err
	}
	p.Variants = variants
	return p, nil
}

func (r *Repo) variantsForProduct(ctx context.Context, productID string) ([]Variant, error) {
	const q = `
		select v.id, v.sku, v.size, v.color_hex, coalesce(v.color_name,''),
			coalesce(v.price_minor, p.base_price_minor) as price_minor,
			coalesce(i.on_hand - i.reserved, 0) as available
		from product_variants v
		join products p on p.id = v.product_id
		left join inventory i on i.variant_id = v.id
		where v.product_id = $1 and v.active and v.deleted_at is null
		order by ` + sizeRank
	rows, err := r.db.Query(ctx, q, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := []Variant{}
	for rows.Next() {
		var v Variant
		if err := rows.Scan(&v.ID, &v.SKU, &v.Size, &v.ColorHex, &v.ColorName,
			&v.PriceMinor, &v.Available); err != nil {
			return nil, err
		}
		v.InStock = v.Available > 0
		variants = append(variants, v)
	}
	return variants, rows.Err()
}

// ListCollections returns active collections with their product counts.
func (r *Repo) ListCollections(ctx context.Context) ([]Collection, error) {
	const q = `
		select c.id, c.slug, c.name, c.description,
			(select count(*) from product_collections pc
			 join products p on p.id = pc.product_id
			 where pc.collection_id = c.id and p.deleted_at is null and p.status='active') as product_count
		from collections c
		where c.deleted_at is null and c.status = 'active'
		order by c.sort_order, c.name`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Collection{}
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.Description, &c.ProductCount); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListTechThemes returns active design themes (the logo library).
func (r *Repo) ListTechThemes(ctx context.Context, category string) ([]TechTheme, error) {
	q := `select id, slug, name, category, logo_path from tech_themes where active = true`
	var args []any
	if category != "" {
		q += " and category = $1"
		args = append(args, category)
	}
	q += " order by category, name"
	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []TechTheme{}
	for rows.Next() {
		var t TechTheme
		var logo *string
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.Category, &logo); err != nil {
			return nil, err
		}
		if logo != nil {
			t.Logo = *logo
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func orderBy(sort string) string {
	switch sort {
	case "price_asc":
		return "p.base_price_minor asc, p.name asc"
	case "price_desc":
		return "p.base_price_minor desc, p.name asc"
	case "name":
		return "p.name asc"
	default: // newest
		return "p.published_at desc nulls last, p.created_at desc"
	}
}
