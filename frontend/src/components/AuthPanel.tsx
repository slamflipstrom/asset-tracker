import type { FormEvent } from 'react';

type AuthMode = 'signin' | 'signup';

interface AuthPanelProps {
  authMode: AuthMode;
  email: string;
  password: string;
  authBusy: boolean;
  authError: string | null;
  authNotice: string | null;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onToggleMode: () => void;
}

export function AuthPanel({
  authMode,
  email,
  password,
  authBusy,
  authError,
  authNotice,
  onSubmit,
  onEmailChange,
  onPasswordChange,
  onToggleMode
}: AuthPanelProps) {
  const isSignIn = authMode === 'signin';

  return (
    <section className="panel auth-panel">
      <h1>Asset Tracker</h1>
      <p>Sign in to manage lots and see your live portfolio snapshot.</p>

      <form onSubmit={onSubmit} className="stack">
        <label>
          Email
          <input
            type="email"
            value={email}
            onChange={(event) => onEmailChange(event.target.value)}
            autoComplete="email"
            required
          />
        </label>

        <label>
          Password
          <input
            type="password"
            value={password}
            onChange={(event) => onPasswordChange(event.target.value)}
            autoComplete={isSignIn ? 'current-password' : 'new-password'}
            required
            minLength={6}
          />
        </label>

        <button type="submit" disabled={authBusy}>
          {authBusy ? 'Working...' : isSignIn ? 'Sign In' : 'Create Account'}
        </button>
      </form>

      <button type="button" className="button-link" onClick={onToggleMode}>
        {isSignIn ? 'Need an account? Sign up' : 'Already have an account? Sign in'}
      </button>

      {authError && <p className="notice notice--error">{authError}</p>}
      {authNotice && <p className="notice">{authNotice}</p>}
    </section>
  );
}
