import { NextResponse } from 'next/server';
import { getPrice } from '@wealth-management/services/server';
import { setCacheValue, getCacheValue, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';

export async function POST(request: Request) {
  try {
    const { symbols } = await request.json();

    if (!symbols || !Array.isArray(symbols)) {
      return NextResponse.json({ error: 'Invalid symbols list' }, { status: 400 });
    }

    const pricePromises = symbols.map(async (item: any) => {
      const symbol = typeof item === 'string' ? item : item.symbol;
      const type = typeof item === 'object' ? item.type : 'crypto';

      const cacheKey = `${CACHE_KEYS.PRICE_SYMBOL(symbol)}:${type}`;
      const cached = await getCacheValue<number>(cacheKey);

      if (cached.success && cached.data !== undefined) {
        return { symbol, price: cached.data };
      }

      const price = await getPrice(symbol, type);
      await setCacheValue(cacheKey, price, CACHE_TTL.MARKET_DATA);
      return { symbol, price };
    });

    const prices = await Promise.all(pricePromises);

    const priceMap = prices.reduce(
      (acc, { symbol, price }) => {
        acc[symbol] = price;
        return acc;
      },
      {} as Record<string, number>,
    );

    return NextResponse.json({ prices: priceMap });
  } catch (error) {
    console.error('[PricesAPI] Error:', error);
    return NextResponse.json({ error: 'Failed to fetch prices' }, { status: 500 });
  }
}
