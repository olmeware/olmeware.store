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
  page.tsx        — home page
  globals.css     — global styles
  admin/create/   — mockup editor: place logos on SVG garments, export PNG
public/           — static assets
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

- This is a very early stage project (v0.1.0). The store UI is not yet built.
- No backend, auth, or payment system is integrated yet.
- Supabase is available via MCP tools for when a database/auth layer is needed.
