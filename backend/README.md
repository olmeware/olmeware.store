# Olmeware Store — Backend

Go API that powers the [Olmeware storefront](../CLAUDE.md): catalog, auth, cart, checkout,
inventory and payments (Stripe cards + Coinbase Commerce crypto).

The design rule of this service: **the backend composes, the frontend renders.** Prices
arrive preformatted, logo paths come from the database, availability is resolved
server-side. The Next.js client does no business logic.

| | |
| --- | --- |
| Language / runtime | Go 1.26 (stdlib `net/http`, method+path routing) |
| Database | Postgres (Supabase project `kimdkwjbuwgwyocjsthu`) via `pgx/v5` pool |
| Auth | HS256 JWT access tokens + opaque rotating refresh tokens |
| Payments | `stripe-go/v79` (cards) · Coinbase Commerce REST (BTC/ETH/SOL) |
| Port / prefix | `:8000`, every route under `/api/v1` |
| Currency | MXN, integer minor units (centavos) |
| Full endpoint reference | [`docs/v1.0.0/v1.0.0.md`](docs/v1.0.0/v1.0.0.md) |

---

## Quickstart

```bash
cd backend
go mod download
# create .env.local with at least DATABASE_URL (see Configuration below)

go run . -seed                      # idempotent: bootstrap admin + 16 products, then exit
go run .                            # start the API on http://localhost:8000

curl -s localhost:8000/api/v1/health
# {"service":"olmeware-backend","status":"ok","version":"v1.0.0"}
```

Point the frontend at it with `NEXT_PUBLIC_API_URL=http://localhost:8000/api/v1`
(that is already the default in `lib/api.ts`), then `pnpm dev` from the repo root.

Everything is one binary:

```bash
go build -trimpath -ldflags="-s -w" -o bin/olmeware .
./bin/olmeware              # serve
./bin/olmeware -seed        # seed and exit
```

---

## Configuration

Config is read from the environment; `.env.local` is loaded best-effort at startup
(`godotenv`), so real env vars in production work with no file present. `.env*` is
git-ignored — never commit keys.

| Variable | Required | Default | Notes |
| --- | :---: | --- | --- |
| `DATABASE_URL` | **yes** | — | Postgres/Supabase URL. Startup fails fast without it. |
| `JWT_SECRET` | prod | `dev-insecure-jwt-secret-change-me` | HS256 signing key. Rotating it invalidates every access token. |
| `PORT` | no | `8000` | |
| `FRONTEND_URL` | no | `http://localhost:3000` | CORS allow-list entry + crypto redirect/cancel URLs. Trailing slash trimmed. |
| `ACCESS_TOKEN_TTL` | no | `15m` | Go duration string. |
| `REFRESH_TOKEN_TTL` | no | `720h` (30d) | Go duration string. |
| `STRIPE_TEST_SECRET_KEY` / `STRIPE_SECRET_KEY` | no | — | Test key wins when both are set. |
| `STRIPE_TEST_PUBLISHABLE_KEY` / `STRIPE_PUBLISHABLE_KEY` | no | — | Returned to the browser by `/payments/config`. |
| `STRIPE_TEST_WEBHOOK_SECRET` / `STRIPE_WEBHOOK_SECRET` | no | — | `whsec_…`; required to accept Stripe webhooks. |
| `STRIPE_ALLOW_LIVE` | no | `false` | Safety latch — see below. |
| `COINBASE_COMMERCE_KEY` | no | — | Absent ⇒ crypto endpoints return `503`. |
| `COINBASE_COMMERCE_WEBHOOK_SECRET` | no | — | HMAC-SHA256 shared secret for `X-CC-Webhook-Signature`. |
| `ADMIN_EMAIL` | no | `admin@olmeware.store` | Bootstrap admin identity used by `-seed`. |
| `ADMIN_PASSWORD` | no | `admin123` | **Change before any public deploy.** |
| `ADMIN_NAME` | no | `Olmeware Admin` | |

**Live-key latch.** `config.StripeEnabled()` refuses a non-`sk_test` secret key unless
`STRIPE_ALLOW_LIVE=true`. Boot logs the resolved mode so you always know what you're
running:

```
olmeware payments: stripe=test crypto=false
olmeware payments: stripe=disabled (live key blocked; set STRIPE_ALLOW_LIVE=true to enable) crypto=true
```

