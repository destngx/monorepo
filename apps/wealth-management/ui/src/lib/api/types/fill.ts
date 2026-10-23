// AI form filling (§10.5, WM-FILL-1). Mirrors domain/fill.go.

export type FillFormKey =
  | 'transaction'
  | 'goal'
  | 'recurring'
  | 'budget-limits'
  | 'budget-move'
  | 'category-definition'
  | 'allocation'
  | 'account'
  | 'minimum-balance'
  | 'account-savings'
  | 'best-card'
  | 'stablecoin'
  | 'fund-trade'
  | 'crypto-holdings'
  | 'card'
  | 'cashback-program'
  | 'merchant';

export interface FillImagePayload {
  mediaType: string;
  /** Base64 image bytes. */
  data: string;
}

export interface FillRequest {
  form: FillFormKey;
  text?: string;
  images?: FillImagePayload[];
  /** What the form holds, for edits (WM-FILL-4). */
  current?: Record<string, unknown>;
}

export interface FillNote {
  field?: string;
  /** The field's label, shown with the message. */
  label?: string;
  /** 0-based row of a list form. */
  row?: number;
  message: string;
}

export type FillValues = Record<string, unknown>;

export interface FillResult {
  form: FillFormKey;
  /** Only the fields the answer set, each checked by the engine. */
  values: FillValues;
  rows?: FillValues[];
  /** Fields that differ from `current`; "rows" when rows were read. */
  filled: string[];
  notes: FillNote[];
}

/** A fill the chat ran, opened in its form on `page` (WM-FILL-7). */
export interface FillChatForm {
  form: FillFormKey;
  title: string;
  page: string;
  text: string;
  result: FillResult;
}
