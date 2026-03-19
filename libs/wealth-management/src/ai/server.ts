// Server-side entry point for AI components
export * from './index'; // Include client-safe exports
export { buildBudgetAdvisorPrompt } from './prompts/budget/advisor';
export { buildChartInsightPrompt, type ChartInsightData } from './prompts/dashboard/insight';
export { buildFinancialHealthPrompt } from './prompts/finance/health';
export {
  buildThinkTankPrompt,
  buildSynthesisPrompt,
  buildActionPrompt,
  buildFallbackThinkTankPrompt,
  buildFallbackSynthesisPrompt,
  buildFallbackActionPrompt,
  getIdentityPrompt,
  formatSearchContext,
} from './prompts';
export { buildSystemPrompt } from './system-prompt';
export { AIOrchestrator } from './core';
export { getLanguageModel } from './providers';
export { financialTools } from './tools';
export { loadTaskPrompt, loadPrompt, replacePlaceholders } from './prompts/loader';
export { prefetchAllContent } from '../services/sheets/content';
export { loadKnowledge } from './knowledge/loader';
