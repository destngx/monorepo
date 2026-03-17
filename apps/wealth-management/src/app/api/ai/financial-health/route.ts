import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildFinancialHealthPrompt } from '@wealth-management/ai/server';

export async function POST(req: Request) {
  try {
    const { accounts, months, loans = [] } = await req.json();

    const totalLoanDebt = loans.reduce((s: number, l: any) => s + (l.yearlyRemaining || 0), 0);
    const netWorth = accounts.reduce((s: number, a: any) => s + (a.balance || 0), 0) - totalLoanDebt;
    const totalAssets = accounts.filter((a: any) => (a.balance || 0) >= 0).reduce((s: number, a: any) => s + (a.balance || 0), 0);
    const totalLiabilities = accounts.filter((a: any) => (a.balance || 0) < 0).reduce((s: number, a: any) => s + Math.abs(a.balance || 0), 0) + totalLoanDebt;

    const taskInstruction = buildFinancialHealthPrompt({
      netWorth,
      totalAssets,
      totalLiabilities,
      loans,
      months
    });

    const result = await AIOrchestrator.runJson<any>({
      systemPromptInstruction: taskInstruction,
      prompt: "Perform a financial health analysis and return the results as JSON.",
    });

    return NextResponse.json(result);
  } catch (error: any) {
    console.error('AI Financial Health Error:', error);
    return NextResponse.json({ error: error.message || 'Failed to generate financial health analysis' }, { status: 500 });
  }
}
