
import { AIOrchestrator } from '@wealth-management/ai/core';
import { getMarketPulseData } from "@wealth-management/services/server";
import { 
  extractInvestmentData, 
  buildSearchQueries 
} from "@wealth-management/services/server";
import { executeSearch, type SearchResult } from "@wealth-management/services/server";
import {
  buildThinkTankPrompt,
  buildSynthesisPrompt,
  buildActionPrompt,
  buildFallbackThinkTankPrompt,
  buildFallbackSynthesisPrompt,
  buildFallbackActionPrompt,
  formatSearchContext
} from "@wealth-management/ai/server";

interface ActionCommands {
  executable_commands: unknown[];
}

export const maxDuration = 300;

export async function POST(req: Request) {
  const encoder = new TextEncoder();
  const body = await req.json() as { accounts: unknown[]; prices?: Record<string, number> };
  const { accounts, prices } = body;

  const stream = new ReadableStream({
    async start(controller) {
      const sendEvent = (data: { type: string; message?: unknown; context?: unknown; error?: string; details?: string }) => {
        controller.enqueue(encoder.encode(JSON.stringify(data) + '\n'));
      };

      try {

        if (!accounts || !Array.isArray(accounts) || accounts.length === 0) {
          sendEvent({ type: 'error', error: 'Invalid accounts data' });
          controller.close();
          return;
        }


        // 1. Extract Data Context

        // 1. Extract Data Context
        const data = await extractInvestmentData(accounts);
        data.prices = prices || {};

        // 2. Market Intelligence (Search + Quantitative)
        const queries = buildSearchQueries(data);
        const searchPromises = queries.map(q => executeSearch(q));
        
        const [searchResponses, marketPulse] = await Promise.all([
          Promise.all(searchPromises),
          getMarketPulseData('1d')
        ]);

        const combinedResults: SearchResult[] = [];
        searchResponses.forEach(res => {
          if (res.results) combinedResults.push(...res.results);
        });

        const uniqueResults = Array.from(new Map(combinedResults.map(item => [item.url, item])).values()).slice(0, 8);
        const searchContext = formatSearchContext(uniqueResults);

        // --- Phase 1: Think Tank Expert Debate ---
        const thinkTankInstruction = buildThinkTankPrompt(data, searchContext, marketPulse);
        
        let thinkTankText = '';
        try {
          thinkTankText = await AIOrchestrator.run({
            systemPromptInstruction: thinkTankInstruction,
            prompt: 'Analyze the market intelligence provided in the system context. Generate the Phase 1 and Phase 2 Think Tank Expert Debate.',
          });
        } catch {
          thinkTankText = await AIOrchestrator.run({
            systemPromptInstruction: buildFallbackThinkTankPrompt(data),
            prompt: 'Generate the Phase 1 and 2 Think Tank analysis now.',
          });
        }

        sendEvent({
          type: 'message',
          message: {
            id: 'think-tank',
            role: 'assistant',
            name: 'Think Tank Council',
            content: thinkTankText
          }
        });

        // --- Phase 2: Synthesis ---
        const synthesisInstruction = buildSynthesisPrompt(data, thinkTankText);
        
        let synthesisText = '';
        try {
          synthesisText = await AIOrchestrator.run({
            systemPromptInstruction: synthesisInstruction,
            prompt: 'Review the expert debate and synthesize the findings. Generate the Phase 3 Chairman Synthesis.',
          });
        } catch {
          synthesisText = await AIOrchestrator.run({
            systemPromptInstruction: buildFallbackSynthesisPrompt(data, thinkTankText),
            prompt: 'Generate the Chairman Synthesis now.',
          });
        }

        sendEvent({
          type: 'message',
          message: {
            id: 'synthesis',
            role: 'assistant',
            name: 'Lộc Phát Tài (Analysis)',
            content: synthesisText
          }
        });

        // --- Phase 3: Action Execution ---
        const actionInstruction = buildActionPrompt(data, synthesisText);
        
        let parsedActions: ActionCommands = { executable_commands: [] };
        try {
          parsedActions = await AIOrchestrator.runJson<ActionCommands>({
            systemPromptInstruction: actionInstruction,
            prompt: 'Based on the synthesis, generate the 10 actionable commands in the requested JSON format.',
          });
        } catch {
          try {
            parsedActions = await AIOrchestrator.runJson<ActionCommands>({
              systemPromptInstruction: buildFallbackActionPrompt(synthesisText),
              prompt: 'Generate the Action commands now.',
            });
          } catch {
             // Action fallback failed.
          }
        }

        sendEvent({
          type: 'message',
          message: {
            id: 'actions',
            role: 'assistant',
            name: 'Lộc Phát Tài (Actions)',
            content: JSON.stringify(parsedActions)
          }
        });

        // Final context closure for the UI
        sendEvent({
          type: 'context',
          context: {
            ...data,
            searchContext,
            marketPulse
          }
        });

        controller.close();
      } catch (error: unknown) {
        const message = error instanceof Error ? error.message : 'Unknown error';
        sendEvent({ type: 'error', error: 'Internal system error during analysis', details: message });
        controller.close();
      }
    }
  });

  return new Response(stream, {
    headers: {
      'Content-Type': 'application/x-ndjson',
      'Cache-Control': 'no-cache',
      'Connection': 'keep-alive',
    },
  });
}
