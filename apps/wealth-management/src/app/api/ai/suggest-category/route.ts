import { NextResponse } from 'next/server';
import { getLanguageModel } from "@wealth-management/ai/providers";
import { generateText } from 'ai';

import { buildSystemPrompt } from "@wealth-management/ai/server";

export async function POST(req: Request) {
  try {
    const { payee, categories } = await req.json() as { payee: string; categories: string[] };

    if (!payee || !categories || !Array.isArray(categories)) {
      return NextResponse.json({ error: 'Missing payee or categories' }, { status: 400 });
    }

    const model = getLanguageModel('github-gpt-4o');

    const taskInstruction = `
      Your task is to accurately suggest the best category for a transaction based on the payee name.
      Payee: "${payee}"
      
      Pick the most suitable category for this transaction from the following list:
      ${categories.join(', ')}
      
      RULES:
      - Return ONLY the exact category name from the list.
      - Do not include any explanation or punctuation.
      - If no category seems a perfect fit, pick the closest one.
      - If it is completely ambiguous, return the first item in the list.
    `;

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    const { text } = await generateText({
      model,
      system: systemPrompt,
      prompt: `Suggest a category for payee: "${payee}"`,
    });

    // Clean up the response just in case the AI adds extra whitespace or quotes
    const suggestedCategory = text.trim().replace(/^["']|["']$/g, '');

    // Final check that the suggested category is actually in the original list
    const finalCategory = categories.includes(suggestedCategory)
      ? suggestedCategory
      : categories.find(c => c.toLowerCase() === suggestedCategory.toLowerCase()) || categories[0];

    return NextResponse.json({ category: finalCategory });
  } catch (error) {
    console.error('AI Category Suggestion Error:', error);
    return NextResponse.json({ error: 'Failed to suggest category' }, { status: 500 });
  }
}
