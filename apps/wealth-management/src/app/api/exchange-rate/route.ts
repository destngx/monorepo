import { NextResponse } from 'next/server';
import { getExchangeRate } from "@wealth-management/services/server";

export async function GET() {
  try {
    const rate = await getExchangeRate();
    return NextResponse.json({ rate });
  } catch {
    return NextResponse.json({ rate: 25400 });
  }
}
