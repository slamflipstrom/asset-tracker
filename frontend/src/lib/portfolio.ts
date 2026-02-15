import type { Asset, Lot, Position } from '../types';

export type RealtimeStatus = 'off' | 'connecting' | 'live' | 'error';

export function realtimeStatusLabel(status: RealtimeStatus): string {
  if (status === 'live') {
    return 'Live updates connected';
  }
  if (status === 'connecting') {
    return 'Connecting live updates...';
  }
  if (status === 'error') {
    return 'Live updates unavailable (polling fallback active)';
  }
  return 'Live updates offline';
}

export function buildAssetOptions(assets: Asset[], lots: Lot[]): Asset[] {
  const assetByID = new Map<number, Asset>();

  for (const asset of assets) {
    assetByID.set(asset.id, asset);
  }

  for (const lot of lots) {
    if (assetByID.has(lot.assetId)) {
      continue;
    }

    assetByID.set(lot.assetId, {
      id: lot.assetId,
      symbol: lot.assetSymbol,
      name: lot.assetName,
      type: lot.assetType
    });
  }

  return Array.from(assetByID.values()).sort((a, b) => a.symbol.localeCompare(b.symbol));
}

export function calculatePortfolioValue(positions: Position[]): number {
  return positions.reduce((total, position) => {
    if (position.currentPrice === null) {
      return total;
    }

    return total + position.totalQty * position.currentPrice;
  }, 0);
}

export function calculateTotalPL(positions: Position[]): number {
  return positions.reduce((total, position) => total + (position.unrealizedPL ?? 0), 0);
}
