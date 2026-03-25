import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildBudgetAdvisorPrompt, loadActionPrompt } from '@wealth-management/ai/server';
import { BudgetItem, Transaction, Goal } from '@wealth-management/types';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';
import { getOrSetCache, CACHE_KEYS, CACHE_TTL } from '@/shared/cache';
import crypto from 'crypto';

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

    const cacheKey = `${CACHE_KEYS.AI_BUDGET_ADVISOR}:${crypto
      .createHash('md5')
      .update(JSON.stringify({ budget, transactions, goals, date, modelId }))
      .digest('hex')}`;

    const recentTxns = transactions.slice(-100);

    const taskInstruction = await buildBudgetAdvisorPrompt({
      date,
      budget,
      goals,
      recentTxns,
    });

    const actionPrompt = await loadActionPrompt('budget-advisor');

    const result = await getOrSetCache(
      cacheKey,
      () =>
        AIOrchestrator.runJson<Record<string, unknown>>({
          modelId: modelId,
          systemPromptInstruction: taskInstruction,
          prompt: actionPrompt,
        }),
      CACHE_TTL.AI_RESPONSES,
      false,
    );

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
