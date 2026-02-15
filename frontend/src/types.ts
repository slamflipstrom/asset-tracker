export type AssetType = 'crypto' | 'stock';

export interface Asset {
  id: number;
  symbol: string;
  name: string;
  type: AssetType;
}

export interface Position {
  assetId: number;
  symbol: string;
  name: string;
  type: AssetType;
  totalQty: number;
  avgCost: number;
  currentPrice: number | null;
  unrealizedPL: number | null;
}

export interface Lot {
  id: number;
  assetId: number;
  assetSymbol: string;
  assetName: string;
  assetType: AssetType;
  quantity: number;
  unitCost: number;
  purchasedAt: string;
}

export interface LotDraft {
  assetId: number;
  quantity: number;
  unitCost: number;
  purchasedAtIso: string;
}
