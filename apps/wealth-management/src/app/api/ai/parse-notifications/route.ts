import { NextResponse } from 'next/server';
import { generateText } from 'ai';
import { openai } from '@ai-sdk/openai';
import { buildSystemPrompt, loadTaskPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';
import { getBudget } from '@wealth-management/services/server';
import { getAccounts } from '@wealth-management/services/server';
import { handleApiError } from '@wealth-management/utils/server';

interface ParseNotificationInput {
  id: string;
  content: string;
}

export async function POST(req: Request) {
  try {
    const { notifications } = (await req.json()) as { notifications: ParseNotificationInput[] };
    if (!notifications || !Array.isArray(notifications)) {
      return NextResponse.json({ error: 'Notifications array is required' }, { status: 400 });
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
      model: openai('gpt-4o'),
      system: systemPrompt,
      prompt: actionPrompt,
    });

    const cleanText = text.trim();
    // More robust JSON extraction: find the first [ and last ]
    const startBracket = cleanText.indexOf('[');
    const endBracket = cleanText.lastIndexOf(']');

    if (startBracket === -1 || endBracket === -1) {
      console.error('AI non-json output:', cleanText);
      throw new Error('AI failed to return a valid list. Please try again.');
    }

    const jsonString = cleanText.substring(startBracket, endBracket + 1);

    try {
      const parsed = JSON.parse(jsonString) as unknown;
      return NextResponse.json(parsed);
    } catch {
      console.error('JSON Parse Error. AI Output:', text);
      throw new Error('AI returned malformed JSON. Please try again.');
    }
  } catch (error: unknown) {
    return handleApiError(error, 'AI Parse Notifications');
  }
}
