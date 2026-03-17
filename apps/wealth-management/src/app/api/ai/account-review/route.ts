import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';
import { buildSystemPrompt } from "@wealth-management/ai/server";

import { Account } from '@wealth-management/types';

export async function POST(req: Request) {
  try {
    const body = await req.json() as { 
      accounts: Account[]; 
      totalAssets: number; 
      totalLiabilities: number; 
      totalNetWorth: number; 
    };
    const { accounts, totalAssets, totalLiabilities, totalNetWorth } = body;

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      Your task is to provide a senior financial advisor's specialized review of the user's account structure. 
      You are not here to give generic advice; you are here to dissect their asset allocation and idle cash.
      
      Summary Stats:
      - Total Assets: ${totalAssets} VND
      - Total Liabilities: ${totalLiabilities} VND
      - Net Worth: ${totalNetWorth} VND
      
      Account Details:
      ${JSON.stringify(accounts.map((a: Account) => ({
      name: a.name,
      balance: a.balance,
      type: a.type,
      note: a.note
    })))}

      SPECIALIZED OBJECTIVES for ACCOUNTS PAGE:
      1. Asset Allocation & Currency Attack: Analyze the ratio of VND to Foreign/Crypto assets. Are they overexposed to risk or taking on inflation risk by holding too much VND?
      2. Idle Cash Leakage: Identify specifically which checking/current accounts hold excessive balances that aren't generating yield, and strongly advise moving exact amounts to high-yield or investment accounts (e.g., Binance Earn, TCB Term Deposit).
      3. Emergency Fund Check: Briefly confirm if their immediately liquid cash covers an assumed 3-6 months.
      4. DO NOT mention credit card optimization or budgeting here; keep strictly to account architecture and asset placement.
      
      Format with clear, actionable bullet points. Use professional, direct Vietnamese financial tone.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: "Analyze the user's account structure and provide a wealth health review.",
    });

    return NextResponse.json({ review: text });
  } catch (error: unknown) {
    console.error('AI Account Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate account review' }, { status: 500 });
  }
}
