/**
 * Investments Feature - Query Layer
 */

export interface Asset {
  symbol: string;
  quantity: number;
  currentPrice: number;
  value: number;
}

export async function getAssets() {
  const response = await fetch('/api/investments/assets');
  if (!response.ok) {
    throw new Error('Failed to fetch assets');
  }
  return response.json();
}

export async function getAssetPrices() {
  const response = await fetch('/api/investments/prices');
  if (!response.ok) {
    throw new Error('Failed to fetch asset prices');
  }
  return response.json();
}
