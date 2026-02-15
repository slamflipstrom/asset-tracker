import { useEffect, useState } from 'react';
import type { Session } from '@supabase/supabase-js';
import { getSupabase } from '../lib/supabase';

export type RealtimeStatus = 'off' | 'connecting' | 'live' | 'error';

const REFRESH_MS = (() => {
  const parsed = Number(import.meta.env.VITE_REFRESH_MS ?? '30000');
  if (!Number.isFinite(parsed) || parsed < 5000) {
    return 30000;
  }
  return parsed;
})();

const REALTIME_DEBOUNCE_MS = 500;

export function usePortfolioRealtime(
  session: Session | null,
  refreshPortfolio: (silent?: boolean) => Promise<void>
) {
  const [realtimeStatus, setRealtimeStatus] = useState<RealtimeStatus>('off');

  useEffect(() => {
    if (!session) {
      setRealtimeStatus('off');
      return;
    }

    const supabase = getSupabase();
    let active = true;
    let refreshTimeout: number | undefined;

    const scheduleRefresh = () => {
      if (!active || refreshTimeout !== undefined) {
        return;
      }

      refreshTimeout = window.setTimeout(() => {
        refreshTimeout = undefined;
        void refreshPortfolio(true);
      }, REALTIME_DEBOUNCE_MS);
    };

    setRealtimeStatus('connecting');
    void refreshPortfolio(false);

    const channel = supabase
      .channel(`portfolio-sync:${session.user.id}`)
      .on(
        'postgres_changes',
        {
          event: '*',
          schema: 'public',
          table: 'lots',
          filter: `user_id=eq.${session.user.id}`
        },
        scheduleRefresh
      )
      .on(
        'postgres_changes',
        {
          event: '*',
          schema: 'public',
          table: 'prices_current'
        },
        scheduleRefresh
      )
      .subscribe((status) => {
        if (!active) {
          return;
        }

        if (status === 'SUBSCRIBED') {
          setRealtimeStatus('live');
          return;
        }

        if (status === 'CHANNEL_ERROR' || status === 'TIMED_OUT') {
          setRealtimeStatus('error');
        }
      });

    const intervalID = window.setInterval(() => {
      void refreshPortfolio(true);
    }, REFRESH_MS);

    return () => {
      active = false;
      setRealtimeStatus('off');
      if (refreshTimeout !== undefined) {
        window.clearTimeout(refreshTimeout);
      }
      window.clearInterval(intervalID);
      void supabase.removeChannel(channel);
    };
  }, [session, refreshPortfolio]);

  return {
    realtimeStatus,
    refreshIntervalSeconds: Math.round(REFRESH_MS / 1000)
  };
}
