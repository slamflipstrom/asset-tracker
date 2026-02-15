import type { Asset, Lot, LotDraft, Position } from '../types';
import { getSupabase } from './supabase';

function toNumber(value: unknown): number | null {
  if (value === null || value === undefined) {
    return null;
  }
  const n = Number(value);
  return Number.isFinite(n) ? n : null;
}

function normalizeType(value: unknown): 'crypto' | 'stock' {
  return value === 'stock' ? 'stock' : 'crypto';
}

export async function fetchPositions(): Promise<Position[]> {
  const supabase = getSupabase();
  const { data: rows, error } = await supabase
    .from('positions_view')
    .select('asset_id,total_qty,avg_cost,current_price,unrealized_pl');

  if (error) {
    throw error;
  }

  const assetIDs = Array.from(
    new Set((rows ?? []).map((row) => Number(row.asset_id)).filter((id) => Number.isFinite(id) && id > 0))
  );

  const assetByID = new Map<number, Asset>();
  if (assetIDs.length > 0) {
    const { data: assetRows, error: assetError } = await supabase
      .from('assets')
      .select('id,symbol,name,type')
      .in('id', assetIDs);

    if (assetError) {
      throw assetError;
    }

    for (const row of assetRows ?? []) {
      const id = Number(row.id);
      if (!Number.isFinite(id)) {
        continue;
      }
      assetByID.set(id, {
        id,
        symbol: String(row.symbol ?? 'UNKNOWN'),
        name: String(row.name ?? 'Unknown asset'),
        type: normalizeType(row.type)
      });
    }
  }

  const positions: Position[] = (rows ?? []).map((row) => {
    const assetId = Number(row.asset_id);
    const asset = assetByID.get(assetId);

    return {
      assetId,
      symbol: asset?.symbol ?? `#${assetId}`,
      name: asset?.name ?? 'Unknown asset',
      type: asset?.type ?? 'crypto',
      totalQty: toNumber(row.total_qty) ?? 0,
      avgCost: toNumber(row.avg_cost) ?? 0,
      currentPrice: toNumber(row.current_price),
      unrealizedPL: toNumber(row.unrealized_pl)
    };
  });

  positions.sort((a, b) => a.symbol.localeCompare(b.symbol));
  return positions;
}

export async function fetchLots(): Promise<Lot[]> {
  const supabase = getSupabase();
  const { data, error } = await supabase
    .from('lots')
    .select('id,asset_id,quantity,unit_cost,purchased_at,assets(id,symbol,name,type)')
    .order('purchased_at', { ascending: false });

  if (error) {
    throw error;
  }

  return (data ?? []).map((row) => {
    const joined = Array.isArray(row.assets) ? row.assets[0] : row.assets;
    return {
      id: Number(row.id),
      assetId: Number(row.asset_id),
      assetSymbol: String(joined?.symbol ?? `#${row.asset_id}`),
      assetName: String(joined?.name ?? 'Unknown asset'),
      assetType: normalizeType(joined?.type),
      quantity: toNumber(row.quantity) ?? 0,
      unitCost: toNumber(row.unit_cost) ?? 0,
      purchasedAt: String(row.purchased_at)
    };
  });
}

export async function searchAssets(query: string, limit = 20): Promise<Asset[]> {
  const supabase = getSupabase();
  const term = query.trim();
  let request = supabase.from('assets').select('id,symbol,name,type').order('symbol').limit(limit);

  if (term.length > 0) {
    const safe = term.replace(/[%(),]/g, '');
    request = request.or(`symbol.ilike.%${safe}%,name.ilike.%${safe}%`);
  }

  const { data, error } = await request;
  if (error) {
    throw error;
  }

  return (data ?? []).map((row) => ({
    id: Number(row.id),
    symbol: String(row.symbol),
    name: String(row.name),
    type: normalizeType(row.type)
  }));
}

export async function createLot(userID: string, draft: LotDraft): Promise<void> {
  const supabase = getSupabase();
  const { error } = await supabase.from('lots').insert({
    user_id: userID,
    asset_id: draft.assetId,
    quantity: draft.quantity,
    unit_cost: draft.unitCost,
    purchased_at: draft.purchasedAtIso
  });

  if (error) {
    throw error;
  }
}

export async function updateLot(lotID: number, draft: LotDraft): Promise<void> {
  const supabase = getSupabase();
  const { error } = await supabase
    .from('lots')
    .update({
      quantity: draft.quantity,
      unit_cost: draft.unitCost,
      purchased_at: draft.purchasedAtIso
    })
    .eq('id', lotID);

  if (error) {
    throw error;
  }
}

export async function deleteLot(lotID: number): Promise<void> {
  const supabase = getSupabase();
  const { error } = await supabase.from('lots').delete().eq('id', lotID);

  if (error) {
    throw error;
  }
}
