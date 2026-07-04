# app

Next.js App Router tree. Two areas share one root layout:

```
(store)/            Storefront (public). Wrapped by the store layout (header + footer).
  page.tsx          Home: hero, garment tiles, featured, collections, newest drops.
  shop/             Catalog with filters (garment, stack, size, collection, price, search, sort).
  product/[id]/     Product detail: mockup gallery (front/back), sizes, quantity, add to cart, related.
  cart/             Cart with quantity editing and a demo checkout.
  login/            Shared login for customers and admins (admins are redirected to /admin).
  register/         Customer registration (admin accounts will be created directly in the database later).

admin/              Admin panel (requires an admin session; guarded client-side by its layout).
  page.tsx          Dashboard: catalog stats and recent products.
  products/         Product list plus the "new product" form.
  collections/      Create and manage collections.
  create/           Mockup editor: compose garment + logos, export PNG or send the design to a new product.
```

All data comes from `lib/store.ts` (localStorage), so pages are client components.
