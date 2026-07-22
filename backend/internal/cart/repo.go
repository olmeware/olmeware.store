package cart

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/money"
)

// ErrNotFound indicates no active cart / no such variant.
var ErrNotFound = errors.New("not found")

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// activeCartIDForUser returns the user's active cart id, or ErrNotFound.
func (r *Repo) activeCartIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`select id from carts where user_id = $1 and status = 'active'`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// activeCartIDForGuest returns the guest's active cart id, or ErrNotFound.
func (r *Repo) activeCartIDForGuest(ctx context.Context, guestHash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`select id from carts where guest_token_hash = $1 and status = 'active'`, guestHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// createUserCart opens a new active cart for a user.
func (r *Repo) createUserCart(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`insert into carts (user_id, status, currency) values ($1, 'active', 'MXN') returning id`,
		userID).Scan(&id)
	return id, err
}

// createGuestCart opens a new active cart for a guest token hash.
func (r *Repo) createGuestCart(ctx context.Context, guestHash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`insert into carts (guest_token_hash, status, currency, expires_at)
		 values ($1, 'active', 'MXN', now() + interval '30 days') returning id`,
		guestHash).Scan(&id)
	return id, err
}

// VariantStock returns available stock and unit price for a variant, or ErrNotFound.
func (r *Repo) VariantStock(ctx context.Context, variantID uuid.UUID) (available int, unitPriceMinor int64, err error) {
	const q = `
		select coalesce(i.on_hand - i.reserved, 0),
			coalesce(v.price_minor, p.base_price_minor)
		from product_variants v
		join products p on p.id = v.product_id
		left join inventory i on i.variant_id = v.id
		where v.id = $1 and v.active and v.deleted_at is null
		  and p.deleted_at is null and p.status = 'active'`
	err = r.db.QueryRow(ctx, q, variantID).Scan(&available, &unitPriceMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	return available, unitPriceMinor, err
}

// AddOrIncrement adds delta to a cart line, creating it if absent.
func (r *Repo) AddOrIncrement(ctx context.Context, cartID, variantID uuid.UUID, delta int) error {
	const q = `
		insert into cart_items (cart_id, variant_id, quantity)
		values ($1, $2, $3)
		on conflict (cart_id, variant_id)
		do update set quantity = cart_items.quantity + excluded.quantity`
	_, err := r.db.Exec(ctx, q, cartID, variantID, delta)
	return err
}

// SetQuantity sets an absolute quantity; qty <= 0 removes the line.
func (r *Repo) SetQuantity(ctx context.Context, cartID, variantID uuid.UUID, qty int) error {
	if qty <= 0 {
		return r.RemoveItem(ctx, cartID, variantID)
	}
	_, err := r.db.Exec(ctx,
		`update cart_items set quantity = $3 where cart_id = $1 and variant_id = $2`,
		cartID, variantID, qty)
	return err
}

// RemoveItem deletes a cart line.
func (r *Repo) RemoveItem(ctx context.Context, cartID, variantID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`delete from cart_items where cart_id = $1 and variant_id = $2`, cartID, variantID)
	return err
}

// Clear removes every line in the cart.
func (r *Repo) Clear(ctx context.Context, cartID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `delete from cart_items where cart_id = $1`, cartID)
	return err
}

// Load composes the full cart with items and totals.
func (r *Repo) Load(ctx context.Context, cartID uuid.UUID) (*Cart, error) {
	c := &Cart{ID: cartID, Currency: "MXN", Items: []Item{}}
	const q = `
		select ci.variant_id, v.product_id, p.slug, p.name, p.tech_label, p.garment,
			tt.logo_path, v.color_hex, v.size, v.sku,
			coalesce(v.price_minor, p.base_price_minor) as unit_price_minor,
			ci.quantity,
			coalesce(i.on_hand - i.reserved, 0) as available
		from cart_items ci
		join product_variants v on v.id = ci.variant_id
		join products p on p.id = v.product_id
		left join tech_themes tt on tt.id = p.tech_theme_id
		left join inventory i on i.variant_id = v.id
		where ci.cart_id = $1
		order by ci.created_at`
	rows, err := r.db.Query(ctx, q, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var it Item
		var logo *string
		if err := rows.Scan(&it.VariantID, &it.ProductID, &it.Slug, &it.Name, &it.Tech,
			&it.Garment, &logo, &it.ColorHex, &it.Size, &it.SKU, &it.UnitPriceMinor,
			&it.Quantity, &it.Available); err != nil {
			return nil, err
		}
		if logo != nil {
			it.Logo = *logo
		}
		it.LineTotalMinor = it.UnitPriceMinor * int64(it.Quantity)
		it.UnitPrice = money.FormatMXN(it.UnitPriceMinor)
		it.LineTotal = money.FormatMXN(it.LineTotalMinor)
		it.InStock = it.Available >= it.Quantity
		c.Items = append(c.Items, it)
		c.ItemCount += it.Quantity
		c.SubtotalMinor += it.LineTotalMinor
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	c.Subtotal = money.FormatMXN(c.SubtotalMinor)
	return c, nil
}
