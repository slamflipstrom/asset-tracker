import { formatMoney } from '../lib/format';

interface MetricsGridProps {
  portfolioValue: number;
  totalPL: number;
  openPositions: number;
  lotCount: number;
}

export function MetricsGrid({ portfolioValue, totalPL, openPositions, lotCount }: MetricsGridProps) {
  return (
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
        <p className="metric-value">{openPositions}</p>
      </article>

      <article className="panel metric-card">
        <h2>Lots</h2>
        <p className="metric-value">{lotCount}</p>
      </article>
    </section>
  );
}
