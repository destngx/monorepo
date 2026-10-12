import { MASK } from '$lib/format';

// Masks money-looking figures in model-written text (WM-UI-2), so AI prose
// does not leak amounts while the privacy mask is on. It errs on the side of
// hiding: any number tagged with a currency or a magnitude unit, and any
// thousand-grouped number, is replaced. Percentages, ratios and dates stay.

const NUMBER = String.raw`\d+(?:[.,]\d{3})*(?:[.,]\d+)?`;
const UNIT = String.raw`(?:triệu|tỷ|tr|bn|k|K|M|B)`;
const CURRENCY = String.raw`(?:VNĐ|VND|USDT|USD|đồng|₫|đ|\$)`;
const PREFIX = String.raw`(?:US\$|\$|₫|VNĐ|VND|USDT|USD)`;
// Not glued to a letter or digit on either side.
const START = String.raw`(?<![\p{L}\p{N}.,])`;
const END = String.raw`(?![\p{L}\p{N}])`;

const patterns = [
  // $1,200 / VND 1.500.000 / $1.5M USD
  `${PREFIX}\\s?[-+]?${NUMBER}(?:\\s?${UNIT})?(?:\\s?${CURRENCY})?${END}`,
  // 1.500.000 ₫ / 300k / 2,5 tr / 1.5M VND
  `[-+]?${NUMBER}(?:\\s?${UNIT}(?:\\s?${CURRENCY})?|\\s?${CURRENCY})${END}`,
  // 12.345.678 with no unit
  String.raw`[-+]?\d{1,3}(?:[.,]\d{3})+(?:[.,]\d+)?(?![\d%]|[.,]\d)`,
];

const money = new RegExp(`${START}(?:${patterns.join('|')})`, 'gu');

export function maskMoney(text: string): string {
  return text.replace(money, MASK);
}

/** The text as shown under the privacy mask: unchanged when revealed. */
export function maskText(text: string, revealed: boolean): string {
  return revealed ? text : maskMoney(text);
}
