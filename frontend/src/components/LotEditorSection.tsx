import type { FormEvent } from 'react';
import type { Asset } from '../types';

interface LotEditorSectionProps {
  editingLotID: number | null;
  assetQuery: string;
  assetID: number | '';
  quantity: string;
  unitCost: string;
  purchasedAt: string;
  assetOptions: Asset[];
  formBusy: boolean;
  formError: string | null;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  onCancelEdit: () => void;
  onAssetQueryChange: (value: string) => void;
  onAssetIDChange: (value: number | '') => void;
  onQuantityChange: (value: string) => void;
  onUnitCostChange: (value: string) => void;
  onPurchasedAtChange: (value: string) => void;
}

export function LotEditorSection({
  editingLotID,
  assetQuery,
  assetID,
  quantity,
  unitCost,
  purchasedAt,
  assetOptions,
  formBusy,
  formError,
  onSubmit,
  onCancelEdit,
  onAssetQueryChange,
  onAssetIDChange,
  onQuantityChange,
  onUnitCostChange,
  onPurchasedAtChange
}: LotEditorSectionProps) {
  return (
    <section className="panel">
      <div className="section-head">
        <h2>{editingLotID === null ? 'Add Lot' : `Edit Lot #${editingLotID}`}</h2>
        {editingLotID !== null && (
          <button type="button" className="button-link" onClick={onCancelEdit}>
            Cancel edit
          </button>
        )}
      </div>

      <form onSubmit={onSubmit} className="lot-form">
        <label>
          Search Assets
          <input
            type="text"
            placeholder="BTC, AAPL, Bitcoin..."
            value={assetQuery}
            onChange={(event) => onAssetQueryChange(event.target.value)}
          />
        </label>

        <label>
          Asset
          <select
            value={assetID}
            onChange={(event) => onAssetIDChange(event.target.value ? Number(event.target.value) : '')}
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
            onChange={(event) => onQuantityChange(event.target.value)}
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
            onChange={(event) => onUnitCostChange(event.target.value)}
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
            onChange={(event) => onPurchasedAtChange(event.target.value)}
            required
          />
        </label>

        <button type="submit" disabled={formBusy}>
          {formBusy ? 'Saving...' : editingLotID === null ? 'Add Lot' : 'Save Changes'}
        </button>
      </form>

      {formError && <p className="notice notice--error">{formError}</p>}
    </section>
  );
}
