package orders

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/olmeware/backend/internal/money"
)

const orderColumns = `o.id, o.order_number, o.status, o.customer_email, o.customer_name,
	o.currency, o.subtotal_minor, o.shipping_minor, o.tax_minor, o.discount_minor,
	o.total_minor, o.created_at`

func scanOrder(row pgx.Row) (*Order, error) {
	var o Order
	if err := row.Scan(&o.ID, &o.OrderNumber, &o.Status, &o.CustomerEmail, &o.CustomerName,
		&o.Currency, &o.SubtotalMinor, &o.ShippingMinor, &o.TaxMinor, &o.DiscountMinor,
		&o.TotalMinor, &o.CreatedAt); err != nil {
		return nil, err
	}
	o.Subtotal = money.FormatMXN(o.SubtotalMinor)
	o.Total = money.FormatMXN(o.TotalMinor)
	o.Items = []OrderItem{}
	return &o, nil
}

// GetByID returns an order with items. ownerID, when non-nil, restricts access
// to that user (admins pass nil to bypass the check).
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID, ownerID *uuid.UUID) (*Order, error) {
	q := `select ` + orderColumns + ` from orders o where o.id = $1`
	args := []any{id}
	if ownerID != nil {
		q += ` and o.user_id = $2`
		args = append(args, *ownerID)
	}
	order, err := scanOrder(r.db.QueryRow(ctx, q, args...))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	items, err := r.itemsFor(ctx, order.ID)
	if err != nil {
		return nil, err
	}
	order.Items = items
	return order, nil
}

// ListForUser returns a user's orders, newest first (items omitted).
func (r *Repo) ListForUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]Order, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := `select ` + orderColumns + ` from orders o
		where o.user_id = $1 order by o.created_at desc limit $2 offset $3`
	rows, err := r.db.Query(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Order{}
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func (r *Repo) itemsFor(ctx context.Context, orderID uuid.UUID) ([]OrderItem, error) {
	const q = `
		select id, product_id, sku, product_name, garment, tech_label, size, color_hex,
			coalesce(product_snapshot->>'logo',''), unit_price_minor, quantity, line_total_minor
		from order_items where order_id = $1 order by created_at`
	rows, err := r.db.Query(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []OrderItem{}
	for rows.Next() {
		var it OrderItem
		var productID *uuid.UUID
		if err := rows.Scan(&it.ID, &productID, &it.SKU, &it.ProductName, &it.Garment, &it.Tech,
			&it.Size, &it.ColorHex, &it.Logo, &it.UnitPriceMinor, &it.Quantity,
			&it.LineTotalMinor); err != nil {
			return nil, err
		}
		if productID != nil {
			it.ProductID = *productID
		}
		it.UnitPrice = money.FormatMXN(it.UnitPriceMinor)
		it.LineTotal = money.FormatMXN(it.LineTotalMinor)
		items = append(items, it)
	}
	return items, rows.Err()
}
