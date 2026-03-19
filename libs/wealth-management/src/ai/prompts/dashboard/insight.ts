import { loadPrompt, replacePlaceholders } from '../loader';

export interface ChartInsightData {
  chartType: string;
  chartDescription?: string;
  chartData: any;
  market?: string;
  timeframe?: string;
}

export async function buildChartInsightPrompt(data: ChartInsightData): Promise<string> {
  const template = await loadPrompt('dashboard', 'insight');
  return replacePlaceholders(template, {
    chartType: data.chartType,
    market: data.market || 'General',
    timeframe: data.timeframe || 'N/A',
    chartDescription: data.chartDescription || 'N/A',
    chartData: data.chartData,
  });
}
