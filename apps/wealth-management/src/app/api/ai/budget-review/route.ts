import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';

import { buildSystemPrompt } from "@wealth-management/ai/server";

export async function POST(req: Request) {
  try {
    const { budget, transactions, totalSpent, totalLimit, view, date } = await req.json();

    const recentTransactions = transactions
      .slice(-30)
      .map((t: any) => ({
        category: t.category,
        amount: t.payment || 0,
        deposit: t.deposit || 0,
        date: t.date,
        payee: t.payee
      }));

    const activeBudgets = budget
      .filter((b: any) => b.monthlySpent > 0 || (b.monthlyLimit && b.monthlyLimit > 0))
      .map((b: any) => ({
        category: b.category,
        spent: b.monthlySpent,
        limit: b.monthlyLimit,
        remaining: b.monthlyRemaining
      }));

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      You are performing a periodic financial review of the user's budget and spending patterns.
      Period: ${view} (${date})

      Summary Stats:
      - Total Spent: ${totalSpent} VND
      - Total Budget Limit: ${totalLimit} VND
      - Usage Percentage: ${totalLimit > 0 ? Math.round((totalSpent / totalLimit) * 100) : 0}%
      
      Budget Details:
      ${JSON.stringify(activeBudgets)}
      
      Recent Transactions:
      ${JSON.stringify(recentTransactions)}

      SPECIALIZED OBJECTIVES for BUDGET PAGE:
      You are a strict, forward-looking Budget Pacer & Forecaster. Do not just summarize what happened; predict where they are going and demand course correction.
      
      CRITICAL CONTEXT: If you see Binance transfers (deposits or incoming transfers) occurring between the 1st and 6th of the month within the recent transactions, classify and treat them as the user's regular wages/salary.
      
      1. Pace Analysis: Based on the current date (${date}) and the period (${view}), tell them if they are burning cash too fast or too slow relative to the elapsed time.
      2. The Projection: State clearly what your forecasted end-of-period outcome is for their most endangered category.
      3. Reallocation Protocol: Demand exactly one category-to-category fund transfer (e.g., "Move 500k from Entertainment to cover the Grocery overage"). If they are doing perfectly, praise them and suggest sweeping excess to investments.
      
      Format concisely using markdown. Be commanding, analytical, and highly structured format.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: `Review the budget for ${view} (${date}).`,
    });

    return NextResponse.json({ review: text });
  } catch (error) {
    console.error('AI Budget Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate review' }, { status: 500 });
  }
}
