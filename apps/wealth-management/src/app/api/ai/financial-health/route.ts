import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildFinancialHealthPrompt } from '@wealth-management/ai/server';
import { Account, Loan } from '@wealth-management/types';

export async function POST(req: Request) {
  try {
    const body = await req.json() as {
      accounts: Account[];
      months: unknown[];
      loans?: Loan[];
    };
    const { accounts, months, loans = [] } = body;

    const totalLoanDebt = loans.reduce((s: number, l: Loan) => s + (l.yearlyRemaining || 0), 0);
    const netWorth = accounts.reduce((s: number, a: Account) => s + (a.balance || 0), 0) - totalLoanDebt;
    const totalAssets = accounts.filter((a: Account) => (a.balance || 0) >= 0).reduce((s: number, a: Account) => s + (a.balance || 0), 0);
    const totalLiabilities = accounts.filter((a: Account) => (a.balance || 0) < 0).reduce((s: number, a: Account) => s + Math.abs(a.balance || 0), 0) + totalLoanDebt;

    const taskInstruction = buildFinancialHealthPrompt({
      netWorth,
      totalAssets,
      totalLiabilities,
      loans,
      months
    });

    const result = await AIOrchestrator.runJson<Record<string, unknown>>({
      systemPromptInstruction: taskInstruction,
      prompt: "Perform a financial health analysis and return the results as JSON.",
    });

    return NextResponse.json(result);
  } catch (error: unknown) {
    console.error('AI Financial Health Error:', error);
    const message = error instanceof Error ? error.message : 'Failed to generate financial health analysis';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
