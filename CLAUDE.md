# Olmeware Store — Frontend Reference (CLAUDE.md)

> **Purpose of this file:** context reference for backend agents. It describes the
> **frontend** as it exists today so the backend can be designed to serve it. The
> `backend/` folder is intentionally **not** documented here — this is the client the
> backend must satisfy.

## What is this project

Tech-culture clothing brand and e-commerce store: shirts, sweaters, hoodies, and caps
themed around programming languages, frameworks, dev tools, and AI. Future lines may
include other apparel, but that's the current scope.

The frontend is fully built and currently runs against an in-browser `localStorage`
data layer seeded on first load. The backend's job is to replace that layer with a real
API + database + auth **without changing the shapes the UI already consumes.**

## Tech stack (frontend)

Next.js 16.2.9 (App Router) · React 19.2 · TypeScript 5 · Tailwind CSS v4 · pnpm ·
ESLint (eslint-config-next). No data-fetching library — all reads/writes go through
`lib/store.ts`.

## Project structure (frontend only)

```
app/
  layout.tsx        Root layout + pre-paint theme init script.
  globals.css       Tailwind import, dark-mode variant + color-variable remap.
  (store)/          Storefront (public): home, shop (filtered catalog), product/[id],
                    cart, login, register.               (see app/README.md)
  admin/            Admin panel (admin session required, guarded client-side):
                    dashboard, products, collections, mockup editor, logo library.
                                                          (see app/admin/README.md)
components/         Shared UI: garment SVGs, product visual/card, header, footer,
                    theme toggle.                         (see components/README.md)
lib/                Data layer + domain model: types, constants, seed, store, hooks.
                                                          (see lib/README.md)
public/logos/       devicon SVG logos (by category) used on merch mockups.
public/clothing/    Reference photos for icon placement on garments.
devicons.md         Catalog of tech themes for product designs, organized by category.
```

## Domain model (the data contract)

Defined in `lib/types.ts` — **this is the source of truth the backend must mirror.**

```ts
GarmentType = "shirt" | "sweater" | "hoodie" | "cap"
Side        = "front" | "back"
Stack       = "languages" | "frontend" | "backend" | "ai-ml" | "devops"
            | "databases" | "cloud" | "tools"
Size        = "XS" | "S" | "M" | "L" | "XL" | "XXL"
Role        = "admin" | "customer"
ProductStatus = "active" | "draft"

Product = {
  id, name, description,
  garment: GarmentType, stack: Stack, tech: string,
  price: number,              // MXN, integer (formatted es-MX, 0 decimals)
  sizes: Size[], color: string /* hex */,
  logo?: string,              // path under /public, e.g. "/logos/python.svg"
  images: string[],           // uploaded mockups; empty ⇒ render live SVG mockup
  collectionId?: string, featured?: boolean,
  status: ProductStatus, createdAt: string /* ISO */,
}

Collection   = { id, name, slug, description, createdAt }
Customization = { display?: "icon" | "icon-name", color?: string, position?: "left" | "center" | "right" }
CartItem     = Customization & { productId, size: Size, qty: number }
User         = { id, name, email, password, role: Role, createdAt }   // password is plaintext today
Session      = { userId, name, email, role: Role }
DesignDraft  = { garment: GarmentType, color: string, images: string[] }  // mockup-editor → new-product handoff
```

Notes for the backend:
- IDs are currently client-generated strings (`crypto.randomUUID()`, or hand-authored
  seed IDs like `prod-python-shirt`). The backend may switch to DB-generated IDs.
- Emails are normalized to `trim().toLowerCase()` before compare/store.
- `password` is stored/compared in plaintext client-side — **the backend must hash** and
  must never return it. `Session` is the safe, password-free projection of a `User`.
- Garments come in black (`#1a1a1a`) or white (`#f5f5f5`) only (`GARMENT_COLORS`);
  designs print on the **front only**. See "Product customization" below.

## Persistence & API surface the frontend expects

Everything the UI needs is the set of functions exported from `lib/store.ts`. Today they
read/write `localStorage`; the backend should expose equivalent operations (REST/RPC),
after which **only `lib/store.ts` needs to be swapped** — types, hooks, and pages stay put.

`localStorage` keys in use (seeded on first read where noted): `olmeware.products` (seeded),
`olmeware.collections` (seeded), `olmeware.users` (seeded), `olmeware.cart`,
`olmeware.session`, `olmeware.designDraft`.

Operations the UI relies on (from `lib/store.ts`):

