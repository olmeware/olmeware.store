package payments

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNotPayable   = errors.New("order is not awaiting payment")
	ErrUnauthorized = errors.New("not allowed")
)

type Repo struct{ db *pgxpool.Pool }

func NewRepo(db *pgxpool.Pool) *Repo { return &Repo{db: db} }

// getOrder loads the payment view of an order.
func (r *Repo) getOrder(ctx context.Context, id uuid.UUID) (*orderForPayment, error) {
	const q = `select id, status, total_minor, currency, user_id, customer_email,
		customer_name, order_number from orders where id = $1`
	var o orderForPayment
	err := r.db.QueryRow(ctx, q, id).Scan(&o.ID, &o.Status, &o.TotalMinor, &o.Currency,
		&o.UserID, &o.Email, &o.Name, &o.OrderNumber)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &o, err
}

// findPayment returns an existing payment for (provider, idempotencyKey), if any.
func (r *Repo) findPayment(ctx context.Context, provider, idempotencyKey string) (id uuid.UUID, intentID, status string, err error) {
	const q = `select id, coalesce(provider_payment_intent_id,''), status
		from payments where provider = $1 and idempotency_key = $2`
	var pid *string
	err = r.db.QueryRow(ctx, q, provider, idempotencyKey).Scan(&id, &pid, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", "", ErrNotFound
	}
	if pid != nil {
		intentID = *pid
	}
	return id, intentID, status, err
}

// insertOrGetPayment creates a pending payment row, or returns the existing one
// for (provider, idempotency_key). The unique index makes concurrent duplicate
// attempts collapse to a single payment — the core once-only guarantee.
func (r *Repo) insertOrGetPayment(ctx context.Context, orderID uuid.UUID, provider string, amount int64, currency, idempotencyKey, intentID string) (id uuid.UUID, created bool, err error) {
	const insert = `
		insert into payments (order_id, provider, status, amount_minor, currency,
			provider_payment_intent_id, idempotency_key)
		values ($1, $2, 'processing', $3, $4, nullif($5,''), $6)
		on conflict (provider, idempotency_key) do nothing
		returning id`
	err = r.db.QueryRow(ctx, insert, orderID, provider, amount, currency, intentID, idempotencyKey).Scan(&id)
	if err == nil {
		return id, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, false, err
	}
	// Row already existed: fetch it.
	err = r.db.QueryRow(ctx,
		`select id from payments where provider = $1 and idempotency_key = $2`,
		provider, idempotencyKey).Scan(&id)
	return id, false, err
}

// setPaymentIntent records the provider intent/charge id on a payment.
func (r *Repo) setPaymentIntent(ctx context.Context, paymentID uuid.UUID, intentID string) error {
	_, err := r.db.Exec(ctx,
		`update payments set provider_payment_intent_id = $2 where id = $1`, paymentID, intentID)
	return err
}

// markSucceededByIntent transitions a payment to succeeded and, if this is the
// first time the order is paid, flips the order to paid and converts inventory
// reservations into sales — all idempotently, in one transaction.
func (r *Repo) markSucceededByIntent(ctx context.Context, provider, intentID, chargeID string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var paymentID, orderID uuid.UUID
	err = tx.QueryRow(ctx, `
		update payments set status = 'succeeded', succeeded_at = now(),
			provider_charge_id = nullif($3,'')
		where provider = $1 and provider_payment_intent_id = $2
		  and status <> 'succeeded'
		returning id, order_id`, provider, intentID, chargeID).Scan(&paymentID, &orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx) // already succeeded: nothing to do.
	}
	if err != nil {
		return err
	}

	// Flip the order to paid only once.
	var confirmedOrder uuid.UUID
	err = tx.QueryRow(ctx, `
		update orders set status = 'paid', paid_at = now()
		where id = $1 and status = 'pending_payment' returning id`, orderID).Scan(&confirmedOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx) // order already paid/processed.
	}
	if err != nil {
		return err
	}

	// Convert reservations to sales for each line.
	rows, err := tx.Query(ctx, `select variant_id, quantity from order_items where order_id = $1`, orderID)
	if err != nil {
		return err
	}
	type line struct {
		variantID uuid.UUID
		qty       int
	}
	var lines []line
	for rows.Next() {
		var l line
		if err := rows.Scan(&l.variantID, &l.qty); err != nil {
			rows.Close()
			return err
		}
		lines = append(lines, l)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}

	for _, l := range lines {
		if _, err := tx.Exec(ctx, `
			update inventory set on_hand = on_hand - $2, reserved = reserved - $2
			where variant_id = $1`, l.variantID, l.qty); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
			insert into inventory_movements (variant_id, movement_type, quantity_delta,
				reservation_delta, reference_type, reference_id)
			values ($1, 'sale', $2, $3, 'order', $4)`,
			l.variantID, -l.qty, -l.qty, orderID); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// markFailedByIntent records a payment failure (leaves the order pending).
func (r *Repo) markFailedByIntent(ctx context.Context, provider, intentID, code, message string) error {
	_, err := r.db.Exec(ctx, `
		update payments set status = 'failed', failure_code = nullif($3,''),
			failure_message = nullif($4,'')
		where provider = $1 and provider_payment_intent_id = $2 and status <> 'succeeded'`,
		provider, intentID, code, message)
	return err
}

// ---- webhook inboxes (idempotent once-only processing) ----

// insertStripeEvent records an event id; returns false if already seen.
func (r *Repo) insertStripeEvent(ctx context.Context, id, eventType, apiVersion string, livemode bool, payload []byte) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		insert into stripe_webhook_events (event_id, event_type, api_version, livemode, payload)
		values ($1, $2, nullif($3,''), $4, $5::jsonb)
		on conflict (event_id) do nothing`, id, eventType, apiVersion, livemode, string(payload))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) markStripeProcessed(ctx context.Context, id string) {
	_, _ = r.db.Exec(ctx, `update stripe_webhook_events set processed_at = now() where event_id = $1`, id)
}

func (r *Repo) markStripeError(ctx context.Context, id, msg string) {
	_, _ = r.db.Exec(ctx, `update stripe_webhook_events
		set processing_attempts = processing_attempts + 1, last_error = $2 where event_id = $1`, id, msg)
}

// insertCryptoEvent records a crypto event id; returns false if already seen.
func (r *Repo) insertCryptoEvent(ctx context.Context, id, eventType, chargeCode string, payload []byte) (bool, error) {
	tag, err := r.db.Exec(ctx, `
		insert into crypto_webhook_events (event_id, provider, event_type, charge_code, payload)
		values ($1, 'coinbase', $2, nullif($3,''), $4::jsonb)
		on conflict (event_id) do nothing`, id, eventType, chargeCode, string(payload))
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (r *Repo) markCryptoProcessed(ctx context.Context, id string) {
	_, _ = r.db.Exec(ctx, `update crypto_webhook_events set processed_at = now() where event_id = $1`, id)
}

func (r *Repo) markCryptoError(ctx context.Context, id, msg string) {
	_, _ = r.db.Exec(ctx, `update crypto_webhook_events
		set processing_attempts = processing_attempts + 1, last_error = $2 where event_id = $1`, id, msg)
}
