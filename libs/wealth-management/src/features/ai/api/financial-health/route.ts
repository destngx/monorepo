import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildFinancialHealthPrompt, loadActionPrompt } from '@wealth-management/ai/server';
import { Account, Loan } from '@wealth-management/types';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      accounts: Account[];
      months: unknown[];
      loans?: Loan[];
    };
    const { accounts, months, loans = [] } = body;

    const totalLoanDebt = loans.reduce((s: number, l: Loan) => s + (l.yearlyRemaining || 0), 0);
    const netWorth = accounts.reduce((s: number, a: Account) => s + (a.balance || 0), 0) - totalLoanDebt;
    const totalAssets = accounts
      .filter((a: Account) => (a.balance || 0) >= 0)
      .reduce((s: number, a: Account) => s + (a.balance || 0), 0);
    const totalLiabilities =
      accounts
        .filter((a: Account) => (a.balance || 0) < 0)
        .reduce((s: number, a: Account) => s + Math.abs(a.balance || 0), 0) + totalLoanDebt;

    const taskInstruction = await buildFinancialHealthPrompt({
      netWorth,
      totalAssets,
      totalLiabilities,
      loans,
      months,
    });

    const actionPrompt = await loadActionPrompt('financial-health');

    const result = await AIOrchestrator.runJson<Record<string, unknown>>({
      systemPromptInstruction: taskInstruction,
      prompt: actionPrompt,
    });

    return NextResponse.json(result);
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('AI Financial Health Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Financial Health Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
