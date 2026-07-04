# components

Shared UI used by both the storefront (`app/(store)`) and the admin panel (`app/admin`).

| File | Purpose |
| --- | --- |
| `garments.tsx` | SVG garment renderer (shirt, sweater, hoodie, cap): paths, print areas, `GarmentBase`/`GarmentOverlay`, clip paths, dark-color helper. Used by the mockup editor and by `product-visual.tsx`. |
| `product-visual.tsx` | Product image. Renders the first uploaded image when the product has one; otherwise composes a live SVG mockup (garment + color + tech logo). |
| `product-card.tsx` | Catalog card: visual, garment/tech label, name, MXN price. Links to the product page. |
| `header.tsx` | Storefront header: nav by garment type, session state (log in / log out / admin shortcut), cart badge. |
| `footer.tsx` | Storefront footer. |
