begin;

create extension if not exists pgcrypto;

create type product_status as enum ('active', 'draft', 'archived');

create table users (
    id uuid primary key default gen_random_uuid(),
    email text not null,
    password_hash text not null,
    full_name text not null,
    role text not null default 'customer' check (role in ('admin', 'customer')),
    status text not null default 'active' check (status in ('active', 'disabled')),
    last_login_at timestamptz,
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);
create unique index users_email_live_uidx on users (lower(email)) where deleted_at is null;

create table user_sessions (
    id uuid primary key default gen_random_uuid(),
    user_id uuid not null references users(id) on delete cascade,
    refresh_token_hash text not null unique,
    user_agent text,
    ip_address inet,
    expires_at timestamptz not null,
    revoked_at timestamptz,
    created_at timestamptz not null default now()
);
create index user_sessions_user_id_idx on user_sessions (user_id);
create index user_sessions_active_idx on user_sessions (refresh_token_hash, expires_at) where revoked_at is null;

create table tech_themes (
    id uuid primary key default gen_random_uuid(),
    slug text not null unique,
    name text not null,
    category text not null,
    logo_path text,
    active boolean not null default true,
    created_at timestamptz not null default now()
);
create index tech_themes_active_category_idx on tech_themes (category, name) where active;

create table collections (
    id uuid primary key default gen_random_uuid(),
    slug text not null,
    name text not null,
    description text not null default '',
    status text not null default 'active' check (status in ('active', 'draft', 'archived')),
    sort_order integer not null default 0,
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);
create unique index collections_slug_live_uidx on collections (slug) where deleted_at is null;
create index collections_storefront_idx on collections (sort_order, name) where deleted_at is null and status = 'active';

