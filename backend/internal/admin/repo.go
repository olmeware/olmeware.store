package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/money"
)

var ErrNotFound = errors.New("not found")

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// ---- Products ----

// ListProducts returns every non-deleted product (all statuses) for the admin table.
func (r *Repo) ListProducts(ctx context.Context) ([]AdminProduct, error) {
	const q = `
		select p.id, p.slug, p.name, p.description, p.garment, p.stack, p.tech_label,
			coalesce(tt.logo_path,''), p.default_color_hex, p.status, p.featured, p.base_price_minor,
			coalesce((
				select array_agg(v.size order by array_position(array['XS','S','M','L','XL','XXL']::text[], v.size))
				from product_variants v where v.product_id = p.id and v.active and v.deleted_at is null
			), '{}') as sizes,
			coalesce((
				select array_agg(c.slug)
				from product_collections pc
				join collections c on c.id = pc.collection_id and c.deleted_at is null
				where pc.product_id = p.id
			), '{}') as collections,
			(select count(*) from product_variants v where v.product_id = p.id and v.deleted_at is null),
			p.created_at
		from products p
		left join tech_themes tt on tt.id = p.tech_theme_id
		where p.deleted_at is null
		order by p.created_at desc`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AdminProduct{}
	for rows.Next() {
		var p AdminProduct
		if err := rows.Scan(&p.ID, &p.Slug, &p.Name, &p.Description, &p.Garment, &p.Stack,
			&p.Tech, &p.Logo, &p.ColorHex, &p.Status, &p.Featured, &p.PriceMinor,
			&p.Sizes, &p.Collections, &p.VariantCount, &p.CreatedAt); err != nil {
			return nil, err
		}
		p.Price = money.FormatMXN(p.PriceMinor)
		p.Images = []string{}
		out = append(out, p)
	}
	return out, rows.Err()
}

