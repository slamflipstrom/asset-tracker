import { useMemo, useState } from 'react';
import type { Session } from '@supabase/supabase-js';
import type { Asset, Lot, Position } from '../types';
import { useAssetSearch } from './useAssetSearch';
import { usePortfolioRealtime } from './usePortfolioRealtime';
import { usePortfolioSnapshot } from './usePortfolioSnapshot';

function buildAssetOptions(assets: Asset[], lots: Lot[]): Asset[] {
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

function calculatePortfolioValue(positions: Position[]): number {
  return positions.reduce((total, position) => {
    if (position.currentPrice === null) {
      return total;
    }

    return total + position.totalQty * position.currentPrice;
  }, 0);
}

function calculateTotalPL(positions: Position[]): number {
  return positions.reduce((total, position) => total + (position.unrealizedPL ?? 0), 0);
}

export function usePortfolioData(session: Session | null) {
  const [assetQuery, setAssetQuery] = useState('');

  const {
    positions,
    lots,
    dataLoading,
    syncing,
    dataError,
    setDataError,
    refreshPortfolio
  } = usePortfolioSnapshot(session);

  const assets = useAssetSearch(session, assetQuery, setDataError);
  const { realtimeStatus, refreshIntervalSeconds } = usePortfolioRealtime(session, refreshPortfolio);

  const assetOptions = useMemo(() => buildAssetOptions(assets, lots), [assets, lots]);
  const portfolioValue = useMemo(() => calculatePortfolioValue(positions), [positions]);
  const totalPL = useMemo(() => calculateTotalPL(positions), [positions]);

  return {
    positions,
    lots,
    assetQuery,
    setAssetQuery,
    dataLoading,
    syncing,
    dataError,
    setDataError,
    realtimeStatus,
    refreshPortfolio,
    refreshIntervalSeconds,
    assetOptions,
    portfolioValue,
    totalPL
  };
}
