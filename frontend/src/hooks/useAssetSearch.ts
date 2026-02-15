import { useEffect, useState } from 'react';
import type { Session } from '@supabase/supabase-js';
import { searchAssets } from '../lib/api';
import type { Asset } from '../types';

const SEARCH_DEBOUNCE_MS = 250;

export function useAssetSearch(
  session: Session | null,
  query: string,
  onError: (message: string) => void
): Asset[] {
  const [assets, setAssets] = useState<Asset[]>([]);

  useEffect(() => {
    if (!session) {
      setAssets([]);
      return;
    }

    let active = true;
    const timeoutID = window.setTimeout(() => {
      void searchAssets(query)
        .then((results) => {
          if (active) {
            setAssets(results);
          }
        })
        .catch((error) => {
          if (!active) {
            return;
          }

          const message = error instanceof Error ? error.message : 'Asset search failed.';
          onError(message);
        });
    }, SEARCH_DEBOUNCE_MS);

    return () => {
      active = false;
      window.clearTimeout(timeoutID);
    };
  }, [session, query, onError]);

  return assets;
}
