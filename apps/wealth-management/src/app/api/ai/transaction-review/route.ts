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

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as { transactions: unknown[] };
    const { transactions } = body;

    const model = getLanguageModel('github-gpt-4o');

    const taskTemplate = await loadTaskPrompt('transaction-review');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      transactionsJson: JSON.stringify(transactions),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt =
      (await loadActionPrompt('transaction-review')) + '\n\n' + STRUCTURED_INSIGHT_FORMAT_INSTRUCTION;

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
      console.error('AI Transaction Review Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Transaction Review Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
