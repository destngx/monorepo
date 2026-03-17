import { NextResponse } from 'next/server';
import { getTransactions, addTransaction } from "@wealth-management/services/server";
import { TransactionSchema } from '@wealth-management/schemas';
import { getCategories } from "@wealth-management/services/server";
import { parseDate } from "@wealth-management/utils";
import { handleApiError } from "@wealth-management/utils/server";

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url);
    const forceFresh = searchParams.get('force') === 'true';

    const [transactions, categories] = await Promise.all([
      getTransactions(forceFresh),
      getCategories()
    ]);

    const enrichedTransactions = transactions.map(t => ({
      ...t,
      categoryType: categories.find(c => c.name.toLowerCase().trim() === t.category.toLowerCase().trim())?.type
    }));

    return NextResponse.json(enrichedTransactions);
  } catch (error) {
    return handleApiError(error, 'Transactions');
  }
}

export async function POST(request: Request) {
  try {
    const body = await request.json();

    // Standardize date parsing using our custom utility
    if (body.date) {
      body.date = parseDate(String(body.date));
    }

    const validatedData = TransactionSchema.parse(body);

    await addTransaction({
      ...validatedData,
      payment: validatedData.payment ?? null,
      deposit: validatedData.deposit ?? null,
      memo: validatedData.memo ?? null,
      referenceNumber: validatedData.referenceNumber ?? null,
    });
    return NextResponse.json({ success: true });
  } catch (error) {
    return handleApiError(error, 'Transactions');
  }
}
