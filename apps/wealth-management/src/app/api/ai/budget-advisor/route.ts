import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildBudgetAdvisorPrompt, loadActionPrompt } from '@wealth-management/ai/server';
import { BudgetItem, Transaction, Goal } from '@wealth-management/types';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      budget: BudgetItem[];
      transactions: Transaction[];
      goals: Goal[];
      date: string;
      modelId: string;
    };
    const { budget, transactions, goals, date, modelId } = body;

    // 2. Prepare 30-day projection context (last 100 txns)
    const recentTxns = transactions.slice(-100);

    const taskInstruction = await buildBudgetAdvisorPrompt({
      date,
      budget,
      goals,
      recentTxns,
    });

    const actionPrompt = await loadActionPrompt('budget-advisor');

    const result = await AIOrchestrator.runJson<Record<string, unknown>>({
      modelId: modelId,
      systemPromptInstruction: taskInstruction,
      prompt: actionPrompt,
    });

    return NextResponse.json(result);
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('AI Budget Advisor Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Budget Advisor Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
