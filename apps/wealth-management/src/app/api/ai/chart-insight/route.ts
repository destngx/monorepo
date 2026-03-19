import { NextResponse } from 'next/server';
import { AIOrchestrator, buildChartInsightPrompt, type ChartInsightData } from '@wealth-management/ai/server';

export const maxDuration = 30;

export async function POST(req: Request) {
  try {
    const data = (await req.json()) as ChartInsightData;

    if (!data.chartType || !data.chartData) {
      return NextResponse.json({ error: 'Missing chartType or chartData' }, { status: 400 });
    }

    const taskInstruction = buildChartInsightPrompt(data);
    const insight = await AIOrchestrator.run({
      systemPromptInstruction: taskInstruction,
      prompt: `Analyze the ${data.chartType} data for ${data.market || 'the current view'} and provide your expert insight.`,
    });

    return NextResponse.json({ insight });
  } catch (error: unknown) {
    console.error('[Chart Insight API Error]:', error);
    const message = error instanceof Error ? error.message : 'Failed to generate chart insight';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
