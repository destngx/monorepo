import { streamText } from "ai";
import { getLanguageModel } from "@wealth-management/ai/providers";
import { financialTools } from "@wealth-management/ai/server";
import { buildSystemPrompt } from "@wealth-management/ai/server";

// Allow responses up to 5 minutes for deep searches and reasoning
export const maxDuration = 300;

export async function POST(req: Request) {
  try {
    const body = await req.json();
    const { messages, modelId, context } = body;

    if (!messages || !Array.isArray(messages)) {
      return new Response(
        JSON.stringify({ error: "Missing or invalid messages array" }),
        { status: 400 }
      );
    }

    // Choose a smart default if no modelId provided
    let selectedModel = modelId;
    if (!selectedModel || selectedModel === "gpt-4o-mini") {
      selectedModel = process.env.GITHUB_TOKEN
        ? "github-gpt-4o"
        : "gpt-4o-mini";
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
      throw new Error("No AI provider API keys configured.");
    }

    // Map messages to CoreMessage format, preserving tool calls and results
    const sdkMessages = messages.map((m: any) => {
      if (m.role === 'assistant' && (m.toolInvocations || m.parts?.some((p: any) => p.type === 'tool-call'))) {
        return {
          role: 'assistant',
          content: m.content || m.parts?.filter((p: any) => p.type === 'text').map((p: any) => p.text).join('\n') || '',
          toolCalls: m.toolInvocations?.map((ti: any) => ({
            type: 'function',
            id: ti.toolCallId,
            function: { name: ti.toolName, arguments: JSON.stringify(ti.args) }
          })) || m.parts?.filter((p: any) => p.type === 'tool-call').map((p: any) => ({
            type: 'function',
            id: p.toolCallId,
            function: { name: p.toolName, arguments: JSON.stringify(p.args) }
          }))
        };
      }
      
      if (m.role === 'tool' || m.toolInvocations?.some((ti: any) => ti.state === 'result')) {
        return {
          role: 'tool',
          content: m.toolInvocations?.map((ti: any) => ({
            type: 'tool-result',
            toolCallId: ti.toolCallId,
            toolName: ti.toolName,
            result: ti.result
          })) || []
        };
      }

      return {
        role: m.role,
        content: m.content || m.parts?.filter((p: any) => p.type === 'text').map((p: any) => p.text).join('\n') || ''
      };
    });

    const result = await streamText({
      model: model as any,
      system: systemPrompt,
      messages: sdkMessages,
      tools: financialTools,
      maxSteps: 10,
    } as any);

    return result.toUIMessageStreamResponse();
  } catch (err: any) {
    console.error("[Chat API Error]:", err.message || err);
    return new Response(
      JSON.stringify({ error: err.message || "Internal Server Error" }),
      {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }
    );
  }
}
