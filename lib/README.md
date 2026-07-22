# lib

Data layer for the store. It talks to the Olmeware Go backend over HTTP.

| File | Purpose |
| --- | --- |
| `types.ts` | Domain types: `Product`, `Collection`, `CartItem`, `User`, `Session`, `DesignDraft`. |
| `constants.ts` | Garment/stack labels, sizes, garment color palette, MXN price formatter. |
| `api.ts` | Typed client for the backend (`NEXT_PUBLIC_API_URL`, default `http://localhost:8000/api/v1`): auth (JWT access + rotating refresh, auto-refresh on 401), catalog, cart, orders, payments, admin. Holds tokens + guest-cart token in `localStorage`. |
| `store.ts` | The layer the UI imports. Fetches catalog/collections/session from `api.ts` into a module cache and dispatches `olmeware:store` so the sync hooks refresh; auth + admin writes call the backend; `checkout()` resolves cart lines to variants, creates the order, and clears the cart. The **cart** and mockup **design-draft** stay in `localStorage` (the cart carries per-line print customization that isn't a backend variant dimension). |
| `hooks.ts` | React hooks (`useProducts`, `useCollections`, `useCart`, `useSession`, `useSessionReady`, `useHydrated`) that read the cache via `useSyncExternalStore` and subscribe to store changes. |
| `logos.ts` | Devicon logo slugs used by the mockup editor / product form. |

Backend seeded/demo admin: `admin@olmeware.store` / `admin123`; customers register via the API.
