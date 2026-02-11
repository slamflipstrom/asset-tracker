begin;

alter table public.assets
  add column if not exists market_data_id text,
  add column if not exists lookup_blockchain text,
  add column if not exists lookup_address text;

update public.assets
set market_data_id = coingecko_id
where type = 'crypto'
  and market_data_id is null
  and coingecko_id is not null;

drop index if exists assets_crypto_coingecko_id_idx;

create unique index if not exists assets_crypto_market_data_id_idx
  on public.assets (market_data_id)
  where type = 'crypto' and market_data_id is not null;

alter table public.assets
  drop constraint if exists assets_crypto_requires_coingecko_id;

alter table public.assets
  drop constraint if exists assets_crypto_requires_market_lookup;

alter table public.assets
  add constraint assets_crypto_requires_market_lookup
  check (
    type <> 'crypto'
    or market_data_id is not null
    or (lookup_blockchain is not null and lookup_address is not null)
  );

alter table public.assets
  drop column if exists coingecko_id;

commit;
