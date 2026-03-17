/**
 * Investments Feature - Mutation Layer
 */

export async function updateAsset(symbol: string, quantity: number): Promise<void> {
  const response = await fetch(`/api/investments/assets/${symbol}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ quantity }),
  });
  if (!response.ok) {
    throw new Error('Failed to update asset');
  }
}

export async function addAsset(symbol: string, quantity: number, buyPrice: number): Promise<void> {
  const response = await fetch('/api/investments/assets', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ symbol, quantity, buyPrice }),
  });
  if (!response.ok) {
    throw new Error('Failed to add asset');
  }
}

export async function removeAsset(symbol: string): Promise<void> {
  const response = await fetch(`/api/investments/assets/${symbol}`, {
    method: 'DELETE',
  });
  if (!response.ok) {
    throw new Error('Failed to remove asset');
  }
}
