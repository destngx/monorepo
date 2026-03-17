import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildBudgetAdvisorPrompt } from '@wealth-management/ai/server';
import { BudgetItem, Transaction, Goal } from '@wealth-management/types';

export async function POST(req: Request) {
  try {
    const body = await req.json() as {
      budget: BudgetItem[];
      transactions: Transaction[];
      goals: Goal[];
      date: string;
      modelId: string;
    };
    const { budget, transactions, goals, date, modelId } = body;

    // 2. Prepare 30-day projection context (last 100 txns)
    const recentTxns = transactions.slice(-100);

    const taskInstruction = buildBudgetAdvisorPrompt({
      date,
      budget,
      goals,
      recentTxns
    });

    const result = await AIOrchestrator.runJson<Record<string, unknown>>({
      modelId: modelId,
      systemPromptInstruction: taskInstruction,
      prompt: "Analyze the budget and transactions to generate the advisor JSON. Ensure the 30-day forecast is realistic based on recurring patterns.",
    });

    return NextResponse.json(result);
  } catch (error: unknown) {
    console.error('AI Budget Advisor Error:', error);
    const message = error instanceof Error ? error.message : 'Failed to generate advisory';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
