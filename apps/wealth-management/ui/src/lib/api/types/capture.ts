import type { TransactionInput } from './transactions';

// Smart Capture (§6.2). The model only reads materials; matching, transfers,
// dedup and balances are engine code.

export interface CaptureLimits {
  maxItems: number;
  maxItemBytes: number;
  maxBatchBytes: number;
}

export const CAPTURE_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'] as const;

export interface CaptureMaterialPayload {
  id: string;
  kind: 'image' | 'text';
  name?: string;
  mediaType?: string;
  /** Base64 image bytes. */
  data?: string;
  text?: string;
}

export interface CaptureEvidence {
  materialId: string;
  quote?: string;
}

export interface CaptureDuplicate {
  transactionId: string;
  date: string;
  payee: string;
  amount: number;
  reference?: string;
}

export type CaptureField = 'amount' | 'date' | 'account' | 'payee' | 'category';

/** Where a row's category came from; "manual" is picked by the user on a draft. */
export type CaptureCategorySource = 'rule' | 'history' | 'ai' | 'transfer' | 'default' | 'manual';

export interface CaptureSuggestion {
  id: string;
  selected: boolean;
  transaction: TransactionInput;
  direction: 'in' | 'out';
  amount: number;
  currency: string;
  occurredAt: string;
  confidence: number;
  lowConfidence: CaptureField[];
  accountHint?: string;
  accountSource?: 'alias' | 'digits' | 'name';
  accountCandidates?: string[];
  categorySource?: CaptureCategorySource;
  tagOptions?: string[];
  sourceApp?: string;
  evidence: CaptureEvidence[];
  duplicate?: CaptureDuplicate;
  /**
   * The account's last recorded date in the sheet (dd/mm/yyyy) when this row is
   * dated on or before it without an exact match: possibly recorded already (WM-CAP-21).
   */
  recordedThrough?: string;
  transferGroup?: string;
  notes?: string[];
}

export interface CaptureBalance {
  account: string;
  currency: string;
  balanceAfter: number;
  asOf: string;
  materialId: string;
  sheetBalance: number;
}

export interface CaptureResult {
  suggestions: CaptureSuggestion[];
  unreadable: { materialId: string; reason: string }[];
  balances: CaptureBalance[];
}

export interface CaptureProgress {
  stage: 'extract' | 'match' | 'categorize';
  materialId?: string;
  status?: 'running' | 'ok' | 'unreadable' | 'error';
  events?: number;
  message?: string;
}

export type CaptureEvent =
  | ({ type: 'progress' } & CaptureProgress)
  | { type: 'result'; result: CaptureResult }
  | { type: 'error'; message: string };

export interface CaptureRule {
  kind: 'alias' | 'payee';
  pattern: string;
  value: string;
  updatedAt?: string;
}

export interface CaptureRulesSaveResult {
  created: number;
  updated: number;
  unchanged: number;
}

// Quick capture (WM-CAP-15..19): the phone posts a screenshot, the engine
// reads it into drafts in the background, the user approves or rejects them.

export type QuickCaptureStatus = 'queued' | 'processing' | 'done' | 'failed';

export interface QuickCapture {
  id: string;
  createdAt: string;
  status: QuickCaptureStatus;
  note?: string;
  hasImage: boolean;
  attempts: number;
  nextAttemptAt?: string;
  message?: string;
  newDrafts: number;
}

export interface QuickCaptureReceipt extends QuickCapture {
  /** Set when the same image and note were sent before. */
  duplicate: boolean;
}

export interface QuickCaptureOverview {
  captures: QuickCapture[];
  pendingDrafts: number;
}

export interface CaptureDraft {
  id: string;
  status: 'pending' | 'approved' | 'rejected';
  createdAt: string;
  decidedAt?: string;
  captureIds: string[];
  suggestion: CaptureSuggestion;
  /** The row as the user last saved it; the suggestion keeps what the engine read. */
  edit?: CaptureDraftEdit;
  transactionId?: string;
}

/** A pending draft's row saved without approving it (WM-CAP-19). */
export interface CaptureDraftEdit {
  transaction: TransactionInput;
  categorySource?: CaptureCategorySource;
}

export interface CaptureMaterialEntry {
  id: string;
  captureId: string;
  kind: 'image' | 'text';
  name: string;
  mediaType?: string;
  text?: string;
  createdAt: string;
}

export interface CaptureDrafts {
  drafts: CaptureDraft[];
  materials: CaptureMaterialEntry[];
  balances: CaptureBalance[];
}
