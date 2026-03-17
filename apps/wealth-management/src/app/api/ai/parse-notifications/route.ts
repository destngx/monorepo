import { NextResponse } from 'next/server';
import { generateText } from 'ai';
import { openai } from '@ai-sdk/openai';
import { buildSystemPrompt } from "@wealth-management/ai/server";
import { getBudget } from "@wealth-management/services/server";
import { getAccounts } from "@wealth-management/services/server";
import { handleApiError } from "@wealth-management/utils/server";

interface ParseNotificationInput {
  id: string;
  content: string;
}

export async function POST(req: Request) {
  try {
    const { notifications } = await req.json() as { notifications: ParseNotificationInput[] };
    if (!notifications || !Array.isArray(notifications)) {
      return NextResponse.json({ error: 'Notifications array is required' }, { status: 400 });
    }

    const budget = (await getBudget()) as { category: string }[];
    const accounts = (await getAccounts()) as { name: string }[];

    const categories = budget.map(b => b.category).join(', ');
    const accountNames = accounts.map(a => a.name).join(', ');

    const taskInstruction = `
      Your task is to accurately parse bank notifications into the application's schema.
      
      Available Accounts: ${accountNames}
      Available Categories: ${categories}
      
      For each notification, identify:
      - date: ISO date string. 
        IMPORTANT: Bank notification content usually uses DD/MM/YYYY or DD/MM format (e.g., '05/03' is March 5th). 
        You MUST correctly interpret the day and month and return a standard ISO date string.
      - payee: The merchant or person
      - amount: Number (positive)
      - type: 'payment' or 'deposit'
      - accountName: Must be one of the available accounts
      - category: Must be one of the available categories.
        IMPORTANT: If the message content indicates a transfer from "NGUYEN PHAM QUANG DINH" to "NGUYEN PHAM QUANG DINH", set the category to "[Transfer]".
      - categoryType: The type of the chosen category ('income', 'expense', or 'non-budget')
      - memo: Brief description
      - notificationId: The exact ID provided in the input
      
      CRITICAL CONTEXT: If you parse a notification for a Binance account (or involving Binance) occurring between the 1st and 6th of the month, classify and treat it as the user's regular wages/salary (e.g. category 'Salary' or similar income category).
      
      Notifications:
      ${notifications.map((n: ParseNotificationInput) => `ID: ${n.id}\nContent: ${n.content}`).join('\n---\n')}
      
      CRITICAL: Return ONLY a valid JSON array. Do not include any markdown formatting, backticks, or explanatory text.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model: openai('gpt-4o'),
      system: systemPrompt,
      prompt: "Parse the notifications provided in the task instructions.",
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