---

## Architecture

```
backend/
├── main.go                  entrypoint: config → pool → (-seed | serve), graceful shutdown
├── Dockerfile               multi-stage static build, non-root runtime
├── database/
│   └── conn.go              pgxpool setup (10 max conns, ping on boot)
├── docs/v1.0.0/v1.0.0.md    the API contract (endpoint-by-endpoint)
└── internal/
    ├── config/              env loading, defaults, Stripe/Coinbase feature flags
    ├── server/              route table + middleware chain + /health   (integration test)
    ├── middleware/          Recover → Logger → CORS
    ├── httpx/               JSON writer, error envelope, strict decoder, client IP
    ├── money/               minor units → "$1,299 MXN"
    ├── auth/                JWT + refresh tokens, bcrypt, RequireAuth/RequireAdmin/OptionalAuth
    ├── catalog/             public read model (products, collections, tech themes)
    ├── cart/                guest + user carts, stock-aware mutations
    ├── orders/              checkout transaction, order read models
    ├── payments/            Stripe + Coinbase clients, idempotency, webhook inboxes
    ├── admin/               admin CRUD, slugify, audit log
    └── seed/                idempotent bootstrap (admin, collections, catalog, inventory)
```

**Layering.** Each feature package is `handler → service → repo`:

- `handler.go` — HTTP only: decode, call the service, write JSON. Never touches SQL.
- `service.go` — validation and business rules; returns `*httpx.APIError` for anything the
  client should see.
- `repo.go` — SQL and transactions; returns sentinel errors (`ErrNotFound`, `ErrOutOfStock`,
  …) that the service maps to HTTP status codes.
- `models.go` — request/response structs with the exact JSON shape the frontend consumes.

Handlers register themselves (`Register(mux, prefix)`), so `internal/server/server.go` is
the single place where the route table is visible.

**Request lifecycle**

```
request
  └─ Recover      panic → 500 JSON, server stays up
     └─ Logger    X-Request-ID, one stdout line: [id] METHOD path -> status (dur, bytes) ip=…
        └─ CORS   allow-list origin, preflight short-circuits with 204
           └─ mux (Go 1.22+ patterns: "POST /api/v1/cart/items")
              └─ RequireAuth | RequireAdmin | OptionalAuth   (per-route, wired at Register)
                 └─ handler → service → repo → Postgres
```

**Shutdown.** SIGINT/SIGTERM triggers `http.Server.Shutdown` with a 15 s drain, then the
pool closes. Timeouts: 10 s read-header, 30 s read/write, 120 s idle.

---

## API surface

Full request/response documentation lives in
[`docs/v1.0.0/v1.0.0.md`](docs/v1.0.0/v1.0.0.md). Route map:

| Group | Routes | Guard |
| --- | --- | --- |
| Health | `GET /health`, `GET /api/v1/health` | public (pings DB; `503 degraded` if down) |
| Auth | `POST /auth/register`, `/auth/login`, `/auth/refresh`, `/auth/logout` · `GET /auth/me` | public / `me` requires bearer |
| Catalog | `GET /catalog/products`, `/catalog/products/{slug}`, `/catalog/collections`, `/catalog/tech-themes` | public |
| Cart | `GET /cart` · `POST /cart/items` · `PUT\|DELETE /cart/items/{variantId}` · `DELETE /cart` | optional auth (guest via `X-Guest-Token`) |
| Orders | `POST /orders` (guest ok) · `GET /orders` · `GET /orders/{id}` | checkout optional-auth, reads require bearer |
| Admin | `GET\|POST /admin/products` · `PUT\|DELETE /admin/products/{id}` · `PATCH /admin/products/{id}/status` · `GET\|POST /admin/collections` · `PUT\|DELETE /admin/collections/{id}` | admin bearer (`403` otherwise) |
| Payments | `GET /payments/config` · `POST /payments/stripe/intent` · `POST /payments/crypto/charge` · `POST /payments/{stripe,crypto}/webhook` | optional auth; webhooks are signature-verified |

### Cross-cutting conventions

**Money.** Everything numeric is `*Minor` (`int64` centavos). Every response that carries a
`*Minor` field also carries the display string produced by `money.FormatMXN`, so the client
never formats: `44900 → "$449 MXN"`. The one exception is the catalog filter
`minPrice`/`maxPrice`, which take **major** pesos for readable URLs.