| Domain | Functions | Backend equivalent |
| --- | --- | --- |
| Products | `getProducts`, `getProduct(id)`, `saveProduct` (upsert), `deleteProduct(id)` | CRUD `/products` |
| Collections | `getCollections`, `saveCollection` (upsert), `deleteCollection(id)` — delete also nulls `collectionId` on affected products | CRUD `/collections` (cascade unset) |
| Auth | `registerUser(name,email,password)`, `login(email,password)`, `logout()`, `getSession()` | `/auth/register`, `/auth/login`, `/auth/logout`, session/me |
| Cart | `getCart`, `addToCart`, `setCartQty`, `removeFromCart`, `clearCart` | per-user cart (see cart-line identity below) |
| Design draft | `saveDesignDraft`, `getDesignDraft`, `clearDesignDraft` | ephemeral editor→form handoff; likely stays client-side |

Result conventions the UI already handles: `registerUser` and `login` return
`{ ok: true, ... } | { ok: false, error: string }`. `registerUser` rejects duplicate
emails; `login` rejects bad credentials with `"Invalid email or password."`

**Cart-line identity:** a cart line is unique by `(productId, size, display, color, position)`
with defaults `display="icon"`, `color=""`, `position="center"` (see `sameLine` in
`lib/store.ts`). Two adds with the same customization merge quantities; different
customizations are separate lines. The backend must preserve this so quantities merge the
same way.

**Reactivity:** every write dispatches a `window` `olmeware:store` event; `lib/hooks.ts`
(`useProducts`, `useCollections`, `useCart`, `useSession`, `useHydrated`) subscribes via
`useSyncExternalStore` so open tabs/components refresh. A backend swap should keep the hook
signatures identical (they can move to fetch/websocket/polling under the hood).

## Auth model

- Two roles: `admin` and `customer`. Registration always creates a `customer`; admin
  accounts are seeded/created directly (future: in the DB).
- Admin panel is guarded **client-side** in `app/admin/layout.tsx` (redirect to `/login`).
  This is not real security — the backend must enforce role checks server-side.
- Seed/demo admin: `admin@olmeware.store` / `admin123`.

## Seed data

`lib/seed.ts` holds the initial catalog (16 products), 3 collections
(`New Arrivals`, `Classics`, `AI Drop`), and the seeded admin user. Useful as a fixture
set / migration seed when standing up the database. `devicons.md` + `public/logos/` are
the design-theme source of truth (check/update `devicons.md` when adding themes).

## Product customization (storefront)

On mockup products (no uploaded `images`) the customer picks color, icon display
(`icon` / `icon-name`), and icon position (`left`/`center`/`right` — wearer-relative,
chest height per `public/clothing/` reference images). Choices persist on cart lines
(`Customization`) and render via `components/product-visual.tsx`. When a product has
uploaded `images`, those are shown instead of the live SVG mockup.

## Theming (light/dark) — frontend concern, informational for backend

- Theme is a `data-theme="light" | "dark"` attribute on `<html>`, set before paint by an
  inline script in `app/layout.tsx` (reads localStorage `theme`, falls back to OS pref).
  `components/theme-toggle.tsx` switches/persists it.
- `app/globals.css` remaps Tailwind's `--color-*` variables under `:root[data-theme="dark"]`,
  so the whole site flips **without `dark:` classes** — don't add `dark:` color utilities;
  add missing dark values to globals.css instead.
- `theme-lock` class opts a subtree out of inversion (admin sidebar stays dark always).
- Product/garment mockup colors are literal hex and intentionally **not** themed.

## Key commands

```bash
pnpm dev      # dev server (localhost:3000)
pnpm build    # production build
pnpm start    # serve production build
pnpm lint     # ESLint
```

## Conventions

- App Router only; co-locate components close to where they're used.
- Tailwind utility classes only — no custom CSS unless Tailwind can't handle it.
- No comments unless the reason is non-obvious. Keep components small and focused.

## Status / notes

- v0.1.0. The frontend is wired to the Go backend (`backend/`) via `lib/api.ts` +
  `lib/store.ts`: catalog, JWT auth, admin writes (all statuses via `/admin/products`),
  and real checkout — create order (reserves inventory) → **Stripe Elements card capture**
  (`components/stripe-payment.tsx`, test mode) — all go through the API. The cart and mockup
  design-draft remain in `localStorage`. Backend also exposes crypto (Coinbase) + webhook
  endpoints. Note: order status flips to `paid` via the Stripe webhook, so local end-to-end
  "paid" state needs `stripe listen` forwarding to `:8000`.
- Supabase is available via MCP tools for the database/auth layer when the backend is built.
- Dev gotcha: a stale `pnpm dev` on port 3000 serves stale CSS — kill it
  (`lsof -nP -iTCP:3000`) before verifying style changes.
</content>
</invoke>
