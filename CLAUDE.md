# Olmeware Store — CLAUDE.md

## What is this project

Olmeware Store is a tech-culture clothing brand and e-commerce store selling shirts, sweaters, hoodies, and caps themed around programming languages, frameworks, developer tools, AI, and tech culture. The product catalog is driven by the icons and themes listed in `devicons.md`.

Future product lines may include intimate clothing and other apparel, but the current scope is shirts, sweaters, hoodies, and caps.

## Tech stack

- **Framework:** Next.js 16 (App Router)
- **Language:** TypeScript
- **Styling:** Tailwind CSS v4
- **Runtime/Package manager:** Node.js + pnpm
- **Linting:** ESLint (eslint-config-next)

## Project structure

```
app/
  layout.tsx      — root layout
  globals.css     — global styles
  (store)/        — storefront: home, shop (filtered catalog), product/[id], cart, login, register
  admin/          — admin panel: dashboard, products, collections, mockup editor (see app/admin/README.md)
components/       — shared UI: garment SVGs, product visual/card, header, footer
lib/              — localStorage data layer: types, constants, seed, store, hooks (see lib/README.md)
public/logos/     — devicon SVG logos used on merch mockups
devicons.md       — full catalog of tech themes for product designs
```

## Key commands

```bash
pnpm dev      # start dev server (localhost:3000)
pnpm build    # production build
pnpm start    # serve production build
pnpm lint     # run ESLint
```

## Product catalog

All tech themes for merch designs live in `devicons.md`, organized by category:

- Programming languages
- Web & backend frameworks
- CSS & styling
- Databases & ORMs
- AI & machine learning
- Cloud platforms
- DevOps & infrastructure
- Version control
- Build tools & package managers
- Testing
- Mobile
- APIs & protocols
- Security & networking
- Operating systems
- IDEs & editors
- Design & productivity tools
- Brands & companies
- Blockchain & Web3

When adding new products or design themes, check `devicons.md` first and keep it updated.

## Conventions

- Use App Router (not Pages Router).
- Co-locate components close to where they're used.
- Tailwind utility classes only — no custom CSS unless Tailwind can't handle it.
- No comments unless the reason is non-obvious.
- Keep components small and focused.

## Notes

- This is a very early stage project (v0.1.0). The storefront and admin frontends are built; data lives in a `localStorage` layer (`lib/store.ts`) seeded from `lib/seed.ts` until the API exists.
- No backend, payments, or real auth yet. Demo login: `admin@olmeware.store` / `admin123`; customers register client-side.
- Supabase is available via MCP tools for when a database/auth layer is needed.
