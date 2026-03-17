import { getCached, setCache } from '@wealth-management/utils';
import { executeSearch } from './search-service';
import { generateText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';

const CACHE_PREFIX = 'price:';
const CACHE_TTL = 3600; // 1 hour

// Curated mapping for common tokens to avoid search when possible
const SYMBOL_TO_CG_ID: Record<string, string> = {
  BTC: 'bitcoin',
  ETH: 'ethereum',
  BNB: 'binancecoin',
  SOL: 'solana',
  ADA: 'cardano',
  XRP: 'ripple',
  DOT: 'polkadot',
  DOGE: 'dogecoin',
  USDT: 'tether',
  USDC: 'usd-coin',
};

/**
 * Fetches the price of a given symbol/asset.
 * Tier 1: CoinGecko (for matched symbols)
 * Tier 2: AI Web Search (for funds, local assets, or unmatched crypto)
 */
export async function getPrice(symbol: string, type: 'crypto' | 'fund' = 'crypto'): Promise<number> {
  if (!symbol) return 0;

  const key = `${CACHE_PREFIX}${symbol.toUpperCase()}`;
  const cached = await getCached<{ price: number }>(key);
  if (cached) return cached.price;

  let price = 0;

  try {
    // Attempt Tier 1: CoinGecko for known crypto (only if type is crypto)
    if (type === 'crypto') {
      const cgId = SYMBOL_TO_CG_ID[symbol.toUpperCase()];
      if (cgId) {
        const response = await fetch(`https://api.coingecko.com/api/v3/simple/price?ids=${cgId}&vs_currencies=usd`);
        if (response.ok) {
          const data = await response.json();
          price = data[cgId]?.usd || 0;
        }
      }
    }

    // Attempt Tier 2: AI Search if Tier 1 failed or symbol is not crypto
    if (price === 0) {
      console.log(`[PriceService] AI Search triggered for (${type}): ${symbol}`);
      const today = new Date().toISOString().split('T')[0]; // YYYY-MM-DD
      const searchQuery =
        type === 'fund'
          ? `giá NAV/đơn vị quỹ ${symbol} mới nhất ngày ${today}`
          : `current price of ${symbol} in USD or VND today ${today}`;

      const { results, error } = await executeSearch(searchQuery);

      if (results && results.length > 0) {
        const context = results.map((r) => `${r.title}: ${r.description}`).join('\n');
        const model = getLanguageModel('github-gpt-4o');

        const { text } = await generateText({
          model,
          system: `You are a financial data extractor specializing in Vietnamese and Global markets. 
          Extract the CURRENT numeric price (NAV per unit) from the context.
          Today's date is ${today}. Always prefer the most recent price closest to today.
          
          CRITICAL RULES:
          1. Respond ONLY with the final number. No text, no currency symbols.
          2. Handle Vietnamese formatting: often "26.350" means twenty-six thousand three hundred fifty. 
          3. If the context says "26.350,00 VNĐ", you must return "26350".
          4. If found in USD, convert to VND using 25400 if it's a Vietnamese asset, otherwise return USD.
          5. If multiple values exist, pick the most recent NAV or Close price closest to ${today}.
          6. Output a clean float or integer string.`,
          prompt: `Context:\n${context}\n\nSymbol: ${symbol}\nAsset Type: ${type}\nDate: ${today}`,
        });

        // Clean result: remove dots (thousands) and fix commas (decimals)
        let cleaned = text.trim();
        if (cleaned.includes(',') && cleaned.includes('.')) {
          // Mixed format like 26.350,00 -> remove dots, then fix comma
          cleaned = cleaned.replace(/\./g, '').replace(',', '.');
        } else if (cleaned.includes(',')) {
          // If only comma exists, check if it's thousands or decimal
          // Usually in VN for large numbers like 26,350 it's thousands (US style used sometimes)
          // or 26.350 (VN style). AI prompt should have handled this, but let's be safe.
          if (cleaned.length > 4 && cleaned.indexOf(',') === cleaned.length - 3) {
            cleaned = cleaned.replace(',', '.'); // likely decimal 26,35
          } else {
            cleaned = cleaned.replace(',', ''); // likely thousands 26,350
          }
        }

        const extracted = parseFloat(cleaned);
        if (!isNaN(extracted)) {
          price = extracted;
        }
      }
    }

    if (price > 0) {
      await setCache(key, { price }, CACHE_TTL);
    }
  } catch (error) {
    console.error(`[PriceService] Failed to fetch price for ${symbol}:`, error);
  }

  return price;
}
