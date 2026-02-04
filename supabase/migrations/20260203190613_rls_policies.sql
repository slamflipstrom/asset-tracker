begin;

-- Enable RLS
alter table public.profiles enable row level security;
alter table public.user_settings enable row level security;
alter table public.lots enable row level security;
alter table public.assets enable row level security;
alter table public.prices_current enable row level security;
alter table public.price_snapshots enable row level security;

-- Profiles
create policy profiles_select_own
on public.profiles
for select
using (id = auth.uid());

create policy profiles_insert_own
on public.profiles
for insert
with check (id = auth.uid());

create policy profiles_update_own
on public.profiles
for update
using (id = auth.uid());

-- User settings
create policy user_settings_select_own
on public.user_settings
for select
using (user_id = auth.uid());

create policy user_settings_insert_own
on public.user_settings
for insert
with check (user_id = auth.uid());

create policy user_settings_update_own
on public.user_settings
for update
using (user_id = auth.uid());

-- Lots
create policy lots_select_own
on public.lots
for select
using (user_id = auth.uid());

create policy lots_insert_own
on public.lots
for insert
with check (user_id = auth.uid());

create policy lots_update_own
on public.lots
for update
using (user_id = auth.uid());

create policy lots_delete_own
on public.lots
for delete
using (user_id = auth.uid());

-- Assets (read-only for authenticated users)
create policy assets_select_authenticated
on public.assets
for select
to authenticated
using (true);

-- Prices (read-only for authenticated users)
create policy prices_current_select_authenticated
on public.prices_current
for select
to authenticated
using (true);

create policy price_snapshots_select_authenticated
on public.price_snapshots
for select
to authenticated
using (true);

commit;
