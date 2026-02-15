import { formatMoney, formatQuantity, toDateInputValue } from '../lib/format';
import type { Lot } from '../types';

interface LotsSectionProps {
  lots: Lot[];
  onEdit: (lot: Lot) => void;
  onDelete: (lotID: number) => void;
}

export function LotsSection({ lots, onEdit, onDelete }: LotsSectionProps) {
  return (
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
                      <button type="button" className="button-link" onClick={() => onEdit(lot)}>
                        Edit
                      </button>
                      <button type="button" className="button-link danger" onClick={() => onDelete(lot.id)}>
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
  );
}
