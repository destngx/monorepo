// The recommendations list (ROADMAP R7, WM-REC-1..). Items come from the AI
// reviews and the engine rules; the user sets the status.

export type RecommendationStatus = 'open' | 'snoozed' | 'done' | 'dismissed';
export const RECOMMENDATION_STATUSES: RecommendationStatus[] = ['open', 'snoozed', 'done', 'dismissed'];

export interface Recommendation {
  id: string;
  row: number;
  /** yyyy-mm-dd */
  created: string;
  /** A review kind (budget-review, ...) or an engine rule (rule:<id>). */
  source: string;
  title: string;
  action: string;
  /** Estimated VND per year; null when not estimated. */
  impactVnd: number | null;
  /** App route of the evidence; empty when the facts stand alone. */
  evidencePath: string;
  evidence: string;
  /** A snooze that ended reads as open. */
  status: RecommendationStatus;
  snoozedUntil?: string;
  statusChanged?: string;
  /** Written by a later review, or "resolved" when a rule's condition is gone. */
  outcome?: string;
  /** App route the item belongs to. */
  page: string;
  etag: string;
}

export interface RecommendationSummary {
  open: number;
  snoozed: number;
  done: number;
  dismissed: number;
  openImpactVnd: number;
  openWithoutImpact: number;
}

export interface RecommendationList {
  tabExists: boolean;
  items: Recommendation[];
  summary: RecommendationSummary;
  issues: { row: number; message: string }[];
}

export interface RecommendationFilter {
  status?: RecommendationStatus;
  page?: string;
  source?: string;
}

export interface RecommendationStatusInput {
  status: RecommendationStatus;
  snoozedUntil?: string;
  snoozeDays?: number;
}

/** What a review wrote to the list. */
export interface RecommendationRecord {
  created: Recommendation[];
  duplicates: number;
  checked: Recommendation[];
  ignoredOutcomes: number;
  /** Set when the list could not be written; the review still stands. */
  error?: string;
}
