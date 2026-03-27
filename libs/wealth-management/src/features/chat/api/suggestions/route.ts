import { generateText, LanguageModel } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { AppError, ValidationError, isAppError, getErrorMessage } from '@wealth-management/utils/errors';
import { env } from '@wealth-management/config';

export const maxDuration = 60;

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as { modelId?: string; context?: unknown };
    const { modelId, context } = body;

    // Choose a smart default if no modelId provided
    let selectedModel: string | undefined = modelId;
    if (!selectedModel || selectedModel === 'gpt-4o-mini') {
      selectedModel = env.ai.githubToken ? 'github-gpt-4o' : 'gpt-4o-mini';
    }

    const model = getLanguageModel(selectedModel);

    const result = await generateText({
      model: model as LanguageModel,
      system: `You are a Wealth Management AI assistant.
      Your task is to generate 3-5 suggested questions a user might want to ask based on their current page context and any active insights.
      Focus on being proactive, analytical, and helpful.
      Questions should be concise and drive financial health improvements.
      
      CRITICAL: You must return ONLY a JSON object in this exact format:
      {
        "suggestions": [
          { "label": "Short Label", "prompt": "Full Question?" }
        ]
      }`,
      prompt: `
        Current Context: ${JSON.stringify(context, null, 2)}
        
        Generate contextually relevant questions. Remember to return ONLY valid JSON.
      `,
    });

    let jsonString = result.text.trim();
    if (jsonString.includes('```json')) {
      jsonString = jsonString.split('```json')[1].split('```')[0].trim();
    } else if (jsonString.includes('```')) {
      jsonString = jsonString.split('```')[1].split('```')[0].trim();
    }

    const parsed = JSON.parse(jsonString) as unknown;

    return new Response(JSON.stringify(parsed), {
      headers: { 'Content-Type': 'application/json' },
    });
  } catch (err: unknown) {
    if (isAppError(err)) {
      console.error('[Suggestions API Error]:', err.toResponse());
      return new Response(JSON.stringify({ error: err.userMessage }), {
        status: err.statusCode,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    const appError = new AppError(getErrorMessage(err));
    console.error('[Suggestions API Error]:', appError.toResponse());
    const fallback = {
      suggestions: [
        { label: 'Financial Review', prompt: 'Give me a full financial review for this month.' },
        { label: 'Net Worth', prompt: 'What is my current net worth breakdown?' },
        { label: 'Savings Tips', prompt: 'How can I save more money this month?' },
      ],
    };
    return new Response(JSON.stringify(fallback), {
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