// CreateProduct inserts a product with variants + inventory and links it to a
// collection, all in one transaction, then writes an audit entry.
func (r *Repo) CreateProduct(ctx context.Context, in ProductInput, adminID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	slug, err := uniqueSlug(ctx, tx, "products", slugify(in.Name))
	if err != nil {
		return uuid.Nil, err
	}
	themeID, err := upsertTheme(ctx, tx, in)
	if err != nil {
		return uuid.Nil, err
	}

	status := normalizeStatus(in.Status)
	const q = `
		insert into products (name, slug, description, garment, stack, tech_theme_id,
			tech_label, status, featured, default_color_hex, base_price_minor, currency,
			created_by, updated_by, published_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8::product_status,$9,$10,$11,'MXN',$12,$12,
			case when $8 = 'active' then now() else null end)
		returning id`
	var id uuid.UUID
	if err := tx.QueryRow(ctx, q, in.Name, slug, in.Description, in.Garment, in.Stack,
		themeID, in.Tech, status, in.Featured, in.ColorHex, int64(in.PriceMajor)*100,
		adminID).Scan(&id); err != nil {
		return uuid.Nil, err
	}

	if err := syncVariants(ctx, tx, id, in.Sizes, in.ColorHex); err != nil {
		return uuid.Nil, err
	}
	if in.CollectionSlug != "" {
		if err := linkCollection(ctx, tx, id, in.CollectionSlug); err != nil {
			return uuid.Nil, err
		}
	}
	if err := audit(ctx, tx, adminID, "product.create", "product", &id, in); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpdateProduct updates a product's fields and re-syncs its variants.
func (r *Repo) UpdateProduct(ctx context.Context, id uuid.UUID, in ProductInput, adminID uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	themeID, err := upsertTheme(ctx, tx, in)
	if err != nil {
		return err
	}
	const q = `
		update products set name=$2, description=$3, garment=$4, stack=$5,
			tech_theme_id=$6, tech_label=$7, featured=$8, default_color_hex=$9,
			base_price_minor=$10, status=$11::product_status, updated_by=$12,
			published_at = case when $11 = 'active' and published_at is null then now() else published_at end
		where id=$1 and deleted_at is null`
	tag, err := tx.Exec(ctx, q, id, in.Name, in.Description, in.Garment, in.Stack, themeID,
		in.Tech, in.Featured, in.ColorHex, int64(in.PriceMajor)*100, normalizeStatus(in.Status), adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if len(in.Sizes) > 0 {
		if err := syncVariants(ctx, tx, id, in.Sizes, in.ColorHex); err != nil {
			return err
		}
	}
	if err := audit(ctx, tx, adminID, "product.update", "product", &id, in); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// SetProductStatus changes a product's status.
func (r *Repo) SetProductStatus(ctx context.Context, id uuid.UUID, status string, adminID uuid.UUID) error {
	status = normalizeStatus(status)
	tag, err := r.db.Exec(ctx, `
		update products set status=$2::product_status, updated_by=$3,
			published_at = case when $2 = 'active' and published_at is null then now() else published_at end
		where id=$1 and deleted_at is null`, id, status, adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_ = auditPool(ctx, r.db, adminID, "product.status", "product", &id, map[string]string{"status": status})
	return nil
}

// SoftDeleteProduct marks a product deleted.
func (r *Repo) SoftDeleteProduct(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`update products set deleted_at = now(), updated_by=$2 where id=$1 and deleted_at is null`,
		id, adminID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_ = auditPool(ctx, r.db, adminID, "product.delete", "product", &id, nil)
	return nil
}

// ---- Collections ----

// ListCollections returns every non-deleted collection with product counts.
func (r *Repo) ListCollections(ctx context.Context) ([]AdminCollection, error) {
	const q = `
		select c.id, c.slug, c.name, c.description, c.sort_order,
			(select count(*) from product_collections pc where pc.collection_id = c.id),
			c.created_at
		from collections c where c.deleted_at is null
		order by c.sort_order, c.name`
	rows, err := r.db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AdminCollection{}
	for rows.Next() {
		var c AdminCollection
		if err := rows.Scan(&c.ID, &c.Slug, &c.Name, &c.Description, &c.SortOrder,
			&c.ProductCount, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateCollection inserts a collection.
func (r *Repo) CreateCollection(ctx context.Context, in CollectionInput, adminID uuid.UUID) (uuid.UUID, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback(ctx)

	slug, err := uniqueSlug(ctx, tx, "collections", slugify(in.Name))
	if err != nil {
		return uuid.Nil, err
	}
	var id uuid.UUID
	if err := tx.QueryRow(ctx, `
		insert into collections (name, slug, description, status, sort_order)
		values ($1,$2,$3,'active',$4) returning id`,
		in.Name, slug, in.Description, in.SortOrder).Scan(&id); err != nil {
		return uuid.Nil, err
	}
	if err := audit(ctx, tx, adminID, "collection.create", "collection", &id, in); err != nil {
		return uuid.Nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// UpdateCollection updates a collection's fields.
func (r *Repo) UpdateCollection(ctx context.Context, id uuid.UUID, in CollectionInput, adminID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		update collections set name=$2, description=$3, sort_order=$4
		where id=$1 and deleted_at is null`, id, in.Name, in.Description, in.SortOrder)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_ = auditPool(ctx, r.db, adminID, "collection.update", "collection", &id, in)
	return nil
}

// SoftDeleteCollection marks a collection deleted (products keep existing).
func (r *Repo) SoftDeleteCollection(ctx context.Context, id uuid.UUID, adminID uuid.UUID) error {
	tag, err := r.db.Exec(ctx,
		`update collections set deleted_at = now() where id=$1 and deleted_at is null`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	_ = auditPool(ctx, r.db, adminID, "collection.delete", "collection", &id, nil)
	return nil
}

// ---- shared helpers ----

func upsertTheme(ctx context.Context, tx pgx.Tx, in ProductInput) (any, error) {
	if in.LogoSlug == "" {
		return nil, nil
	}
	const q = `
		insert into tech_themes (name, slug, category, logo_path, active)
		values ($1,$2,$3,$4,true)
		on conflict (slug) do update set name=excluded.name, logo_path=excluded.logo_path
		returning id`
	var id uuid.UUID
	err := tx.QueryRow(ctx, q, in.Tech, in.LogoSlug, in.Stack, "/logos/"+in.LogoSlug+".svg").Scan(&id)
	return id, err
}

// syncVariants activates the given sizes (creating variants + made-to-order
// inventory) and deactivates any others for the product.
func syncVariants(ctx context.Context, tx pgx.Tx, productID uuid.UUID, sizes []string, colorHex string) error {
	for _, size := range sizes {
		size = strings.TrimSpace(size)
		if size == "" {
			continue
		}
		sku := fmt.Sprintf("%s-%s", strings.ToUpper(strings.ReplaceAll(shortID(productID), "-", "")), size)
		var variantID uuid.UUID
		if err := tx.QueryRow(ctx, `
			insert into product_variants (product_id, sku, size, color_hex, active)
			values ($1,$2,$3,$4,true)
			on conflict (product_id, size, color_hex) where (deleted_at is null)
			do update set active=true
			returning id`, productID, sku, size, colorHex).Scan(&variantID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`insert into inventory (variant_id, on_hand, reorder_level) values ($1, 1000, 0)
			 on conflict (variant_id) do nothing`, variantID); err != nil {
			return err
		}
	}
	if len(sizes) > 0 {
		if _, err := tx.Exec(ctx, `
			update product_variants set active=false
			where product_id=$1 and deleted_at is null and size <> all($2)`,
			productID, sizes); err != nil {
			return err
		}
	}
	return nil
}

func linkCollection(ctx context.Context, tx pgx.Tx, productID uuid.UUID, slug string) error {
	_, err := tx.Exec(ctx, `
		insert into product_collections (product_id, collection_id)
		select $1, c.id from collections c where c.slug=$2 and c.deleted_at is null
		on conflict (product_id, collection_id) do nothing`, productID, slug)
	return err
}

// uniqueSlug returns base, or base-2, base-3, ... if taken (among non-deleted).
func uniqueSlug(ctx context.Context, tx pgx.Tx, table, base string) (string, error) {
	if base == "" {
		base = "item"
	}
	slug := base
	for i := 2; ; i++ {
		var exists bool
		q := fmt.Sprintf(`select exists(select 1 from %s where slug=$1 and deleted_at is null)`, table)
		if err := tx.QueryRow(ctx, q, slug).Scan(&exists); err != nil {
			return "", err
		}
		if !exists {
			return slug, nil
		}
		slug = fmt.Sprintf("%s-%d", base, i)
	}
}

func audit(ctx context.Context, tx pgx.Tx, adminID uuid.UUID, action, entity string, entityID *uuid.UUID, after any) error {
	data, _ := json.Marshal(after)
	_, err := tx.Exec(ctx, `
		insert into admin_audit_log (admin_user_id, action, entity_type, entity_id, after_data)
		values ($1,$2,$3,$4,$5::jsonb)`, adminID, action, entity, entityID, nullableJSON(data))
	return err
}

func auditPool(ctx context.Context, db *pgxpool.Pool, adminID uuid.UUID, action, entity string, entityID *uuid.UUID, after any) error {
	data, _ := json.Marshal(after)
	_, err := db.Exec(ctx, `
		insert into admin_audit_log (admin_user_id, action, entity_type, entity_id, after_data)
		values ($1,$2,$3,$4,$5::jsonb)`, adminID, action, entity, entityID, nullableJSON(data))
	return err
}

func normalizeStatus(s string) string {
	switch s {
	case "active", "draft", "archived":
		return s
	default:
		return "draft"
	}
}

func shortID(id uuid.UUID) string { return strings.ReplaceAll(id.String(), "-", "")[:12] }

func nullableJSON(b []byte) any {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	return string(b)
}
