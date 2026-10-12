import type { ValuedHolding } from '$lib/api/types';

export const nf = new Intl.NumberFormat('vi-VN', { maximumFractionDigits: 8 });
export const quantity = (value: number | null) => (value === null ? '-' : nf.format(value));
export const holdingSource = (h: ValuedHolding) =>
  ({ quantity_x_price: 'qty × sheet price', quantity_x_live_price: 'qty × live price' })[h.valueSource];
