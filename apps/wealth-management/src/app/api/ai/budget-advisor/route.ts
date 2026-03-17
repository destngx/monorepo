import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildBudgetAdvisorPrompt } from '@wealth-management/ai/server';

export async function POST(req: Request) {
  try {
    const { budget, transactions, goals, date, modelId } = await req.json();

    // 2. Prepare 30-day projection context (last 100 txns)
    const recentTxns = transactions.slice(-100);

    const taskInstruction = buildBudgetAdvisorPrompt({
      date,
      budget,
      goals,
      recentTxns
    });

    const result = await AIOrchestrator.runJson<any>({
      modelId: modelId,
      systemPromptInstruction: taskInstruction,
      prompt: "Analyze the budget and transactions to generate the advisor JSON. Ensure the 30-day forecast is realistic based on recurring patterns.",
    });

    return NextResponse.json(result);
  } catch (error: any) {
    console.error('AI Budget Advisor Error:', error);
    return NextResponse.json({ error: error.message || 'Failed to generate advisory' }, { status: 500 });
  }
}