create table products (
    id uuid primary key default gen_random_uuid(),
    slug text not null,
    name text not null,
    description text not null default '',
    garment text not null check (garment in ('shirt', 'sweater', 'hoodie', 'cap')),
    stack text not null check (stack in ('languages', 'frontend', 'backend', 'ai-ml', 'devops', 'databases', 'cloud', 'tools')),
    tech_theme_id uuid references tech_themes(id) on delete set null,
    tech_label text not null,
    status product_status not null default 'draft',
    featured boolean not null default false,
    default_color_hex text not null check (default_color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    base_price_minor bigint not null check (base_price_minor >= 0),
    currency text not null default 'MXN',
    created_by uuid references users(id) on delete set null,
    updated_by uuid references users(id) on delete set null,
    published_at timestamptz,
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);
create unique index products_slug_live_uidx on products (slug) where deleted_at is null;
create index products_storefront_idx on products (published_at desc, created_at desc) where deleted_at is null and status = 'active';
create index products_garment_idx on products (garment) where deleted_at is null and status = 'active';
create index products_stack_idx on products (stack) where deleted_at is null and status = 'active';

create table product_variants (
    id uuid primary key default gen_random_uuid(),
    product_id uuid not null references products(id) on delete cascade,
    sku text not null,
    size text not null check (size in ('XS', 'S', 'M', 'L', 'XL', 'XXL')),
    color_name text,
    color_hex text not null check (color_hex ~ '^#[0-9A-Fa-f]{6}$'),
    price_minor bigint check (price_minor is null or price_minor >= 0),
    active boolean not null default true,
    created_at timestamptz not null default now(),
    deleted_at timestamptz
);
create unique index product_variants_option_live_uidx
    on product_variants (product_id, size, color_hex) where deleted_at is null;
create index product_variants_sku_idx on product_variants (sku);
create index product_variants_product_active_idx on product_variants (product_id) where active and deleted_at is null;

create table product_collections (
    product_id uuid not null references products(id) on delete cascade,
    collection_id uuid not null references collections(id) on delete cascade,
    primary key (product_id, collection_id)
);
create index product_collections_collection_idx on product_collections (collection_id, product_id);

create table inventory (
    variant_id uuid primary key references product_variants(id) on delete cascade,
    on_hand integer not null default 0 check (on_hand >= 0),
    reserved integer not null default 0 check (reserved >= 0 and reserved <= on_hand),
    reorder_level integer not null default 0 check (reorder_level >= 0)
);

create table carts (
    id uuid primary key default gen_random_uuid(),
    user_id uuid references users(id) on delete cascade,
    guest_token_hash text,
    status text not null default 'active' check (status in ('active', 'converted', 'abandoned')),
    currency text not null default 'MXN',
    converted_order_id uuid,
    expires_at timestamptz,
    created_at timestamptz not null default now(),
    check ((user_id is not null) <> (guest_token_hash is not null))
);
create unique index carts_one_active_user_uidx on carts (user_id) where status = 'active' and user_id is not null;
create unique index carts_one_active_guest_uidx on carts (guest_token_hash) where status = 'active' and guest_token_hash is not null;

create table cart_items (
    cart_id uuid not null references carts(id) on delete cascade,
    variant_id uuid not null references product_variants(id) on delete cascade,
    quantity integer not null check (quantity > 0),
    created_at timestamptz not null default now(),
    primary key (cart_id, variant_id)
);

create table orders (
    id uuid primary key default gen_random_uuid(),
    order_number bigint generated always as identity unique,
    user_id uuid references users(id) on delete set null,
    cart_id uuid references carts(id) on delete set null,
    status text not null check (status in ('pending_payment', 'paid', 'processing', 'shipped', 'delivered', 'cancelled')),
    customer_email text not null,
    customer_name text not null,
    customer_phone text,
    currency text not null default 'MXN',
    subtotal_minor bigint not null check (subtotal_minor >= 0),
    discount_minor bigint not null default 0 check (discount_minor >= 0),
    shipping_minor bigint not null default 0 check (shipping_minor >= 0),
    tax_minor bigint not null default 0 check (tax_minor >= 0),
    total_minor bigint not null check (total_minor >= 0),
    shipping_address jsonb not null,
    billing_address jsonb,
    customer_note text,
    placed_at timestamptz,
    paid_at timestamptz,
    created_at timestamptz not null default now()
);
create index orders_user_created_idx on orders (user_id, created_at desc);
create index orders_status_idx on orders (status);
create unique index orders_cart_uidx on orders (cart_id) where cart_id is not null;

alter table carts add constraint carts_converted_order_fk
    foreign key (converted_order_id) references orders(id) on delete set null;

create table order_items (
    id uuid primary key default gen_random_uuid(),
    order_id uuid not null references orders(id) on delete cascade,
    variant_id uuid not null references product_variants(id),
    product_id uuid references products(id) on delete set null,
    sku text not null,
    product_name text not null,
    garment text not null,
    tech_label text not null,
    size text not null,
    color_hex text not null,
    unit_price_minor bigint not null check (unit_price_minor >= 0),
    quantity integer not null check (quantity > 0),
    line_total_minor bigint not null check (line_total_minor >= 0),
    product_snapshot jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);
create index order_items_order_created_idx on order_items (order_id, created_at);

create table inventory_movements (
    id uuid primary key default gen_random_uuid(),
    variant_id uuid not null references product_variants(id),
    movement_type text not null check (movement_type in ('reservation', 'sale', 'release', 'adjustment', 'restock')),
    quantity_delta integer not null default 0,
    reservation_delta integer not null default 0,
    reference_type text,
    reference_id uuid,
    created_at timestamptz not null default now()
);
create index inventory_movements_variant_created_idx on inventory_movements (variant_id, created_at);
create index inventory_movements_reference_idx on inventory_movements (reference_type, reference_id);

create table payments (
    id uuid primary key default gen_random_uuid(),
    order_id uuid not null references orders(id) on delete cascade,
    provider text not null check (provider = 'stripe'),
    status text not null check (status in ('processing', 'succeeded', 'failed', 'cancelled', 'refunded')),
    amount_minor bigint not null check (amount_minor >= 0),
    currency text not null,
    provider_payment_intent_id text,
    provider_charge_id text,
    idempotency_key text not null,
    failure_code text,
    failure_message text,
    succeeded_at timestamptz,
    created_at timestamptz not null default now(),
    unique (provider, idempotency_key)
);
create index payments_order_idx on payments (order_id);
create unique index payments_provider_intent_uidx on payments (provider, provider_payment_intent_id)
    where provider_payment_intent_id is not null;

create table stripe_webhook_events (
    event_id text primary key,
    event_type text not null,
    api_version text,
    livemode boolean not null,
    payload jsonb not null,
    processing_attempts integer not null default 0,
    last_error text,
    processed_at timestamptz,
    created_at timestamptz not null default now()
);

create table admin_audit_log (
    id uuid primary key default gen_random_uuid(),
    admin_user_id uuid references users(id) on delete set null,
    action text not null,
    entity_type text not null,
    entity_id uuid,
    after_data jsonb,
    created_at timestamptz not null default now()
);
create index admin_audit_log_admin_created_idx on admin_audit_log (admin_user_id, created_at desc);
create index admin_audit_log_entity_idx on admin_audit_log (entity_type, entity_id);

do $rls$
declare
    table_name text;
begin
    foreach table_name in array array[
        'users', 'user_sessions', 'tech_themes', 'collections', 'products',
        'product_variants', 'product_collections', 'inventory', 'carts',
        'cart_items', 'orders', 'order_items', 'inventory_movements', 'payments',
        'stripe_webhook_events', 'admin_audit_log'
    ] loop
        execute format('alter table %I enable row level security', table_name);
        execute format('revoke all on table %I from public', table_name);
        if exists (select 1 from pg_roles where rolname = 'anon') then
            execute format('revoke all on table %I from anon', table_name);
        end if;
        if exists (select 1 from pg_roles where rolname = 'authenticated') then
            execute format('revoke all on table %I from authenticated', table_name);
        end if;
    end loop;
end
$rls$;

revoke all on all sequences in schema public from public;

commit;