**Errors.** One envelope, always:

```json
{ "error": { "code": "conflict", "message": "Not enough stock for that quantity." } }
```

Codes: `bad_request` (400), `unauthorized` (401), `forbidden` (403), `not_found` (404),
`conflict` (409), `payments_unavailable` (503), `internal_error` (500). Anything that isn't
an `*httpx.APIError` is logged server-side and masked as `internal_error` — internal
details never reach a client.

**Request bodies.** `httpx.Decode` caps bodies at 1 MiB and sets `DisallowUnknownFields`,
so a typo'd field is a 400 rather than a silently ignored write.

**IDs.** Server-generated UUIDs (`gen_random_uuid()`); `orders.order_number` is a separate
human-facing identity column.

---

## Data model

The schema lives in Postgres (Supabase). Enums are real Postgres types, soft deletes are
`deleted_at is null` partial-unique indexes, and money columns are `bigint` minor units.

| Area | Tables |
| --- | --- |
| Identity | `users`, `user_sessions`, `addresses`, `customer_payment_profiles` |
| Catalog | `collections`, `tech_themes`, `products`, `product_collections`, `product_variants`, `product_media`, `product_designs` |
| Inventory | `inventory`, `inventory_movements` |
| Commerce | `carts`, `cart_items`, `orders`, `order_items` |
| Money | `payments`, `refunds`, `stripe_webhook_events`, `crypto_webhook_events` |
| Ops | `fulfillments`, `fulfillment_items`, `admin_audit_log` |

Enums: `user_role(customer, admin)` · `user_status(active, disabled)` ·
`product_status(draft, active, archived)` · `garment_type(shirt, sweater, hoodie, cap)` ·
`cart_status(active, converted, abandoned)` ·
`order_status(pending_payment, paid, processing, shipped, delivered, cancelled, refunded, partially_refunded)` ·
`payment_status(pending, requires_action, processing, succeeded, failed, cancelled, partially_refunded, refunded)` ·
`inventory_movement_type(initial, adjustment, reservation, reservation_release, sale, return)`.

### Invariants worth knowing

- `users_email_unique` — `unique (lower(email)) where deleted_at is null`. Emails are
  normalized to `lower(btrim(...))` in Go *and* CHECK-constrained in SQL.
- `carts_one_active_user_idx` / `carts_active_guest_token_idx` — at most one **active**
  cart per user and per guest token.
- `cart_items (cart_id, variant_id)` unique — a line is the variant; adds merge quantities.
- `product_variants (product_id, size, color_hex)` unique alive; SKUs unique alive.
- `payments (provider, idempotency_key)` unique — the once-only charge guarantee.
- `payments (provider, provider_payment_intent_id)` unique where not null.
- `orders.cart_id` unique — a cart converts into at most one order.
- `inventory_movements` is **append-only**; `inventory.on_hand/reserved` is the materialized
  view of that ledger and must be updated in the same transaction as the movement row.

