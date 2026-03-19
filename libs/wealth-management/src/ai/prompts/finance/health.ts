import { loadPrompt, replacePlaceholders } from '../loader';

export interface HealthData {
  netWorth: number;
  totalAssets: number;
  totalLiabilities: number;
  loans: any[];
  months: any[];
}

export function buildFinancialHealthPrompt(data: HealthData): string {
  const template = loadPrompt('finance', 'health');
  return replacePlaceholders(template, {
    netWorth: data.netWorth,
    totalAssets: data.totalAssets,
    totalLiabilities: data.totalLiabilities,
    loans: data.loans,
    months: data.months,
  });
}
