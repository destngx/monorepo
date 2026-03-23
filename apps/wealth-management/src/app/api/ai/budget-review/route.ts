import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { BudgetItem, Transaction } from '@wealth-management/types';
import {
  extractAndParseJSON,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
  type StructuredInsight,
} from '@wealth-management/ai/server';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      budget: BudgetItem[];
      transactions: Transaction[];
      totalSpent: number;
      totalLimit: number;
      view: string;
      date: string;
    };
    const { budget, transactions, totalSpent, totalLimit, view, date } = body;

    const recentTransactions = transactions.slice(-30).map((t: Transaction) => ({
      category: t.category,
      amount: t.payment || 0,
      deposit: t.deposit || 0,
      date: t.date,
      payee: t.payee,
    }));

    const activeBudgets = budget
      .filter((b: BudgetItem) => b.monthlySpent > 0 || (b.monthlyLimit && b.monthlyLimit > 0))
      .map((b: BudgetItem) => ({
        category: b.category,
        spent: b.monthlySpent,
        limit: b.monthlyLimit,
        remaining: b.monthlyRemaining,
      }));

    const model = getLanguageModel('github-gpt-4o');

    const taskTemplate = await loadTaskPrompt('budget-review');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      view,
      date,
      totalSpent,
      totalLimit,
      usagePercentage: totalLimit > 0 ? Math.round((totalSpent / totalLimit) * 100) : 0,
      activeBudgetsJson: JSON.stringify(activeBudgets),
      recentTransactionsJson: JSON.stringify(recentTransactions),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionTemplate = await loadActionPrompt('budget-review');
    const actionPrompt =
      replacePlaceholders(actionTemplate, { view, date }) + '\n\n' + STRUCTURED_INSIGHT_FORMAT_INSTRUCTION;

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    try {
      const structured = extractAndParseJSON<StructuredInsight>(text);
      return NextResponse.json({ review: structured });
    } catch {
      return NextResponse.json({ review: text });
    }
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('AI Budget Review Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Budget Review Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
