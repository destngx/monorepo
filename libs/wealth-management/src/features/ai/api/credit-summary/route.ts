import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import {
  extractAndParseJSON,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
  type StructuredInsight,
} from '@wealth-management/ai/server';
import { Transaction } from '@wealth-management/types';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      transactions: Transaction[];
      cardStats: unknown;
      targetMonth: string;
    };
    const { transactions, cardStats, targetMonth } = body;

    // targetMonth is "MM/YYYY" or "current"
    const isPastMonth = targetMonth && targetMonth !== 'current';
    const monthLabel = isPastMonth ? `for the month of ${targetMonth}` : 'for the current month';

    // Filter transactions to the selected month if provided
    let filteredTransactions = transactions;
    if (isPastMonth) {
      const [m, y] = targetMonth.split('/');
      filteredTransactions = transactions.filter((t: Transaction) => {
        const d = new Date(t.date);
        return d.getMonth() + 1 === parseInt(m) && d.getFullYear() === parseInt(y);
      });
    }

    const recentTransactions = filteredTransactions.slice(-100).map((t: Transaction) => ({
      payee: t.payee,
      category: t.category,
      amount: t.payment,
      tags: t.tags,
      date: t.date,
    }));

    const model = getLanguageModel('github-gpt-4.1');

    const taskTemplate = await loadTaskPrompt('credit-summary');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      monthLabel,
      cardStatsJson: JSON.stringify(cardStats),
      recentTransactionsJson: JSON.stringify(recentTransactions),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt = (await loadActionPrompt('credit-summary')) + '\n\n' + STRUCTURED_INSIGHT_FORMAT_INSTRUCTION;

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    try {
      const structured = extractAndParseJSON<StructuredInsight>(text);
      return NextResponse.json({ summary: structured });
    } catch {
      return NextResponse.json({ summary: text });
    }
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('AI Summary API Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Summary API Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
