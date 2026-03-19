import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';

import { Account } from '@wealth-management/types';

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as {
      accounts: Account[];
      totalAssets: number;
      totalLiabilities: number;
      totalNetWorth: number;
    };
    const { accounts, totalAssets, totalLiabilities, totalNetWorth } = body;

    const model = getLanguageModel('github-gpt-4o');

    const taskTemplate = await loadTaskPrompt('account-review');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      totalAssets,
      totalLiabilities,
      totalNetWorth,
      accountsJson: JSON.stringify(
        accounts.map((a: Account) => ({
          name: a.name,
          balance: a.balance,
          type: a.type,
          note: a.note,
        })),
      ),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt = await loadActionPrompt('account-review');

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    return NextResponse.json({ review: text });
  } catch (error: unknown) {
    console.error('AI Account Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate account review' }, { status: 500 });
  }
}
