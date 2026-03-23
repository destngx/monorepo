import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import {
  extractAndParseJSON,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
  type StructuredInsight,
} from '@wealth-management/ai/server';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';
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
    const actionPrompt = (await loadActionPrompt('account-review')) + '\n\n' + STRUCTURED_INSIGHT_FORMAT_INSTRUCTION;

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
      console.error('AI Account Review Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Account Review Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
