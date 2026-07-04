# lib

Shared, framework-agnostic data layer for the store. There is no backend yet: all data
lives in the browser's `localStorage`, seeded on first load, so the admin panel and the
storefront share the same catalog. When the real API exists, only `store.ts` needs to be
replaced.

| File | Purpose |
| --- | --- |
| `types.ts` | Domain types: `Product`, `Collection`, `CartItem`, `User`, `Session`, `DesignDraft`. |
| `constants.ts` | Garment/stack labels, sizes, garment color palette, MXN price formatter. |
| `seed.ts` | Initial catalog, collections, and the seeded admin user (`admin@olmeware.store` / `admin123`). |
| `store.ts` | `localStorage`-backed CRUD for products, collections, users, cart, session, and the mockup-editor design draft. Every write dispatches an `olmeware:store` event so open components refresh. |
| `hooks.ts` | React hooks (`useProducts`, `useCollections`, `useCart`, `useSession`, `useHydrated`) that subscribe to store changes. |
