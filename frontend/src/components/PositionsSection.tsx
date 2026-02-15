import { formatMoney, formatQuantity } from '../lib/format';
import type { Position } from '../types';

interface PositionsSectionProps {
  positions: Position[];
  dataLoading: boolean;
  realtimeLabel: string;
  refreshSeconds: number;
}

export function PositionsSection({ positions, dataLoading, realtimeLabel, refreshSeconds }: PositionsSectionProps) {
  return (
    <section className="panel">
      <div className="section-head">
        <h2>Positions</h2>
        <p>
          {realtimeLabel}. Polling fallback every {refreshSeconds}s
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
  );
}
