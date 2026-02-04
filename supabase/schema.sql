begin;

-- Types
create type public.asset_type as enum ('crypto', 'stock');

-- Core tables
create table if not exists public.assets (
  id bigserial primary key,
  symbol text not null,
  type public.asset_type not null,
  name text not null,
  created_at timestamptz not null default now(),
  unique (symbol, type)
);

create table if not exists public.profiles (
  id uuid primary key references auth.users(id) on delete cascade,
  email text,
  created_at timestamptz not null default now()
);

create table if not exists public.app_settings (
  id smallint primary key default 1,
  min_refresh_interval_sec integer not null default 60,
  max_refresh_interval_sec integer not null default 3600,
  updated_at timestamptz not null default now(),
  constraint app_settings_singleton check (id = 1),
  constraint app_settings_min_le_max check (min_refresh_interval_sec <= max_refresh_interval_sec)
);

insert into public.app_settings (id)
values (1)
on conflict do nothing;

create table if not exists public.user_settings (
  user_id uuid primary key references auth.users(id) on delete cascade,
  refresh_interval_sec integer not null default 300,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint user_settings_refresh_positive check (refresh_interval_sec > 0)
);

create table if not exists public.lots (
  id bigserial primary key,
  user_id uuid not null references auth.users(id) on delete cascade,
  asset_id bigint not null references public.assets(id) on delete cascade,
  quantity numeric(30, 10) not null,
  unit_cost numeric(30, 10) not null,
  purchased_at timestamptz not null,
  created_at timestamptz not null default now(),
  updated_at timestamptz not null default now(),
  constraint lots_quantity_positive check (quantity > 0),
  constraint lots_unit_cost_non_negative check (unit_cost >= 0)
);

create table if not exists public.prices_current (
  asset_id bigint primary key references public.assets(id) on delete cascade,
  price numeric(30, 10) not null,
  fetched_at timestamptz not null,
  provider text not null
);

create table if not exists public.price_snapshots (
  id bigserial primary key,
  asset_id bigint not null references public.assets(id) on delete cascade,
  price numeric(30, 10) not null,
  fetched_at timestamptz not null,
  provider text not null
);

-- Indexes
create index if not exists lots_user_id_idx on public.lots (user_id);
create index if not exists lots_asset_id_idx on public.lots (asset_id);
create index if not exists lots_user_asset_idx on public.lots (user_id, asset_id);
create index if not exists price_snapshots_asset_fetched_idx on public.price_snapshots (asset_id, fetched_at desc);

-- Helper functions and triggers
create or replace function public.set_updated_at()
returns trigger
language plpgsql
as $$
begin
  new.updated_at = now();
  return new;
end;
$$;

create or replace function public.clamp_refresh_interval()
returns trigger
language plpgsql
as $$
declare
  min_val integer;
  max_val integer;
begin
  select min_refresh_interval_sec, max_refresh_interval_sec
    into min_val, max_val
  from public.app_settings
  where id = 1;

  if min_val is not null and new.refresh_interval_sec < min_val then
    new.refresh_interval_sec = min_val;
  end if;
  if max_val is not null and new.refresh_interval_sec > max_val then
    new.refresh_interval_sec = max_val;
  end if;

  return new;
end;
$$;

create or replace function public.handle_new_user()
returns trigger
language plpgsql
security definer
set search_path = public
as $$
begin
  insert into public.profiles (id, email)
  values (new.id, new.email)
  on conflict do nothing;

  insert into public.user_settings (user_id)
  values (new.id)
  on conflict do nothing;

  return new;
end;
$$;

-- Triggers
create trigger lots_set_updated_at
before update on public.lots
for each row execute procedure public.set_updated_at();

create trigger user_settings_set_updated_at
before update on public.user_settings
for each row execute procedure public.set_updated_at();

create trigger app_settings_set_updated_at
before update on public.app_settings
for each row execute procedure public.set_updated_at();

create trigger user_settings_clamp_refresh_interval
before insert or update on public.user_settings
for each row execute procedure public.clamp_refresh_interval();

create trigger on_auth_user_created
after insert on auth.users
for each row execute procedure public.handle_new_user();

-- Views
create or replace view public.positions_view as
select
  l.user_id,
  l.asset_id,
  sum(l.quantity) as total_qty,
  sum(l.quantity * l.unit_cost) / nullif(sum(l.quantity), 0) as avg_cost,
  pc.price as current_price,
  (pc.price - (sum(l.quantity * l.unit_cost) / nullif(sum(l.quantity), 0))) * sum(l.quantity) as unrealized_pl
from public.lots l
left join public.prices_current pc on pc.asset_id = l.asset_id
group by l.user_id, l.asset_id, pc.price;

create or replace view public.lot_performance_view as
select
  l.id as lot_id,
  l.user_id,
  l.asset_id,
  l.quantity,
  l.unit_cost,
  l.purchased_at,
  pc.price as current_price,
  (pc.price - l.unit_cost) * l.quantity as unrealized_pl
from public.lots l
left join public.prices_current pc on pc.asset_id = l.asset_id;

commit;
