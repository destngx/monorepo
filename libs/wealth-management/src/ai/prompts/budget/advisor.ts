import { loadPrompt, replacePlaceholders } from '../loader';

export interface BudgetAdvisorData {
  date: string;
  budget: any;
  goals: any;
  recentTxns: any[];
}

export function buildBudgetAdvisorPrompt(data: BudgetAdvisorData): string {
  const template = loadPrompt('budget', 'advisor');
  return replacePlaceholders(template, {
    date: data.date,
    budget: data.budget,
    goals: data.goals,
    recentTxns: data.recentTxns.slice(-50),
  });
}
