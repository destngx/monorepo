import { NextResponse } from 'next/server';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { generateText } from 'ai';

import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { AppError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

export async function POST(req: Request) {
  try {
    const { payee, categories } = (await req.json()) as { payee: string; categories: string[] };

    if (!payee || !categories || !Array.isArray(categories)) {
      return NextResponse.json({ error: 'Missing payee or categories' }, { status: 400 });
    }

    const model = getLanguageModel('github-gpt-4o');

    const taskTemplate = await loadTaskPrompt('suggest-category');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      payee,
      categories: categories.join(', '),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionTemplate = await loadActionPrompt('suggest-category');
    const actionPrompt = replacePlaceholders(actionTemplate, { payee });

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: actionPrompt,
    });

    // Clean up the response just in case the AI adds extra whitespace or quotes
    const suggestedCategory = text.trim().replace(/^["']|["']$/g, '');

    // Final check that the suggested category is actually in the original list
    const finalCategory = categories.includes(suggestedCategory)
      ? suggestedCategory
      : categories.find((c) => c.toLowerCase() === suggestedCategory.toLowerCase()) || categories[0];

    return NextResponse.json({ category: finalCategory });
  } catch (error) {
    if (isAppError(error)) {
      console.error('AI Category Suggestion Error:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('AI Category Suggestion Error:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
