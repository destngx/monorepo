import type { Allocation } from './investments';

export interface HealthComponent {
  key: string;
  label: string;
  value: number;
  target: number;
  unit: string;
  score: number;
  maxScore: number;
  formula: string;
  detail: string;
}

export interface FinancialHealth {
  score: number;
  band: 'red' | 'yellow' | 'green';
  components: HealthComponent[];
  recommendations: { component: string; title: string; action: string }[];
  liquidAssets: number;
  avgMonthlyIncome: number;
  avgMonthlyExpense: number;
  netWorth: number;
  freedomMonths: number | null;
  /** The one engine allocation; the diversification pillar scores its target classes. */
  allocation: Allocation;
  lookbackMonths: number;
  insufficientIncome: boolean;
  /** Net worth projected with WM-RULE-12; absent from engines before WM-HEA-2. */
  projection?: NetWorthProjection[];
}

export interface NetWorthProjection {
  years: number;
  nominal: number;
  /** Inflation-adjusted, in today's VND. */
  real: number;
}

export type DependencyStatus = 'ok' | 'degraded' | 'down' | 'unconfigured';

export interface EngineHealth {
  status: DependencyStatus;
  checks: { name: string; status: DependencyStatus; error?: string; latencyMs?: number }[];
}

/** The connected spreadsheet (WM-SET-2). */
export interface SheetInfo {
  spreadsheetId: string;
  title: string;
  /** Engine-side time of the last successful sync; null until one after a restart. */
  lastSyncAt: string | null;
}

export interface SyncResult {
  success: boolean;
  message: string;
  deleted?: number;
}
