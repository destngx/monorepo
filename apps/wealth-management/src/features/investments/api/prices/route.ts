import { NextResponse } from 'next/server';
import { getPrice } from "@wealth-management/services/server";

export async function POST(request: Request) {
  try {
    const { symbols } = await request.json();

    if (!symbols || !Array.isArray(symbols)) {
      return NextResponse.json({ error: 'Invalid symbols list' }, { status: 400 });
    }

    const pricePromises = symbols.map(async (item: any) => {
      const symbol = typeof item === 'string' ? item : item.symbol;
      const type = typeof item === 'object' ? item.type : 'crypto';
      const price = await getPrice(symbol, type);
      return { symbol, price };
    });

    const prices = await Promise.all(pricePromises);

    // Convert to a map for easier frontend consumption
    const priceMap = prices.reduce((acc, { symbol, price }) => {
      acc[symbol] = price;
      return acc;
    }, {} as Record<string, number>);

    return NextResponse.json({ prices: priceMap });
  } catch (error) {
    console.error('[PricesAPI] Error:', error);
    return NextResponse.json({ error: 'Failed to fetch prices' }, { status: 500 });
  }
}
