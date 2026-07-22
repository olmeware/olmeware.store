package orders

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/olmeware/backend/internal/money"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrEmptyCart  = errors.New("cart is empty")
	ErrOutOfStock = errors.New("insufficient stock")
)

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// ActiveCartIDForUser resolves a user's active cart, or ErrNotFound.
func (r *Repo) ActiveCartIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`select id from carts where user_id = $1 and status = 'active'`, userID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

// ActiveCartIDForGuest resolves a guest's active cart, or ErrNotFound.
func (r *Repo) ActiveCartIDForGuest(ctx context.Context, guestHash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRow(ctx,
		`select id from carts where guest_token_hash = $1 and status = 'active'`, guestHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return id, err
}

type lineInput struct {
	variantID uuid.UUID
	productID uuid.UUID
	quantity  int
	sku       string
	name      string
	garment   string
	tech      string
	size      string
	colorHex  string
	logo      string
	unitMinor int64
	available int
}

// CreateFromCart converts an active cart into a pending order in one
// transaction: it locks inventory, validates stock, snapshots line items,
// reserves stock (append-only ledger), and marks the cart converted.
func (r *Repo) CreateFromCart(ctx context.Context, cartID uuid.UUID, userID *uuid.UUID, req CreateOrderRequest) (*Order, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	lines, err := lockCartLines(ctx, tx, cartID)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, ErrEmptyCart
	}

	var subtotal int64
	for _, l := range lines {
		if l.available < l.quantity {
			return nil, ErrOutOfStock
		}
		subtotal += l.unitMinor * int64(l.quantity)
	}

	shippingJSON, err := json.Marshal(req.ShippingAddress)
	if err != nil {
		return nil, err
	}
	var billingJSON []byte
	if req.BillingAddress != nil {
		if billingJSON, err = json.Marshal(req.BillingAddress); err != nil {
			return nil, err
		}
	}

	order := &Order{
		Status: "pending_payment", CustomerEmail: req.Email, CustomerName: req.Name,
		Currency: "MXN", SubtotalMinor: subtotal, TotalMinor: subtotal,
	}
	const orderQ = `
		insert into orders (user_id, cart_id, status, customer_email, customer_name,
			customer_phone, currency, subtotal_minor, discount_minor, shipping_minor,
			tax_minor, total_minor, shipping_address, billing_address, customer_note, placed_at)
		values ($1, $2, 'pending_payment', lower(btrim($3)), $4, nullif($5,''), 'MXN',
			$6, 0, 0, 0, $6, $7::jsonb, $8::jsonb, nullif($9,''), now())
		returning id, order_number, created_at`
	if err := tx.QueryRow(ctx, orderQ, userID, cartID, req.Email, req.Name, req.Phone,
		subtotal, string(shippingJSON), nullableJSON(billingJSON), req.Note).
		Scan(&order.ID, &order.OrderNumber, &order.CreatedAt); err != nil {
		return nil, err
	}

	for _, l := range lines {
		snapshot, _ := json.Marshal(map[string]any{"logo": l.logo, "tech": l.tech})
		lineTotal := l.unitMinor * int64(l.quantity)
		const itemQ = `
			insert into order_items (order_id, variant_id, product_id, sku, product_name,
				garment, tech_label, size, color_hex, unit_price_minor, quantity,
				line_total_minor, product_snapshot)
			values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13::jsonb)
			returning id`
		var itemID uuid.UUID
		if err := tx.QueryRow(ctx, itemQ, order.ID, l.variantID, l.productID, l.sku, l.name,
			l.garment, l.tech, l.size, l.colorHex, l.unitMinor, l.quantity, lineTotal,
			string(snapshot)).Scan(&itemID); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx,
			`update inventory set reserved = reserved + $2 where variant_id = $1`,
			l.variantID, l.quantity); err != nil {
			return nil, err
		}
		if _, err := tx.Exec(ctx, `
			insert into inventory_movements (variant_id, movement_type, quantity_delta,
				reservation_delta, reference_type, reference_id)
			values ($1, 'reservation', 0, $2, 'order', $3)`,
			l.variantID, l.quantity, order.ID); err != nil {
			return nil, err
		}
		order.Items = append(order.Items, OrderItem{
			ID: itemID, ProductID: l.productID, SKU: l.sku, ProductName: l.name,
			Garment: l.garment, Tech: l.tech, Size: l.size, ColorHex: l.colorHex, Logo: l.logo,
			UnitPriceMinor: l.unitMinor, UnitPrice: money.FormatMXN(l.unitMinor),
			Quantity: l.quantity, LineTotalMinor: lineTotal, LineTotal: money.FormatMXN(lineTotal),
		})
	}

	if _, err := tx.Exec(ctx,
		`update carts set status = 'converted', converted_order_id = $2 where id = $1`,
		cartID, order.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	order.Subtotal = money.FormatMXN(order.SubtotalMinor)
	order.Total = money.FormatMXN(order.TotalMinor)
	return order, nil
}

func lockCartLines(ctx context.Context, tx pgx.Tx, cartID uuid.UUID) ([]lineInput, error) {
	const q = `
		select ci.variant_id, v.product_id, ci.quantity, v.sku, p.name, p.garment,
			p.tech_label, v.size, v.color_hex, coalesce(tt.logo_path,''),
			coalesce(v.price_minor, p.base_price_minor),
			coalesce(i.on_hand - i.reserved, 0)
		from cart_items ci
		join product_variants v on v.id = ci.variant_id
		join products p on p.id = v.product_id
		left join tech_themes tt on tt.id = p.tech_theme_id
		join inventory i on i.variant_id = v.id
		where ci.cart_id = $1
		order by ci.created_at
		for update of i`
	rows, err := tx.Query(ctx, q, cartID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lines []lineInput
	for rows.Next() {
		var l lineInput
		if err := rows.Scan(&l.variantID, &l.productID, &l.quantity, &l.sku, &l.name,
			&l.garment, &l.tech, &l.size, &l.colorHex, &l.logo, &l.unitMinor,
			&l.available); err != nil {
			return nil, err
		}
		lines = append(lines, l)
	}
	return lines, rows.Err()
}

func nullableJSON(b []byte) any {
	if len(b) == 0 {
		return nil
	}
	return string(b)
}
