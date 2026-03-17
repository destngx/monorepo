import { NextResponse } from 'next/server';
import { getLoans } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET() {
  try {
    const loans = await getLoans();
    return NextResponse.json(loans);
  } catch (error: any) {
    return handleApiError(error, 'Loans');
  }
}
