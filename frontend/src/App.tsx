import { useEffect, useState, type FormEvent } from 'react';
import type { Session } from '@supabase/supabase-js';
import { AppHeader } from './components/AppHeader';
import { AuthPanel } from './components/AuthPanel';
import { LotEditorSection } from './components/LotEditorSection';
import { LotsSection } from './components/LotsSection';
import { MetricsGrid } from './components/MetricsGrid';
import { PositionsSection } from './components/PositionsSection';
import { usePortfolioData } from './hooks/usePortfolioData';
import { toDateInputValue } from './lib/format';
import { createLot, deleteLot, updateLot } from './lib/api';
import { realtimeStatusLabel } from './lib/portfolio';
import { getSupabase, isSupabaseConfigured } from './lib/supabase';
import type { Lot, LotDraft } from './types';

type AuthMode = 'signin' | 'signup';

function defaultPurchaseDate(): string {
  return new Date().toISOString().slice(0, 10);
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

  const [formBusy, setFormBusy] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [editingLotID, setEditingLotID] = useState<number | null>(null);

  const [assetID, setAssetID] = useState<number | ''>('');
  const [quantity, setQuantity] = useState('');
  const [unitCost, setUnitCost] = useState('');
  const [purchasedAt, setPurchasedAt] = useState(defaultPurchaseDate);

  const {
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
  } = usePortfolioData(session);

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
    setPurchasedAt(defaultPurchaseDate());
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
        await createLot(draft);
      } else {
        await updateLot(editingLotID, draft);
      }

      await refreshPortfolio(false);
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
      await refreshPortfolio(false);

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
        <AuthPanel
          authMode={authMode}
          email={email}
          password={password}
          authBusy={authBusy}
          authError={authError}
          authNotice={authNotice}
          onSubmit={handleAuthSubmit}
          onEmailChange={setEmail}
          onPasswordChange={setPassword}
          onToggleMode={() => setAuthMode((mode) => (mode === 'signin' ? 'signup' : 'signin'))}
        />
      </main>
    );
  }

  return (
    <main className="shell">
      <AppHeader
        email={session.user.email}
        syncing={syncing}
        dataLoading={dataLoading}
        onRefresh={() => void refreshPortfolio(false)}
        onSignOut={() => void handleSignOut()}
      />

      {dataError && <p className="notice notice--error">{dataError}</p>}

      <MetricsGrid
        portfolioValue={portfolioValue}
        totalPL={totalPL}
        openPositions={positions.length}
        lotCount={lots.length}
      />

      <PositionsSection
        positions={positions}
        dataLoading={dataLoading}
        realtimeLabel={realtimeStatusLabel(realtimeStatus)}
        refreshSeconds={refreshIntervalSeconds}
      />

      <LotEditorSection
        editingLotID={editingLotID}
        assetQuery={assetQuery}
        assetID={assetID}
        quantity={quantity}
        unitCost={unitCost}
        purchasedAt={purchasedAt}
        assetOptions={assetOptions}
        formBusy={formBusy}
        formError={formError}
        onSubmit={handleSubmitLot}
        onCancelEdit={resetForm}
        onAssetQueryChange={setAssetQuery}
        onAssetIDChange={setAssetID}
        onQuantityChange={setQuantity}
        onUnitCostChange={setUnitCost}
        onPurchasedAtChange={setPurchasedAt}
      />

      <LotsSection
        lots={lots}
        onEdit={handleEditLot}
        onDelete={(lotID) => {
          void handleDeleteLot(lotID);
        }}
      />
    </main>
  );
}
