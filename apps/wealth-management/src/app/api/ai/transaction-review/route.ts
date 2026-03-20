import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import {
  extractAndParseJSON,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
  type StructuredInsight,
} from '@wealth-management/ai/server';

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
    console.error('AI Transaction Review Error:', error);
    return NextResponse.json({ error: 'Failed to generate review' }, { status: 500 });
  }
}
