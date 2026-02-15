import { useCallback, useEffect, useMemo, useState, type FormEvent } from 'react';
import type { Session } from '@supabase/supabase-js';
import {
  createLot,
  deleteLot,
  fetchLots,
  fetchPositions,
  searchAssets,
  updateLot
} from './lib/api';
import { getSupabase, isSupabaseConfigured } from './lib/supabase';
import type { Asset, Lot, LotDraft, Position } from './types';

const REFRESH_MS = (() => {
  const parsed = Number(import.meta.env.VITE_REFRESH_MS ?? '30000');
  if (!Number.isFinite(parsed) || parsed < 5000) {
    return 30000;
  }
  return parsed;
})();

const REALTIME_DEBOUNCE_MS = 500;

type RealtimeStatus = 'off' | 'connecting' | 'live' | 'error';
type AuthMode = 'signin' | 'signup';

function formatMoney(value: number | null): string {
  if (value === null) {
    return '--';
  }
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 2
  }).format(value);
}

function formatQuantity(value: number): string {
  return new Intl.NumberFormat('en-US', {
    maximumFractionDigits: 8
  }).format(value);
}

function toDateInputValue(isoTimestamp: string): string {
  const date = new Date(isoTimestamp);
  if (Number.isNaN(date.getTime())) {
    return new Date().toISOString().slice(0, 10);
  }
  return date.toISOString().slice(0, 10);
}

