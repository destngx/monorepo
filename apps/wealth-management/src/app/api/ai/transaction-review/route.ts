import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';
import { buildSystemPrompt } from "@wealth-management/ai/server";

export async function POST(req: Request) {
  try {
    const body = await req.json() as { transactions: unknown[] };
    const { transactions } = body;

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      You are performing a weekly review of the user's recent transactions (last 7 days).
      
      Review these transactions:
      ${JSON.stringify(transactions)}

      SPECIALIZED OBJECTIVES for TRANSACTIONS PAGE:
      You are a strict Behavioral Finance Coach reviewing a granular list of this week's spending.
      Do not summarize the totals; your job is to spot the micro-habits.
      
      CRITICAL CONTEXT: If you see Binance transfers (deposits or incoming transfers) occurring between the 1st and 6th of the month, classify and treat them as the user's regular wages/salary.
      
      You MUST output exactly and only these 3 sections in Markdown:
      
      **1. The Immediate Observation**
      (Call out the most glaring recurring or unusual spending habit you see right now from the list).
      
      **2. The Hidden Leak**
      (Identify a sneaky expense or lifestyle creep indicator—something small that adds up or seems misaligned).
      
      **3. The 48-Hour Action Plan**
      (Give exactly one aggressive, micro-behavioral nudge the user must do in the next 48 hours to correct the leak).
      
      Keep the tone professional, slightly commanding, and deeply analytical. Output only the 3 sections.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: "Review the last 7 days of transactions and provide insights.",
    });

    return NextResponse.json({ review: text });
  } catch (error: unknown) {
    console.error('AI Transaction Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate review' }, { status: 500 });
  }
}
