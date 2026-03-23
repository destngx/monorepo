import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { AppError, isAppError } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { accounts, transactions, budget, loans, modelId } = body as {
      accounts?: Array<{ balance: number }>;
      transactions?: Array<{ date: string; deposit?: number; payment?: number }>;
      budget?: unknown[];
      loans?: Array<{ yearlyRemaining?: number }>;
      modelId?: string;
    };

    // Data Aggregation for AI
    const totalLoanDebt = loans?.reduce((s: number, l) => s + (l.yearlyRemaining || 0), 0) || 0;
    const totalAssets = accounts?.filter((a) => a.balance >= 0).reduce((s: number, a) => s + a.balance, 0) || 0;
    const totalLiabilities =
      (accounts?.filter((a) => a.balance < 0).reduce((s: number, a) => s + Math.abs(a.balance), 0) || 0) +
      totalLoanDebt;
    const netWorth = totalAssets - totalLiabilities;

    // Calculate Cash Flow (Current Month)
    const now = new Date();
    const currentMonthTxns =
      transactions?.filter((t) => {
        const d = new Date(t.date);
        return d.getMonth() === now.getMonth() && d.getFullYear() === now.getFullYear();
      }) || [];

    const income = currentMonthTxns.reduce((s: number, t) => s + (t.deposit || 0), 0);
    const expense = currentMonthTxns.reduce((s: number, t) => s + (t.payment || 0), 0);
    const cashFlow = income - expense;
    const savingsRate = income > 0 ? (cashFlow / income) * 100 : 0;

    const model = getLanguageModel(modelId || 'gpt-4o-mini');

    const taskTemplate = await loadTaskPrompt('intelligence-briefing');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      netWorth,
      totalAssets,
      totalLiabilities,
      cashFlow,
      savingsRate: savingsRate.toFixed(1),
      accountsJson: JSON.stringify(accounts?.slice(0, 5)),
      transactionsJson: JSON.stringify(transactions?.slice(0, 10)),
      budgetJson: JSON.stringify(budget?.slice(0, 5)),
      loansJson: JSON.stringify(loans),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt = await loadActionPrompt('intelligence-briefing');

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    const match = text.match(/\{[\s\S]*\}/);
    if (!match) throw new Error('Invalid AI response');

    // Robulst cleaning: Replace numbers formatted with dots (e.g. 1.234.567) into pure numbers (1234567)
    // This targets values following a colon in the JSON string
    const cleanedJson = match[0].replace(/:\s*(\d[\d.]+\d)/g, (m, p1) => {
      // If the number has multiple dots, it's definitely a formatted number (e.g. 2.916.834)
      // If it has one dot, it might be a decimal, but in this specific domain/VND,
      // most large numbers with dots are thousands separators.
      // For VND, we usually don't have decimals. Let's be aggressive for now.
      return `: ${p1.replace(/\./g, '')}`;
    });

    const result = JSON.parse(cleanedJson);

    return NextResponse.json(result);
  } catch (error) {
    const appError = isAppError(error)
      ? error
      : new AppError(error instanceof Error ? error.message : 'Failed to generate intelligence briefing');
    console.error('Intelligence Briefing Error:', appError.toResponse());
    return NextResponse.json(
      {
        briefing: 'Your financial dashboard is ready. Review your assets and liabilities to stay on track.',
        alerts: [],
      },
      { status: appError.statusCode },
    );
  }
}
