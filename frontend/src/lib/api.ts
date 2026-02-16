import type { Asset, AssetType, Lot, LotDraft, Position } from '../types';
import { getSupabase } from './supabase';

const API_BASE_URL = (() => {
  const raw = (import.meta.env.VITE_API_BASE_URL ?? '').trim();
  return raw.replace(/\/+$/, '');
})();

interface APIErrorPayload {
  error?: string;
}

class APIError extends Error {
  readonly status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = 'APIError';
    this.status = status;
  }
}

interface PositionDTO {
  asset_id: number;
  symbol: string;
  name: string;
  type: string;
  total_qty: number;
  avg_cost: number;
  current_price: number | null;
  unrealized_pl: number | null;
}

interface LotDTO {
  id: number;
  asset_id: number;
  symbol: string;
  name: string;
  type: string;
  quantity: number;
  unit_cost: number;
  purchased_at: string;
}

interface AssetDTO {
  id: number;
  symbol: string;
  name: string;
  type: string;
}

function normalizeAssetType(value: unknown): AssetType {
  return value === 'stock' ? 'stock' : 'crypto';
}

function toNumber(value: unknown): number | null {
  if (value === null || value === undefined) {
    return null;
  }

  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

async function getAccessToken(): Promise<string> {
  const supabase = getSupabase();
  const { data, error } = await supabase.auth.getSession();
  if (error) {
    throw new Error(`Failed to read auth session: ${error.message}`);
  }

  const token = data.session?.access_token;
  if (!token) {
    throw new Error('Not authenticated. Please sign in again.');
  }

  return token;
}

async function parseErrorMessage(response: Response): Promise<string> {
  try {
    const body = (await response.json()) as APIErrorPayload;
    if (typeof body.error === 'string' && body.error.trim() !== '') {
      return body.error;
    }
  } catch {
    // Ignore parse errors and fall back to status text.
  }

  return response.statusText || `Request failed (${response.status})`;
}

async function apiRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = await getAccessToken();
  const headers = new Headers(init.headers);
  headers.set('Authorization', `Bearer ${token}`);
  headers.set('Accept', 'application/json');

  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(`${API_BASE_URL}${path}`, {
    ...init,
    headers
  });

  if (response.status === 204) {
    return undefined as T;
  }

  if (!response.ok) {
    const message = await parseErrorMessage(response);
    throw new APIError(response.status, message);
  }

  return (await response.json()) as T;
}

export async function fetchPositions(): Promise<Position[]> {
  const rows = await apiRequest<PositionDTO[]>('/api/v1/positions');

  return rows
    .map((row) => ({
      assetId: Number(row.asset_id),
      symbol: String(row.symbol ?? `#${row.asset_id}`),
      name: String(row.name ?? 'Unknown asset'),
      type: normalizeAssetType(row.type),
      totalQty: toNumber(row.total_qty) ?? 0,
      avgCost: toNumber(row.avg_cost) ?? 0,
      currentPrice: toNumber(row.current_price),
      unrealizedPL: toNumber(row.unrealized_pl)
    }))
    .sort((a, b) => a.symbol.localeCompare(b.symbol));
}

export async function fetchLots(): Promise<Lot[]> {
  const rows = await apiRequest<LotDTO[]>('/api/v1/lots');

  return rows.map((row) => ({
    id: Number(row.id),
    assetId: Number(row.asset_id),
    assetSymbol: String(row.symbol ?? `#${row.asset_id}`),
    assetName: String(row.name ?? 'Unknown asset'),
    assetType: normalizeAssetType(row.type),
    quantity: toNumber(row.quantity) ?? 0,
    unitCost: toNumber(row.unit_cost) ?? 0,
    purchasedAt: String(row.purchased_at)
  }));
}

export async function searchAssets(query: string, limit = 20): Promise<Asset[]> {
  const params = new URLSearchParams();
  const trimmedQuery = query.trim();
  if (trimmedQuery !== '') {
    params.set('q', trimmedQuery);
  }
  params.set('limit', String(limit));

  const rows = await apiRequest<AssetDTO[]>(`/api/v1/assets/search?${params.toString()}`);

  return rows.map((row) => ({
    id: Number(row.id),
    symbol: String(row.symbol),
    name: String(row.name),
    type: normalizeAssetType(row.type)
  }));
}

export async function createLot(draft: LotDraft): Promise<void> {
  await apiRequest<{ id: number }>('/api/v1/lots', {
    method: 'POST',
    body: JSON.stringify({
      asset_id: draft.assetId,
      quantity: draft.quantity,
      unit_cost: draft.unitCost,
      purchased_at: draft.purchasedAtIso
    })
  });
}

export async function updateLot(lotID: number, draft: LotDraft): Promise<void> {
  await apiRequest<void>(`/api/v1/lots/${lotID}`, {
    method: 'PATCH',
    body: JSON.stringify({
      quantity: draft.quantity,
      unit_cost: draft.unitCost,
      purchased_at: draft.purchasedAtIso
    })
  });
}

export async function deleteLot(lotID: number): Promise<void> {
  await apiRequest<void>(`/api/v1/lots/${lotID}`, {
    method: 'DELETE'
  });
}
