import { loadPrompt, replacePlaceholders } from '../loader';
import { type SearchResult } from '../../../services/search-service';
import { type MarketPulseResponse } from '../../../services/market-data-service';

export function formatSearchContext(searchResults: SearchResult[]): string {
  if (searchResults.length === 0) {
    return '[INTELLIGENCE BLACKOUT: All search sources temporarily unavailable. Proceed with deep internal knowledge of macro trends.]';
  }

  return searchResults
    .map((r, i) => `[${i + 1}] "${r.title}"\n    ${r.description}\n    Source: ${r.url}`)
    .join('\n\n');
}

export interface DataContext {
  cryptoHoldings: any[];
  ifcHoldings: any[];
  totalVND: number;
  totalCrypto: string;
  accountsSummary: any[];
  prices: Record<string, number>;
  loans: any[];
  recentTransactions: any[];
  budget: { income: number; expenses: number };
  p2pRate: number;
}

function formatCorrelationMatrix(assets: string[], matrix: number[][]): string {
  if (!assets.length || !matrix.length) return 'N/A';

  let output = 'ASSET CORRELATION HEATMATRIX (Pairwise Coefficients):\n';
  output += 'Columns: ' + assets.join(' | ') + '\n';

  for (let i = 0; i < assets.length; i++) {
    const row = matrix[i].map((val) => val.toFixed(2)).join(' | ');
    output += `${assets[i].padEnd(10)}: ${row}\n`;
  }
  return output;
}

export async function buildThinkTankPrompt(
  data: DataContext,
  searchContext: string,
  marketPulse: MarketPulseResponse,
  newsAnalysis?: Record<string, any>,
): Promise<string> {
  const template = await loadPrompt('market', 'think-tank');
  return replacePlaceholders(template, {
    totalVND: data.totalVND,
    totalCrypto: data.totalCrypto || 'None',
    income: data.budget.income,
    expenses: data.budget.expenses,
    p2pRate: data.p2pRate,
    accountsSummary: JSON.stringify(data.accountsSummary),
    loans: JSON.stringify(data.loans),
    recentTransactions: JSON.stringify(data.recentTransactions),
    cryptoHoldings: JSON.stringify(data.cryptoHoldings),
    ifcHoldings: JSON.stringify(data.ifcHoldings),
    prices: JSON.stringify(data.prices),
    newsAnalysis: newsAnalysis ? JSON.stringify(newsAnalysis, null, 2) : 'N/A: No pre-analyzed news available.',
    searchContext,
    usRegimeName: marketPulse.us.scenarios?.[0]?.name,
    usRegimeType: marketPulse.us.scenarios?.[0]?.regime,
    usRegimeConfidence: marketPulse.us.scenarios?.[0]?.confidence,
    usAssets: marketPulse.us.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}%)`).join(', '),
    usDrivers: marketPulse.us.drivers?.summaryEn || 'N/A',
    usCapitalFlowSignal: marketPulse.us.capitalFlow.signal,
    usCapitalFlowSummary: marketPulse.us.capitalFlow.summaryEn,
    usCorrelationMatrix: formatCorrelationMatrix(marketPulse.us.assetList, marketPulse.us.correlationMatrix),
    usScenarioSummary: marketPulse.us.scenarios?.[0]?.summaryEn,
    vnRegimeName: marketPulse.vn.scenarios?.[0]?.name,
    vnRegimeType: marketPulse.vn.scenarios?.[0]?.regime,
    vnRegimeConfidence: marketPulse.vn.scenarios?.[0]?.confidence,
    vnAssets: marketPulse.vn.assets.map((a) => `${a.name}: ${a.price} (${a.percentChange.toFixed(2)}%)`).join(', '),
    vnDrivers: marketPulse.vn.drivers?.summaryEn || 'N/A',
    vnCapitalFlowSignal: marketPulse.vn.capitalFlow.signal,
    vnCapitalFlowSummary: marketPulse.vn.capitalFlow.summaryEn,
    vnCorrelationMatrix: formatCorrelationMatrix(marketPulse.vn.assetList, marketPulse.vn.correlationMatrix),
    vnScenarioSummary: marketPulse.vn.scenarios?.[0]?.summaryVi,
  });
}

export async function buildSynthesisPrompt(data: DataContext, expertDebateContext: string): Promise<string> {
  const template = await loadPrompt('market', 'synthesis');
  return replacePlaceholders(template, {
    totalVND: data.totalVND,
    totalCrypto: data.totalCrypto || 'None',
    income: data.budget.income,
    expenses: data.budget.expenses,
    p2pRate: data.p2pRate,
    accountsSummary: JSON.stringify(data.accountsSummary),
    loans: JSON.stringify(data.loans),
    recentTransactions: JSON.stringify(data.recentTransactions),
    expertDebateContext,
  });
}

export async function buildActionPrompt(data: DataContext, synthesisContext: string): Promise<string> {
  const template = await loadPrompt('market', 'action');
  return replacePlaceholders(template, {
    accountsSummary: JSON.stringify(data.accountsSummary),
    cryptoHoldings: JSON.stringify(data.cryptoHoldings),
    ifcHoldings: JSON.stringify(data.ifcHoldings),
    synthesisContext,
  });
}

export async function buildFallbackThinkTankPrompt(data: DataContext): Promise<string> {
  const template = await loadPrompt('market', 'fallback-think-tank');
  return replacePlaceholders(template, {
    totalVND: data.totalVND,
    totalCrypto: data.totalCrypto || 'None',
  });
}

export async function buildFallbackSynthesisPrompt(data: DataContext, debate: string): Promise<string> {
  const template = await loadPrompt('market', 'fallback-synthesis');
  return replacePlaceholders(template, {
    accountsSummary: JSON.stringify(data.accountsSummary),
    debate,
  });
}

export async function buildFallbackActionPrompt(synthesis: string): Promise<string> {
  const template = await loadPrompt('market', 'fallback-action');
  return replacePlaceholders(template, {
    synthesis,
  });
}
