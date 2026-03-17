import { NextResponse } from 'next/server';
import { AIOrchestrator } from '@wealth-management/ai/core';
import { buildChartInsightPrompt } from '@wealth-management/ai/server';

export const maxDuration = 30;

export async function POST(req: Request) {
  try {
    const data = await req.json();

    if (!data.chartType || !data.chartData) {
      return NextResponse.json({ error: 'Missing chartType or chartData' }, { status: 400 });
    }

    const taskInstruction = buildChartInsightPrompt(data);
    const insight = await AIOrchestrator.run({
      systemPromptInstruction: taskInstruction,
      prompt: `Analyze the ${data.chartType} data for ${data.market || 'the current view'} and provide your expert insight.`,
    });

    return NextResponse.json({ insight });
  } catch (error: any) {
    console.error('[Chart Insight API Error]:', error.message || error);
    return NextResponse.json(
      { error: error.message || 'Failed to generate chart insight' },
      { status: 500 }
    );
  }
}
