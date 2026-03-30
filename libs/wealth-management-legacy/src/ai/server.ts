// Server-side entry point for AI components
export * from './index'; // Include client-safe exports
export {
  type StructuredInsight,
  type InsightSection,
  type InsightSeverity,
  type InsightIconHint,
  STRUCTURED_INSIGHT_FORMAT_INSTRUCTION,
} from './core/types';
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
export { AIOrchestrator } from './core/orchestrator';
export { extractAndParseJSON } from './core/parser';
export { getLanguageModel } from './providers';
export { financialTools } from './tools';
export { loadTaskPrompt, loadPrompt, loadActionPrompt, replacePlaceholders } from './prompts/loader';
export { prefetchAllContent } from '../services/sheets/content';
export { loadKnowledge } from './knowledge/loader';
