import { streamText } from 'ai';
import { getLanguageModel } from '@wealth-management/ai/providers';
import { financialTools } from '@wealth-management/ai/server';
import { buildSystemPrompt } from '@wealth-management/ai/server';

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
      return new Response(JSON.stringify({ error: 'Missing or invalid messages array' }), { status: 400 });
    }

    // Choose a smart default if no modelId provided
    let selectedModel = modelId;
    if (!selectedModel || selectedModel === 'gpt-4o-mini') {
      selectedModel = process.env.GITHUB_TOKEN ? 'github-gpt-4o' : 'gpt-4o-mini';
    }

    const model = getLanguageModel(selectedModel);

    let taskInstruction = `You are in CHAT MODE — the user's interactive financial copilot session. Use all available tools proactively. Prioritize webSearch for any real-time data needs.`;

    if (context) {
      taskInstruction += `\n\nPrevious Global Macro Scan Context (use this for follow-up questions):\n${JSON.stringify(context, null, 2)}`;
    }

    const systemPrompt = await buildSystemPrompt(taskInstruction);

    if (
      !process.env.OPENAI_API_KEY &&
      !process.env.GOOGLE_GENERATIVE_AI_API_KEY &&
      !process.env.ANTHROPIC_API_KEY &&
      !process.env.GITHUB_TOKEN
    ) {
      throw new Error('No AI provider API keys configured.');
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
    const error = err as Error;
    console.error('[Chat API Error]:', error.message || error);
    return new Response(JSON.stringify({ error: error.message || 'Internal Server Error' }), {
      status: 500,
      headers: { 'Content-Type': 'application/json' },
    });
  }
}
