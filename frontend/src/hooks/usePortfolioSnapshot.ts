import { useCallback, useEffect, useState } from 'react';
import type { Session } from '@supabase/supabase-js';
import { fetchLots, fetchPositions } from '../lib/api';
import type { Lot, Position } from '../types';

export function usePortfolioSnapshot(session: Session | null) {
  const [positions, setPositions] = useState<Position[]>([]);
  const [lots, setLots] = useState<Lot[]>([]);
  const [dataLoading, setDataLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [dataError, setDataError] = useState<string | null>(null);

  const refreshPortfolio = useCallback(
    async (silent = false) => {
      if (!session) {
        return;
      }

      if (silent) {
        setSyncing(true);
      } else {
        setDataLoading(true);
      }

      try {
        const [nextPositions, nextLots] = await Promise.all([fetchPositions(), fetchLots()]);
        setPositions(nextPositions);
        setLots(nextLots);
        setDataError(null);
      } catch (error) {
        const message = error instanceof Error ? error.message : 'Failed loading portfolio data.';
        setDataError(message);
      } finally {
        if (silent) {
          setSyncing(false);
        } else {
          setDataLoading(false);
        }
      }
    },
    [session]
  );

  useEffect(() => {
    if (!session) {
      setPositions([]);
      setLots([]);
      setDataLoading(false);
      setSyncing(false);
      setDataError(null);
    }
  }, [session]);

  return {
    positions,
    lots,
    dataLoading,
    syncing,
    dataError,
    setDataError,
    refreshPortfolio
  };
}
