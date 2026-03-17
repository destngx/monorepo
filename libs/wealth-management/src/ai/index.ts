// Client-safe re-exports for AI logic
export { AI_MODELS, type AIModelId } from '@wealth-management/types/chat';
export { buildBudgetAdvisorPrompt } from './prompts/budget/advisor';
export { buildChartInsightPrompt } from './prompts/dashboard/insight';
export { buildFinancialHealthPrompt } from './prompts/finance/health';
export {
  buildThinkTankPrompt,
  buildSynthesisPrompt,
  buildActionPrompt,
  buildFallbackThinkTankPrompt,
  buildFallbackSynthesisPrompt,
  buildFallbackActionPrompt,
  LOC_PHAT_TAI_IDENTITY,
  formatSearchContext,
} from './prompts';

// DO NOT export buildSystemPrompt or server-side classes here
// as they might depend on server-only services.
