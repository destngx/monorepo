import { NextResponse } from 'next/server';
import { generateText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { getBudget } from '@wealth-management/services/server';
import { getAccounts } from '@wealth-management/services/server';
import { AppError, ValidationError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';

interface ParseNotificationInput {
  id: string;
  content: string;
}

export async function POST(req: Request) {
  try {
    const { notifications } = (await req.json()) as { notifications: ParseNotificationInput[] };
    if (!notifications || !Array.isArray(notifications)) {
      throw new ValidationError('Notifications array is required');
    }

    const budget = (await getBudget()) as { category: string }[];
    const accounts = (await getAccounts()) as { name: string }[];

    const categories = budget.map((b) => b.category).join(', ');
    const accountNames = accounts.map((a) => a.name).join(', ');

    const taskTemplate = await loadTaskPrompt('parse-notifications');
    const taskInstruction = replacePlaceholders(taskTemplate, {
      accountNames,
      categories,
      notificationsList: notifications
        .map((n: ParseNotificationInput) => `ID: ${n.id}\nContent: ${n.content}`)
        .join('\n---\n'),
    });

    const systemPrompt = await buildSystemPrompt(taskInstruction);
    const actionPrompt = await loadActionPrompt('parse-notifications');

    const { text } = await generateText({
      model: getLanguageModel('github-gpt-4o'),
      system: systemPrompt,
      prompt: actionPrompt,
    });

    const cleanText = text.trim();
    console.log('AI Response:', cleanText.substring(0, 500)); // Log first 500 chars
    const startBracket = cleanText.indexOf('[');
    const endBracket = cleanText.lastIndexOf(']');

    if (startBracket === -1 || endBracket === -1) {
      console.error('AI non-json output:', cleanText);
      throw new AppError('AI failed to return a valid list. Please try again.');
    }

    const jsonString = cleanText.substring(startBracket, endBracket + 1);

    try {
      const parsed = JSON.parse(jsonString) as unknown;
      return NextResponse.json(parsed);
    } catch {
      console.error('JSON Parse Error. AI Output:', text);
      throw new AppError('AI returned malformed JSON. Please try again.');
    }
  } catch (error: unknown) {
    if (isAppError(error)) {
      console.error('[API Error] AI Parse Notifications:', error.toResponse());
      return NextResponse.json({ error: error.userMessage }, { status: error.statusCode });
    }
    const appError = new AppError(getErrorMessage(error));
    console.error('[API Error] AI Parse Notifications:', appError.toResponse());
    return NextResponse.json({ error: appError.userMessage }, { status: appError.statusCode });
  }
}
