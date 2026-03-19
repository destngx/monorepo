import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildChartInsightPrompt, loadActionPrompt, replacePlaceholders } from '@wealth-management/ai/server';

export const maxDuration = 30;

export async function POST(req: Request) {
  try {
    const data = await req.json();

    const taskInstruction = await buildChartInsightPrompt(data);
    const actionTemplate = await loadActionPrompt('chart-insight');
    const actionPrompt = replacePlaceholders(actionTemplate, {
      chartType: data.chartType,
      market: data.market || 'the current view',
    });

    const insight = await AIOrchestrator.run({
      systemPromptInstruction: taskInstruction,
      prompt: actionPrompt,
    });

    return NextResponse.json({ insight });
  } catch (error: unknown) {
    console.error('[Chart Insight API Error]:', error);
    const message = error instanceof Error ? error.message : 'Failed to generate chart insight';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