function realtimeStatusLabel(status: RealtimeStatus): string {
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

export function App() {
  const [session, setSession] = useState<Session | null>(null);
  const [booting, setBooting] = useState(true);

  const [authMode, setAuthMode] = useState<AuthMode>('signin');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [authBusy, setAuthBusy] = useState(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [authNotice, setAuthNotice] = useState<string | null>(null);

  const [positions, setPositions] = useState<Position[]>([]);
  const [lots, setLots] = useState<Lot[]>([]);
  const [assets, setAssets] = useState<Asset[]>([]);
  const [assetQuery, setAssetQuery] = useState('');

  const [dataLoading, setDataLoading] = useState(false);
  const [syncing, setSyncing] = useState(false);
  const [dataError, setDataError] = useState<string | null>(null);
  const [realtimeStatus, setRealtimeStatus] = useState<RealtimeStatus>('off');

  const [formBusy, setFormBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [editingLotID, setEditingLotID] = useState<number | null>(null);

  const [assetID, setAssetID] = useState<number | ''>('');
  const [quantity, setQuantity] = useState('');
  const [unitCost, setUnitCost] = useState('');
  const [purchasedAt, setPurchasedAt] = useState(new Date().toISOString().slice(0, 10));

  useEffect(() => {
    if (!isSupabaseConfigured) {
      setBooting(false);
      return;
    }

    const supabase = getSupabase();
    let alive = true;

    void supabase.auth
      .getSession()
      .then(({ data, error }) => {
        if (!alive) {
          return;
        }
        if (error) {
          setAuthError(error.message);
        }
        setSession(data.session);
      })
      .finally(() => {
        if (alive) {
          setBooting(false);
        }
      });

    const {
      data: { subscription }
    } = supabase.auth.onAuthStateChange((_event, nextSession) => {
      setSession(nextSession);
    });

    return () => {
      alive = false;
      subscription.unsubscribe();
    };
  }, []);

  const loadPortfolio = useCallback(
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
        void loadPortfolio(true);
      }, REALTIME_DEBOUNCE_MS);
    };

    setRealtimeStatus('connecting');
    void loadPortfolio(false);

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
      void loadPortfolio(true);
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
  }, [session, loadPortfolio]);

  useEffect(() => {
    if (!session) {
      setAssets([]);
      return;
    }

    let active = true;
    const timeoutID = window.setTimeout(() => {
      void searchAssets(assetQuery)
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
          setDataError(message);
        });
    }, 250);

    return () => {
      active = false;
      window.clearTimeout(timeoutID);
    };
  }, [session, assetQuery]);

  const assetOptions = useMemo(() => {
    const map = new Map<number, Asset>();
    for (const asset of assets) {
      map.set(asset.id, asset);
    }
    for (const lot of lots) {
      if (!map.has(lot.assetId)) {
        map.set(lot.assetId, {
          id: lot.assetId,
          symbol: lot.assetSymbol,
          name: lot.assetName,
          type: lot.assetType
        });
      }
    }
    return Array.from(map.values()).sort((a, b) => a.symbol.localeCompare(b.symbol));
  }, [assets, lots]);

  const portfolioValue = useMemo(
    () =>
      positions.reduce((acc, position) => {
        if (position.currentPrice === null) {
          return acc;
        }
        return acc + position.totalQty * position.currentPrice;
      }, 0),
    [positions]
  );

  const totalPL = useMemo(
    () => positions.reduce((acc, position) => acc + (position.unrealizedPL ?? 0), 0),
    [positions]
  );

  const handleAuthSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setAuthBusy(true);
    setAuthError(null);
    setAuthNotice(null);

    try {
      const supabase = getSupabase();
      if (authMode === 'signin') {
        const { error } = await supabase.auth.signInWithPassword({ email, password });
        if (error) {
          throw error;
        }
      } else {
        const { error } = await supabase.auth.signUp({ email, password });
        if (error) {
          throw error;
        }
        setAuthNotice('Account created. If email confirmation is enabled, check your inbox before signing in.');
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Authentication failed.';
      setAuthError(message);
    } finally {
      setAuthBusy(false);
    }
  };

  const resetForm = () => {
    setEditingLotID(null);
    setAssetID('');
    setQuantity('');
    setUnitCost('');
    setPurchasedAt(new Date().toISOString().slice(0, 10));
    setFormError(null);
  };

  const handleEditLot = (lot: Lot) => {
    setEditingLotID(lot.id);
    setAssetID(lot.assetId);
    setQuantity(String(lot.quantity));
    setUnitCost(String(lot.unitCost));
    setPurchasedAt(toDateInputValue(lot.purchasedAt));
    setFormError(null);
  };

  const handleSubmitLot = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!session) {
      return;
    }

    const parsedQuantity = Number(quantity);
    const parsedUnitCost = Number(unitCost);

    if (!assetID || !Number.isFinite(Number(assetID))) {
      setFormError('Select an asset.');
      return;
    }
    if (!Number.isFinite(parsedQuantity) || parsedQuantity <= 0) {
      setFormError('Quantity must be greater than 0.');
      return;
    }
    if (!Number.isFinite(parsedUnitCost) || parsedUnitCost < 0) {
      setFormError('Unit cost must be 0 or greater.');
      return;
    }
    if (!purchasedAt) {
      setFormError('Purchase date is required.');
      return;
    }

    const parsedDate = new Date(`${purchasedAt}T00:00:00Z`);
    if (Number.isNaN(parsedDate.getTime())) {
      setFormError('Purchase date is invalid.');
      return;
    }

    const draft: LotDraft = {
      assetId: Number(assetID),
      quantity: parsedQuantity,
      unitCost: parsedUnitCost,
      purchasedAtIso: parsedDate.toISOString()
    };

    setFormBusy(true);
    setFormError(null);

    try {
      if (editingLotID === null) {
        await createLot(session.user.id, draft);
      } else {
        await updateLot(editingLotID, draft);
      }
      await loadPortfolio(false);
      resetForm();
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to save lot.';
      setFormError(message);
    } finally {
      setFormBusy(false);
    }
  };

  const handleDeleteLot = async (lotID: number) => {
    const confirmed = window.confirm('Delete this lot?');
    if (!confirmed) {
      return;
    }

    try {
      await deleteLot(lotID);
      await loadPortfolio(false);
      if (editingLotID === lotID) {
        resetForm();
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to delete lot.';
      setDataError(message);
    }
  };

  const handleSignOut = async () => {
    if (!session) {
      return;
    }

    const supabase = getSupabase();
    const { error } = await supabase.auth.signOut();
    if (error) {
      setDataError(error.message);
    }
  };

  if (!isSupabaseConfigured) {
    return (
      <main className="shell">
        <section className="panel panel--centered">
          <h1>Asset Tracker UI</h1>
          <p>
            Missing Supabase config. Set <code>VITE_SUPABASE_URL</code> and{' '}
            <code>VITE_SUPABASE_PUBLISHABLE_KEY</code> (or <code>VITE_SUPABASE_ANON_KEY</code>) in{' '}
            <code>frontend/.env.local</code>.
          </p>
        </section>
      </main>
    );
  }

  if (booting) {
    return (
      <main className="shell">
        <section className="panel panel--centered">
          <h1>Asset Tracker UI</h1>
          <p>Loading session...</p>
        </section>
      </main>
    );
  }

  if (!session) {
    return (
      <main className="shell">
        <section className="panel auth-panel">
          <h1>Asset Tracker</h1>
          <p>Sign in to manage lots and see your live portfolio snapshot.</p>

          <form onSubmit={handleAuthSubmit} className="stack">
            <label>
              Email
              <input
                type="email"
                value={email}
                onChange={(event) => setEmail(event.target.value)}
                autoComplete="email"
                required
              />
            </label>

            <label>
              Password
              <input
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                autoComplete={authMode === 'signin' ? 'current-password' : 'new-password'}
                required
                minLength={6}
              />
            </label>

            <button type="submit" disabled={authBusy}>
              {authBusy ? 'Working...' : authMode === 'signin' ? 'Sign In' : 'Create Account'}
            </button>
          </form>

          <button
            type="button"
            className="button-link"
            onClick={() => setAuthMode(authMode === 'signin' ? 'signup' : 'signin')}
          >
            {authMode === 'signin' ? 'Need an account? Sign up' : 'Already have an account? Sign in'}
          </button>

          {authError && <p className="notice notice--error">{authError}</p>}
          {authNotice && <p className="notice">{authNotice}</p>}
        </section>
      </main>
    );
  }

  return (
    <main className="shell">
      <header className="app-header">
        <div>
          <h1>Asset Tracker</h1>
          <p>{session.user.email}</p>
        </div>

        <div className="header-actions">
          <button type="button" onClick={() => void loadPortfolio(false)} disabled={dataLoading || syncing}>
            {syncing ? 'Syncing...' : 'Refresh'}
          </button>
          <button type="button" onClick={() => void handleSignOut()} className="button-ghost">
            Sign Out
          </button>
        </div>
      </header>

      {dataError && <p className="notice notice--error">{dataError}</p>}

      <section className="metrics-grid">
        <article className="panel metric-card">
          <h2>Portfolio Value</h2>
          <p className="metric-value">{formatMoney(portfolioValue)}</p>
        </article>

        <article className="panel metric-card">
          <h2>Unrealized P/L</h2>
          <p className={`metric-value ${totalPL >= 0 ? 'positive' : 'negative'}`}>{formatMoney(totalPL)}</p>
        </article>

        <article className="panel metric-card">
          <h2>Open Positions</h2>
          <p className="metric-value">{positions.length}</p>
        </article>

        <article className="panel metric-card">
          <h2>Lots</h2>
          <p className="metric-value">{lots.length}</p>
        </article>
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>Positions</h2>
          <p>
            {realtimeStatusLabel(realtimeStatus)}. Polling fallback every {Math.round(REFRESH_MS / 1000)}s
          </p>
        </div>

        {dataLoading ? (
          <p>Loading portfolio...</p>
        ) : positions.length === 0 ? (
          <p>No positions yet. Add your first lot below.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Asset</th>
                  <th>Type</th>
                  <th>Qty</th>
                  <th>Avg Cost</th>
                  <th>Price</th>
                  <th>P/L</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((position) => (
                  <tr key={position.assetId}>
                    <td>
                      <strong>{position.symbol}</strong>
                      <span className="subtext">{position.name}</span>
                    </td>
                    <td>{position.type}</td>
                    <td>{formatQuantity(position.totalQty)}</td>
                    <td>{formatMoney(position.avgCost)}</td>
                    <td>{formatMoney(position.currentPrice)}</td>
                    <td
                      className={
                        position.unrealizedPL === null ? '' : position.unrealizedPL >= 0 ? 'positive' : 'negative'
                      }
                    >
                      {formatMoney(position.unrealizedPL)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section className="panel">
        <div className="section-head">
          <h2>{editingLotID === null ? 'Add Lot' : `Edit Lot #${editingLotID}`}</h2>
          {editingLotID !== null && (
            <button type="button" className="button-link" onClick={resetForm}>
              Cancel edit
            </button>
          )}
        </div>

        <form onSubmit={handleSubmitLot} className="lot-form">
          <label>
            Search Assets
            <input
              type="text"
              placeholder="BTC, AAPL, Bitcoin..."
              value={assetQuery}
              onChange={(event) => setAssetQuery(event.target.value)}
            />
          </label>

          <label>
            Asset
            <select
              value={assetID}
              onChange={(event) => setAssetID(event.target.value ? Number(event.target.value) : '')}
              required
              disabled={editingLotID !== null}
            >
              <option value="">Select an asset</option>
              {assetOptions.map((asset) => (
                <option key={asset.id} value={asset.id}>
                  {asset.symbol} - {asset.name} ({asset.type})
                </option>
              ))}
            </select>
          </label>

          <label>
            Quantity
            <input
              type="number"
              value={quantity}
              onChange={(event) => setQuantity(event.target.value)}
              min="0"
              step="any"
              required
            />
          </label>

          <label>
            Unit Cost (USD)
            <input
              type="number"
              value={unitCost}
              onChange={(event) => setUnitCost(event.target.value)}
              min="0"
              step="any"
              required
            />
          </label>

          <label>
            Purchase Date
            <input
              type="date"
              value={purchasedAt}
              onChange={(event) => setPurchasedAt(event.target.value)}
              required
            />
          </label>

          <button type="submit" disabled={formBusy}>
            {formBusy ? 'Saving...' : editingLotID === null ? 'Add Lot' : 'Save Changes'}
          </button>
        </form>

        {formError && <p className="notice notice--error">{formError}</p>}
      </section>

      <section className="panel">
        <h2>Lots</h2>
        {lots.length === 0 ? (
          <p>No lots yet.</p>
        ) : (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Asset</th>
                  <th>Type</th>
                  <th>Qty</th>
                  <th>Unit Cost</th>
                  <th>Purchased</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                {lots.map((lot) => (
                  <tr key={lot.id}>
                    <td>
                      <strong>{lot.assetSymbol}</strong>
                      <span className="subtext">{lot.assetName}</span>
                    </td>
                    <td>{lot.assetType}</td>
                    <td>{formatQuantity(lot.quantity)}</td>
                    <td>{formatMoney(lot.unitCost)}</td>
                    <td>{toDateInputValue(lot.purchasedAt)}</td>
                    <td>
                      <div className="row-actions">
                        <button type="button" className="button-link" onClick={() => handleEditLot(lot)}>
                          Edit
                        </button>
                        <button
                          type="button"
                          className="button-link danger"
                          onClick={() => void handleDeleteLot(lot.id)}
                        >
                          Delete
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>
    </main>
  );
}
