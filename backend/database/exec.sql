begin;

insert into users (email, password_hash, full_name, role, status)
values (
    'admin@olmeware.store',
    '$2y$12$CneD7JuKdOSKDhkQDJr69.ySmlMEhLnV1O5NOq2QRr1l//HkhiJNK',
    'Olmeware Admin',
    'admin',
    'active'
)
on conflict (lower(email)) where deleted_at is null
do update set role = 'admin';

insert into collections (slug, name, description, status, sort_order)
values
    ('new-arrivals', 'New Arrivals', 'The latest drops, fresh from the print shop.', 'active', 0),
    ('classics', 'Classics', 'Timeless stacks that never go out of style.', 'active', 1),
    ('ai-drop', 'AI Drop', 'Wear the models that changed everything.', 'active', 2)
on conflict (slug) where deleted_at is null
do update set name = excluded.name,
              description = excluded.description,
              status = excluded.status,
              sort_order = excluded.sort_order;

create temporary table seed_products on commit drop as
select *
from jsonb_to_recordset($products$
[
  {"slug":"python-classic-tee","name":"Python Classic Tee","description":"Soft cotton tee with the Python logo front and center. Indentation not included.","garment":"shirt","stack":"languages","tech":"Python","logo_slug":"python","price_major":449,"color_hex":"#1a1a1a","sizes":["S","M","L","XL","XXL"],"collection_slug":"classics","featured":true},
  {"slug":"typescript-strict-tee","name":"TypeScript Strict Tee","description":"For those who never use any. Blue square, big energy, fully typed comfort.","garment":"shirt","stack":"languages","tech":"TypeScript","logo_slug":"typescript","price_major":449,"color_hex":"#f5f5f5","sizes":["XS","S","M","L","XL"],"collection_slug":"classics","featured":true},
  {"slug":"javascript-og-tee","name":"JavaScript OG Tee","description":"The language that runs the web, on the shirt that runs your wardrobe.","garment":"shirt","stack":"languages","tech":"JavaScript","logo_slug":"javascript","price_major":429,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"classics","featured":false},
  {"slug":"react-atomic-hoodie","name":"React Atomic Hoodie","description":"Heavyweight hoodie with the atom everyone re-renders for. Hooks sold separately.","garment":"hoodie","stack":"frontend","tech":"React","logo_slug":"react","price_major":899,"color_hex":"#1a1a1a","sizes":["S","M","L","XL","XXL"],"collection_slug":"new-arrivals","featured":true},
  {"slug":"vue-progressive-tee","name":"Vue Progressive Tee","description":"Approachable, performant, versatile. Also, it's green.","garment":"shirt","stack":"frontend","tech":"Vue","logo_slug":"vuejs","price_major":429,"color_hex":"#f5f5f5","sizes":["S","M","L","XL"],"collection_slug":"","featured":false},
  {"slug":"tailwind-utility-sweater","name":"Tailwind Utility Sweater","description":"flex items-center justify-cozy. A utility-first sweater for utility-first people.","garment":"sweater","stack":"frontend","tech":"Tailwind CSS","logo_slug":"tailwindcss","price_major":749,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"new-arrivals","featured":false},
  {"slug":"nextjs-edge-hoodie","name":"Next.js Edge Hoodie","description":"Server-rendered warmth with zero layout shift. Ships instantly to your closet.","garment":"hoodie","stack":"frontend","tech":"Next.js","logo_slug":"nextjs","price_major":949,"color_hex":"#1a1a1a","sizes":["S","M","L","XL","XXL"],"collection_slug":"new-arrivals","featured":true},
  {"slug":"nodejs-runtime-tee","name":"Node.js Runtime Tee","description":"Non-blocking, event-driven, machine-washable.","garment":"shirt","stack":"backend","tech":"Node.js","logo_slug":"nodejs","price_major":449,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"","featured":false},
  {"slug":"go-gopher-tee","name":"Go Gopher Tee","description":"Concurrency you can wear. Goroutines not included, gopher is.","garment":"shirt","stack":"backend","tech":"Go","logo_slug":"go","price_major":449,"color_hex":"#f5f5f5","sizes":["S","M","L","XL","XXL"],"collection_slug":"","featured":false},
  {"slug":"rust-fearless-hoodie","name":"Rust Fearless Hoodie","description":"Memory-safe warmth with zero-cost abstractions. The borrow checker approves.","garment":"hoodie","stack":"languages","tech":"Rust","logo_slug":"rust","price_major":899,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"classics","featured":false},
  {"slug":"docker-container-cap","name":"Docker Container Cap","description":"Works on your head. Works on every head. That's the point.","garment":"cap","stack":"devops","tech":"Docker","logo_slug":"docker","price_major":349,"color_hex":"#1a1a1a","sizes":["M","L"],"collection_slug":"","featured":false},
  {"slug":"kubernetes-helm-sweater","name":"Kubernetes Helm Sweater","description":"Self-healing comfort that scales with you. Warmth orchestrated across all pods.","garment":"sweater","stack":"devops","tech":"Kubernetes","logo_slug":"kubernetes","price_major":779,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"","featured":false},
  {"slug":"git-commit-cap","name":"Git Commit Cap","description":"git checkout style. Merge conflicts with your outfit resolved.","garment":"cap","stack":"tools","tech":"Git","logo_slug":"git","price_major":329,"color_hex":"#1a1a1a","sizes":["M","L"],"collection_slug":"","featured":false},
  {"slug":"linux-tux-tee","name":"Linux Tux Tee","description":"The penguin that powers the internet, now powering your look.","garment":"shirt","stack":"tools","tech":"Linux","logo_slug":"linux","price_major":449,"color_hex":"#f5f5f5","sizes":["S","M","L","XL","XXL"],"collection_slug":"","featured":false},
  {"slug":"pytorch-gradient-hoodie","name":"PyTorch Gradient Hoodie","description":"Backpropagate warmth through every layer. Autograd for your torso.","garment":"hoodie","stack":"ai-ml","tech":"PyTorch","logo_slug":"pytorch","price_major":929,"color_hex":"#1a1a1a","sizes":["S","M","L","XL"],"collection_slug":"ai-drop","featured":true},
  {"slug":"graphql-query-tee","name":"GraphQL Query Tee","description":"Ask for exactly what you want. Get exactly this shirt.","garment":"shirt","stack":"backend","tech":"GraphQL","logo_slug":"graphql","price_major":449,"color_hex":"#f5f5f5","sizes":["S","M","L","XL"],"collection_slug":"new-arrivals","featured":false}
]
$products$::jsonb) as p(
    slug text, name text, description text, garment text, stack text, tech text,
    logo_slug text, price_major integer, color_hex text, sizes text[],
    collection_slug text, featured boolean
);

