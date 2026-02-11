begin;

alter table public.assets
  add column if not exists coingecko_id text;

update public.assets
set coingecko_id = lower(symbol)
where type = 'crypto'
  and coingecko_id is null;

create unique index if not exists assets_crypto_coingecko_id_idx
  on public.assets (coingecko_id)
  where type = 'crypto';

alter table public.assets
  drop constraint if exists assets_crypto_requires_coingecko_id;

alter table public.assets
  add constraint assets_crypto_requires_coingecko_id
  check (type <> 'crypto' or coingecko_id is not null);

commit;
