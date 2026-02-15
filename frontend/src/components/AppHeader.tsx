interface AppHeaderProps {
  email: string | undefined;
  syncing: boolean;
  dataLoading: boolean;
  onRefresh: () => void;
  onSignOut: () => void;
}

export function AppHeader({ email, syncing, dataLoading, onRefresh, onSignOut }: AppHeaderProps) {
  return (
    <header className="app-header">
      <div>
        <h1>Asset Tracker</h1>
        <p>{email}</p>
      </div>

      <div className="header-actions">
        <button type="button" onClick={onRefresh} disabled={dataLoading || syncing}>
          {syncing ? 'Syncing...' : 'Refresh'}
        </button>
        <button type="button" onClick={onSignOut} className="button-ghost">
          Sign Out
        </button>
      </div>
    </header>
  );
}