> ⚠️ **Schema source of truth.** `database/scheme.sql`, `alter.sql` and `exec.sql` are
> currently **empty placeholders** — the live schema exists only in the Supabase project and
> its migration history. Dumping the schema into `database/scheme.sql` is the top item in
> [Known gaps](#known-gaps--roadmap); until then, reproduce an environment by cloning the
> Supabase project rather than by running these files.

RLS is enabled deny-by-default on every table. The backend connects as the owner role and
bypasses RLS — **authorization is enforced in Go**, not by the database.

---

## Auth model

Two roles. Registration always creates a `customer`; the single `admin` is created by
`-seed` from `ADMIN_EMAIL`/`ADMIN_PASSWORD` (re-running seed never clobbers a rotated
password — it only re-asserts the role).

- **Passwords**: bcrypt, cost 12. `password_hash` is `json:"-"`, so it cannot leak through a
  response struct.
- **Access token**: HS256 JWT, 15 min, `sub` = user id, custom `role` claim. Parsing pins
  `jwt.WithValidMethods([]string{"HS256"})` — no `alg: none` downgrade.
- **Refresh token**: 32 random bytes, base64url. Only its **SHA-256 hash** is stored in
  `user_sessions`; the raw value is returned to the client exactly once.
- **Rotation**: `/auth/refresh` revokes the presented token before minting a new pair, so a
  replayed refresh token is dead on arrival.
- **Middleware**: `RequireAuth` (401 without a valid bearer), `RequireAdmin` (+403 for
  non-admins), `OptionalAuth` (attaches a principal when present, never rejects — this is
  what lets guests use carts and checkout).

Login timing note: a missing user and a wrong password both return
`401 "Invalid email or password."`, but the missing-user path skips bcrypt and so answers
faster. Closing that user-enumeration side channel means comparing against a dummy hash.

---

## Carts & guest tokens

A cart belongs to a user (`user_id`) or to an anonymous visitor (`guest_token_hash`).

1. A tokenless guest POSTs to `/cart/items`. The service mints a 24-byte token, stores its
   SHA-256, and returns the raw token once as `guestToken` on the cart response.
2. The client persists it and sends `X-Guest-Token: <token>` on every later cart/checkout
   call.
3. An authenticated caller (Bearer) always resolves to the user's single active cart; the
   guest header is ignored.

Stock is validated on every mutation against `on_hand - reserved`, including the *resulting*
quantity on an increment — over-requesting returns `409`, not a clamped quantity. Reads
return an empty cart (`items: []`) rather than 404 so the UI has nothing to special-case.

> Guest→user cart merge on login is **not** implemented: signing in with items in a guest
> cart leaves those items behind in the guest cart.

---

## Checkout & the inventory ledger

`POST /orders` runs entirely inside one transaction (`orders/repo.go:CreateFromCart`):

1. `select … from cart_items … for update of i` — locks the inventory rows for every line,
   serializing concurrent checkouts of the same variant.
2. Validates `available >= quantity` per line → `ErrOutOfStock` → `409`.
3. Inserts the `orders` row (`pending_payment`, address JSONB snapshots, `placed_at`).
4. Inserts an `order_items` snapshot per line — name, SKU, size, color, logo, unit price —
   so later catalog edits never rewrite order history.
5. `inventory.reserved += qty` **and** an append-only `inventory_movements` row
   (`movement_type='reservation'`, `reference_type='order'`).
6. Marks the cart `converted` and links `converted_order_id`.

Stock is therefore *reserved* at checkout and only *sold* when payment succeeds. Totals:
`shipping_minor`, `tax_minor` and `discount_minor` are `0` in v1.0.0, so `total = subtotal`.

> Reservations are never released automatically. An order that is created and never paid
> holds its reservation until someone releases it — see [Known gaps](#known-gaps--roadmap).

---

## Payments

```
POST /orders                    → order (pending_payment, stock reserved)
POST /payments/stripe/intent    → { clientSecret, publishableKey, … }   ← Stripe Elements
        or
POST /payments/crypto/charge    → { hostedUrl, chargeCode, … }          ← Coinbase hosted page
                                        ↓
                            provider webhook (verified)
                                        ↓
        payment=succeeded · order pending_payment→paid · reservations→sales
```

**The order flips to `paid` only from a webhook.** A successful card confirmation in the
browser does not, by itself, change order state.

### Idempotency — charge once, no matter how many clicks

Three independent layers:

1. **Stable key per order+provider**: `stripe_order_<orderID>` / `coinbase_order_<orderID>`.
2. **Database**: `unique (provider, idempotency_key)` on `payments`, written with
   `on conflict do nothing`. Concurrent duplicate requests collapse onto one payment row.
3. **Provider**: the same key is sent as Stripe's `Idempotency-Key`, so even a retry that
   races past the DB returns Stripe's original PaymentIntent.

Repeat calls return the existing intent/charge with `"reused": true`. Before creating
anything, the service re-checks order state: already-paid orders get `409`, orders belonging
to another user get `403` (guest orders are reachable only via their unguessable UUID).

**Webhook exactly-once.** Every event is inserted into `stripe_webhook_events` /
`crypto_webhook_events` keyed by provider event id, `on conflict do nothing`. A duplicate
delivery inserts zero rows and returns immediately. Applying a success is itself guarded:
the payment update requires `status <> 'succeeded'` and the order update requires
`status = 'pending_payment'`, both `returning id` — so a redelivery can never double-decrement
inventory. Failures are recorded on the inbox row (`processing_attempts`, `last_error`) and
leave the order payable.

Signatures: Stripe via `webhook.ConstructEvent` on the **raw** body; Coinbase via
constant-time HMAC-SHA256 of the raw body against `X-CC-Webhook-Signature`. Both webhook
handlers read the body directly (never `httpx.Decode`) because signature verification needs
the exact bytes. Handled events: `payment_intent.succeeded`,
`payment_intent.payment_failed`, `charge:confirmed`, `charge:resolved`, `charge:failed`;
everything else is acknowledged and ignored.

### Local webhook testing

```bash
stripe listen --forward-to localhost:8000/api/v1/payments/stripe/webhook
# copy the printed whsec_… into STRIPE_TEST_WEBHOOK_SECRET and restart the API
stripe trigger payment_intent.succeeded
```

Without `stripe listen` running, a local end-to-end checkout will pay in the Stripe
dashboard but the order stays `pending_payment` — that is expected, not a bug. Coinbase has
no CLI forwarder; use a tunnel (`cloudflared`, `ngrok`) and register the public URL in the
Coinbase Commerce dashboard.

---

## Admin

Admin routes require an admin bearer token and are the only way to see `draft`/`archived`
products. Behavior worth noting:

- **Slugs** are derived from the name (`slugify`, matching the DB regex
  `^[a-z0-9]+(?:-[a-z0-9]+)*$`) and de-duplicated on collision.
- **Variants re-sync** on update: sizes in the payload are upserted and re-activated, sizes
  no longer listed are deactivated (never hard-deleted — order history keeps its FK).
- New variants get `inventory(on_hand = 1000, reorder_level = 0)`, i.e. effectively
  made-to-order, versus `100/10` for seeded products.
- **Deletes are soft** (`deleted_at = now()`), which is why every uniqueness index is
  partial on `deleted_at is null`.
- Every mutation writes an `admin_audit_log` row (`action`, `entity_type`, `entity_id`,
  `after_data`), inside the same transaction as the change where one exists.
- `status` is normalized: anything unrecognized becomes `draft`.

---

## Seeding

```bash
go run . -seed
```

Idempotent and transactional — safe to re-run at any time. It creates the bootstrap admin,
the three collections (`new-arrivals`, `classics`, `ai-drop`), 16 tech themes with their
`/logos/*.svg` paths, 16 products mirroring the frontend's `lib/seed.ts`, their variants
(66 SKUs) and `inventory` rows at 100 units each. Product/collection upserts key on `slug`,
so editing `internal/seed/data.go` and re-seeding updates rows in place.

---

## Testing

```bash
go test ./...                                    # unit tests, no database required
DATABASE_URL="postgres://…" go test ./...        # + full storefront→checkout integration test
go test -run TestStorefrontFlow ./internal/server -v
go vet ./...
```

Unit coverage: MXN formatting, slugify/status normalization, JWT round-trip + wrong-secret +
expiry, refresh-token hashing, bcrypt, Coinbase HMAC verification and payload parsing,
`X-Forwarded-For` client-IP parsing.

`internal/server/integration_test.go` spins up the real handler with `httptest`, then drives
register → catalog → add to cart → checkout → read orders against a live database, cleaning
up after itself (releases the reservation, deletes movements/order/user). It **skips** when
`DATABASE_URL` is unset or the database is unreachable, so CI without a DB stays green.

> If you run DB-backed tests through a sandboxed shell, the Postgres port is typically
> blocked — run those commands unsandboxed.

---

## Observability

Every request logs one line to stdout via the stdlib `log` package, prefixed `olmeware`:

```
olmeware 2026/07/22 02:41:07 [3f9c1a2b] POST /api/v1/cart/items -> 200 (12.184ms, 431 bytes) ip=127.0.0.1
```

The 8-char request id is also returned as `X-Request-ID`, so a client error report maps
straight to a log line. Panics log `PANIC recovered: …` and return a JSON 500. Payment
transitions log explicitly (`payment: stripe intent pi_… succeeded`). `GET /health` pings
the pool and answers `503 degraded` when the database is unreachable — wire it to your
platform's health check.

There is no log file writer; `logs/SYSTEM.log` and `logs/SYSTEM.csv` are empty placeholders.
Collect stdout at the platform level.

---

## Deployment

```bash
docker build -t olmeware-backend ./backend
docker run --rm -p 8000:8000 --env-file backend/.env.local olmeware-backend
```

`Dockerfile` is a two-stage build: `golang:1.26-alpine` compiles a static
(`CGO_ENABLED=0`, `-trimpath -ldflags="-s -w"`) binary, and `alpine:3.20` runs it as
non-root uid 10001 with `ca-certificates` for outbound TLS to Stripe/Coinbase. No config
files are baked in — everything comes from the environment.

Production checklist:

- [ ] Strong `JWT_SECRET`; rotated `ADMIN_PASSWORD` (never ship `admin123`).
- [ ] `FRONTEND_URL` set to the real origin — CORS still hardcodes `localhost:3000`
      alongside it, which you likely want to strip for production.
- [ ] Live Stripe keys **plus** `STRIPE_ALLOW_LIVE=true` (both, deliberately).
- [ ] Webhook endpoints registered with each provider and their secrets configured.
- [ ] Health check → `/health`; allow ≥15 s for graceful shutdown on deploy.
- [ ] Pool sizing: 10 connections per instance (`database/conn.go`) — check it against your
      Postgres/Supabase connection limit before scaling replicas.
- [ ] TLS/proxy in front; `X-Forwarded-For` is already honored for client IPs.

---

## Adding an endpoint

1. Add the request/response structs to the feature's `models.go` (JSON tags must match what
   the frontend consumes).
2. SQL and transactions go in `repo.go`, returning sentinel errors.
3. Validation and error mapping go in `service.go`, returning `*httpx.APIError`.
4. Add the route to that package's `Register(mux, prefix)` with the right guard
   (`RequireAuth` / `RequireAdmin` / `OptionalAuth` / none).
5. Document it in `docs/v1.0.0/v1.0.0.md` — the API contract lives there, and a breaking
   change means a new `/api/v2` prefix, not an edit in place.
6. `go test ./... && go vet ./...`.

Style: no comments unless the reason is non-obvious; no exotic dependencies; every money
value is minor units with a formatted sibling; every error the client sees is an
`httpx.APIError`.

---

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| `config: DATABASE_URL is required` | No `.env.local` in the working directory (it's loaded relative to CWD — run from `backend/`). |
| `database: ping database: …` at boot | Bad URL, network blocked, or a paused Supabase project. |
| `payments: stripe=disabled` in the boot log | No secret key, or a live key without `STRIPE_ALLOW_LIVE=true`. |
| `503 payments_unavailable` | Same as above, or `COINBASE_COMMERCE_KEY` unset for crypto. |
| Order stuck in `pending_payment` after paying locally | No `stripe listen` forwarding to `:8000`, or the wrong `whsec_…`. |
| `400 Invalid Stripe signature.` | Webhook secret mismatch, or a proxy that rewrote the raw body. |
| Frontend requests blocked by CORS | `FRONTEND_URL` doesn't match the browser origin exactly (scheme + port). |
| `400 Invalid request body: json: unknown field "x"` | Strict decoding — the client is sending a field the struct doesn't declare. |
| `409 Not enough stock` on a fresh catalog | Stale reservations from unpaid test orders; inspect `inventory` against `inventory_movements`. |

---

## Known gaps & roadmap

Deliberately tracked, in rough priority order:

1. **`database/scheme.sql` is empty** — the schema exists only in Supabase. Dump it (plus
   RLS policies and enums) so the database is reproducible from this repo.
2. **No reservation expiry** — abandoned `pending_payment` orders hold stock forever. Needs
   a sweeper that releases reservations (`reservation_release` movements) past a TTL, plus
   `carts.expires_at` cleanup for abandoned carts.
3. **No rate limiting** on `/auth/login`, `/auth/register` or `/auth/refresh`.
4. **Refunds and fulfillments have tables but no endpoints** (`refunds`, `fulfillments`,
   `fulfillment_items`), and there is no admin order-management surface — admins cannot list
   or advance orders through the API.
5. **`product_media` is never read** — `Product.images` is always `[]`, so the storefront
   always falls back to the live SVG mockup. Uploaded mockups need read + upload paths.
6. **No guest→user cart merge** on login.
7. **Login user-enumeration timing** — compare against a dummy bcrypt hash on the
   missing-user path.
8. **CORS hardcodes `localhost:3000`** in addition to `FRONTEND_URL`; make dev origins
   conditional on environment.
9. `product_designs`, `addresses` and `customer_payment_profiles` are modeled but unused —
   the mockup design draft still lives in the browser's `localStorage`.