insert into tech_themes (name, slug, category, logo_path, active)
select distinct tech, logo_slug,
       case stack
           when 'ai-ml' then 'ai-machine-learning'
           when 'devops' then 'devops-infrastructure'
           when 'languages' then 'languages'
           when 'frontend' then 'frontend'
           when 'backend' then 'backend'
           else 'tools'
       end,
       '/logos/' || logo_slug || '.svg', true
from seed_products
on conflict (slug) do update
set name = excluded.name,
    category = excluded.category,
    logo_path = excluded.logo_path,
    active = true;

insert into products (
    name, slug, description, garment, stack, tech_theme_id, tech_label, status,
    featured, default_color_hex, base_price_minor, currency, created_by,
    updated_by, published_at
)
select sp.name, sp.slug, sp.description, sp.garment, sp.stack, tt.id, sp.tech,
       'active', sp.featured, sp.color_hex, sp.price_major::bigint * 100, 'MXN',
       u.id, u.id, now()
from seed_products sp
join tech_themes tt on tt.slug = sp.logo_slug
cross join lateral (
    select id from users where lower(email) = 'admin@olmeware.store' and deleted_at is null
) u
on conflict (slug) where deleted_at is null do update
set name = excluded.name,
    description = excluded.description,
    garment = excluded.garment,
    stack = excluded.stack,
    tech_theme_id = excluded.tech_theme_id,
    tech_label = excluded.tech_label,
    status = excluded.status,
    featured = excluded.featured,
    default_color_hex = excluded.default_color_hex,
    base_price_minor = excluded.base_price_minor,
    updated_by = excluded.updated_by;

insert into product_variants (product_id, sku, size, color_name, color_hex, active)
select p.id,
       upper(replace(sp.slug, '-', '')) || '-' || size,
       size,
       case lower(sp.color_hex) when '#1a1a1a' then 'Black' when '#f5f5f5' then 'White' else '' end,
       sp.color_hex,
       true
from seed_products sp
join products p on p.slug = sp.slug and p.deleted_at is null
cross join lateral unnest(sp.sizes) size
on conflict (product_id, size, color_hex) where deleted_at is null do update
set sku = excluded.sku,
    color_name = excluded.color_name,
    active = true;

insert into inventory (variant_id, on_hand, reorder_level)
select pv.id, 100, 10
from seed_products sp
join products p on p.slug = sp.slug and p.deleted_at is null
join product_variants pv on pv.product_id = p.id
    and pv.color_hex = sp.color_hex
    and pv.size = any(sp.sizes)
    and pv.deleted_at is null
on conflict (variant_id) do nothing;

insert into product_collections (product_id, collection_id)
select p.id, c.id
from seed_products sp
join products p on p.slug = sp.slug and p.deleted_at is null
join collections c on c.slug = sp.collection_slug and c.deleted_at is null
where sp.collection_slug <> ''
on conflict (product_id, collection_id) do nothing;

commit;
