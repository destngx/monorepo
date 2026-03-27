import { streamText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { financialTools } from '@wealth-management/ai/server';
import { buildSystemPrompt } from '@wealth-management/ai/server';
import { ChatError, NetworkError, ValidationError, AppError } from '@wealth-management/utils/errors';
import { env } from '@wealth-management/config';

// Allow responses up to 5 minutes for deep searches and reasoning
export const maxDuration = 300;

interface ChatBody {
  messages: any[];
  modelId?: string;
  context?: any;
}

export async function POST(req: Request) {
  try {
    const body = (await req.json()) as ChatBody;
    const { messages, modelId, context } = body;

    if (!messages || !Array.isArray(messages)) {
      throw new ValidationError('Missing or invalid messages array');
    }

    // Choose a smart default if no modelId provided
    let selectedModel = modelId;
    if (!selectedModel || selectedModel === 'gpt-4o-mini') {
      selectedModel = env.ai.githubToken ? 'github-gpt-4o' : 'gpt-4o-mini';
    }

    const model = getLanguageModel(selectedModel);

    let taskInstruction = `You are in CHAT MODE — the user's interactive financial copilot session. Use all available tools proactively. Prioritize webSearch for any real-time data needs.`;

    if (context) {
      taskInstruction += `\n\nPrevious Global Macro Scan Context (use this for follow-up questions):\n${JSON.stringify(context, null, 2)}`;
    }

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    if (!env.ai.openaiApiKey && !env.ai.googleGenAiApiKey && !env.ai.anthropicApiKey && !env.ai.githubToken) {
      throw new ChatError('No AI provider API keys configured');
    }

    // Map messages to CoreMessage format, preserving tool calls and results
    const sdkMessages = messages.map((m) => {
      const parts = m.parts as
        | Array<{ type: string; text?: string; toolCallId?: string; toolName?: string; args?: unknown }>
        | undefined;
      const toolInvocations = m.toolInvocations as
        | Array<{ toolCallId: string; toolName: string; args: unknown; state?: string; result?: unknown }>
        | undefined;

      if (m.role === 'assistant' && (toolInvocations || parts?.some((p) => p.type === 'tool-call'))) {
        return {
          role: 'assistant',
          content:
            m.content ||
            parts
              ?.filter((p) => p.type === 'text')
              .map((p) => p.text)
              .join('\n') ||
            '',
          toolCalls:
            toolInvocations?.map((ti) => ({
              type: 'function',
              id: ti.toolCallId,
              function: { name: ti.toolName, arguments: JSON.stringify(ti.args) },
            })) ||
            parts
              ?.filter((p) => p.type === 'tool-call')
              .map((p) => ({
                type: 'function',
                id: p.toolCallId,
                function: { name: p.toolName, arguments: JSON.stringify(p.args) },
              })),
        };
      }

      if (m.role === 'tool' || toolInvocations?.some((ti) => ti.state === 'result')) {
        return {
          role: 'tool',
          content:
            toolInvocations?.map((ti) => ({
              type: 'tool-result',
              toolCallId: ti.toolCallId,
              toolName: ti.toolName,
              result: ti.result,
            })) || [],
        };
      }

      return {
        role: m.role,
        content:
          m.content ||
          parts
            ?.filter((p) => p.type === 'text')
            .map((p) => p.text)
            .join('\n') ||
          '',
      };
    });

    const result = streamText({
      model: model as any,
      system: systemPrompt,
      messages: sdkMessages as any,
      tools: financialTools,
      maxSteps: 10,
    } as any);

    return result.toUIMessageStreamResponse();
  } catch (err: unknown) {
    if (err instanceof ValidationError) {
      console.error('[Chat API Error]', err.toResponse());
      return new Response(JSON.stringify({ error: err.userMessage }), {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    if (err instanceof ChatError) {
      console.error('[Chat API Error]', err.toResponse());
      return new Response(JSON.stringify({ error: err.userMessage }), {
        status: err.statusCode,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    if (err instanceof AppError) {
      console.error('[Chat API Error]', err.toResponse());
      return new Response(JSON.stringify({ error: err.userMessage }), {
        status: err.statusCode,
        headers: { 'Content-Type': 'application/json' },
      });
    }

    const error = err instanceof Error ? err : new Error('Internal Server Error');
    const appError = new ChatError(error.message, {
      originalError: error.message,
    });
    console.error('[Chat API Error]', appError.toResponse());
    return new Response(JSON.stringify({ error: appError.userMessage }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
