// Mirrors the engine's JSON (apps/health-watcher/domain).

export type Role = 'owner' | 'editor' | 'viewer';

export interface Tenant {
  id: string;
  name: string;
  createdAt: string;
}

export interface Membership {
  tenant: Tenant;
  role: Role;
}

export interface User {
  id: string;
  email: string;
  displayName: string;
  /** May change the engine settings (HW-OPS-9). */
  admin: boolean;
  createdAt: string;
}

export interface Me {
  user: User;
  memberships: Membership[];
  csrfToken: string;
}

export interface SetupStatus {
  needsSetup: boolean;
}

export interface Member {
  userId: string;
  email: string;
  displayName: string;
  role: Role;
  joinedAt: string;
}

export interface Invite {
  id: string;
  tenantId: string;
  role: Role;
  createdBy: string;
  createdAt: string;
  expiresAt: string;
}

export interface CreatedInvite {
  invite: Invite;
  token: string;
}

export interface InviteInfo {
  tenantName: string;
  role: Role;
}

export interface UploadToken {
  id: string;
  name: string;
  tenantId: string;
  userId: string;
  createdAt: string;
  lastUsedAt: string | null;
  expiresAt: string | null;
}

export interface CreatedUploadToken {
  uploadToken: UploadToken;
  token: string;
}

export type Sex = '' | 'female' | 'male';

export interface Person {
  id: string;
  displayName: string;
  birthDate?: string;
  sex?: Sex;
  userId?: string;
  createdAt: string;
  lastVisitOn?: string;
  visitCount: number;
}

export interface PersonInput {
  displayName: string;
  birthDate: string;
  sex: Sex;
  userId: string;
}

export type ProviderKind = 'hospital' | 'clinic' | 'dental' | 'lab' | 'pharmacy' | 'other';

export interface Provider {
  id: string;
  name: string;
  kind: ProviderKind;
}

export type VisitType =
  'checkup' | 'lab' | 'imaging' | 'dental' | 'specialist' | 'emergency' | 'inpatient' | 'vaccination' | 'other';

export type DocumentKind =
  | 'lab_result'
  | 'imaging_report'
  | 'imaging_film'
  | 'prescription'
  | 'discharge_summary'
  | 'dental_record'
  | 'record_book_page'
  | 'invoice'
  | 'other';

export interface StoredFile {
  id: string;
  sha256: string;
  mime: string;
  size: number;
  originalName: string;
  createdAt: string;
}

export interface HealthDocument {
  id: string;
  visitId: string;
  personId: string;
  kind: DocumentKind;
  title: string;
  /** The date printed on the paper; absent means the visit date. */
  date?: string;
  files: StoredFile[];
  createdAt: string;
}

export interface Visit {
  id: string;
  personId: string;
  date: string;
  /** The last day of an inpatient stay; absent for a visit of one day. */
  endDate?: string;
  type: VisitType;
  title: string;
  note: string;
  provider: Provider | null;
  documentCount: number;
  /** Approved values the paper flagged high or low (HW-FIND-1). */
  flaggedCount: number;
  exportGaps?: ExportGap[];
  documents?: HealthDocument[];
  createdAt: string;
}

export interface VisitInput {
  personId: string;
  date: string;
  /** An inpatient stay's last day; '' clears it, absent keeps it. */
  endDate?: string;
  type: VisitType;
  title: string;
  note: string;
  providerName: string;
  providerKind: ProviderKind | '';
}

export interface VisitFilter {
  type?: VisitType | '';
  providerId?: string;
  from?: string;
  to?: string;
}

export type InboxSource = 'web' | 'shortcut' | 'phone';

export interface InboxItem {
  id: string;
  status: 'new' | 'filed' | 'discarded';
  source: InboxSource;
  note: string;
  uploadedBy: string;
  files: StoredFile[];
  documentId?: string;
  /** The backlog person every page belongs to (HW-IMP-6). */
  personId?: string;
  ocr?: OCRProgress;
  reading?: Reading;
  createdAt: string;
}

export interface OCRProgress {
  total: number;
  done: number;
  failed: number;
  error?: string;
}

export type ReadingStatus = 'waiting' | 'off' | 'needs_check' | 'queued' | 'done' | 'failed' | 'skipped';

/** The reading's suggestion for filing; the person comes from the local name match. */
export interface Proposal {
  personId?: string;
  date?: string;
  visitType?: VisitType;
  providerName?: string;
  providerKind?: ProviderKind;
  kind?: DocumentKind;
  title?: string;
  confidence: number;
}

export interface Reading {
  itemId: string;
  status: ReadingStatus;
  reason?: string;
  model?: string;
  proposal?: Proposal;
  attempts: number;
  nextAttemptAt?: string;
  error?: string;
  pagesSent: number;
  updatedAt: string;
}

/** A position on a page, 0-1 from the top-left corner. */
export interface Point {
  x: number;
  y: number;
}

/** Four corners: top-left, top-right, bottom-right, bottom-left. */
export type Quad = [Point, Point, Point, Point];

export type MaskStatus = 'ready' | 'needs_check' | 'checked';

/** A box the masking rules put on a page; a person can turn it off when it hides clinical text. */
export interface MaskBox {
  /** The line or barcode it covers, e.g. "line:4" or "barcode:0". */
  key: string;
  kind: 'line' | 'barcode';
  quad: Quad;
  /** The line it hides, "" for a barcode. */
  text: string;
  off: boolean;
}

export interface PageMask {
  status: MaskStatus;
  reasons: string[] | null;
  boxes: Quad[] | null;
  /** The rule boxes, numbered in this order, including the ones turned off. */
  auto: MaskBox[] | null;
  manual: Quad[] | null;
  maskedLines: number;
  /** The masked text: personal lines replaced with [ẩn]. */
  text: string;
}

export interface Page {
  id: string;
  fileId: string;
  number: number;
  method: 'text' | 'ocr';
  text: string;
  confidence: number;
  width: number;
  height: number;
  hasImage: boolean;
  /** What the page holds, written by the AI (HW-READ-9); missing until a reading names it. */
  title?: string;
  mask?: PageMask;
}

export type RecordStatus = 'draft' | 'approved' | 'rejected';

/**
 * What a record lacks for a complete FHIR export (D9): an inpatient stay's
 * last day, or who approved a record approved before approvals were kept.
 */
export type ExportGap = 'end_date' | 'approver';

/** How a record was made (D9): AI reading, typed by a person, or imported. */
export type RecordSource = 'reading' | 'manual' | 'import';

/** Version and origin of an observation or finding (DATA-MODEL.md §5). */
export interface Provenance {
  /** 1 at first; each edit of an approved record raises it. */
  version: number;
  source: RecordSource;
  /** The reading's model, or the import id. */
  sourceDetail?: string;
  approvedBy?: string;
  approvedAt?: string;
  exportGaps?: ExportGap[];
}

export interface Observation extends Provenance {
  id: string;
  status: RecordStatus;
  itemId?: string;
  documentId?: string;
  personId?: string;
  pageId?: string;
  pageNumber?: number;
  analyte: string;
  metricCode?: string;
  value: string;
  valueNum?: number;
  unit: string;
  refText: string;
  refLow?: number;
  refHigh?: number;
  /** H, L or *, as printed; empty when not flagged. */
  flag: string;
  sampleDate?: string;
  /** The reader's confidence, 0-100; 100 once a person edits it. */
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface ObservationInput {
  analyte: string;
  metricCode: string;
  value: string;
  unit: string;
  reference: string;
  flag: string;
  sampleDate: string;
  pageId: string;
}

export type FindingKind =
  | 'diagnosis'
  | 'exam'
  | 'prescription'
  | 'procedure'
  | 'imaging_report'
  | 'advice'
  | 'allergy'
  | 'vaccination'
  | 'note';

export interface Finding extends Provenance {
  id: string;
  status: RecordStatus;
  itemId?: string;
  documentId?: string;
  personId?: string;
  pageId?: string;
  pageNumber?: number;
  kind: FindingKind;
  section?: string;
  text: string;
  icd10?: string;
  drug?: string;
  dose?: string;
  frequency?: string;
  duration?: string;
  confidence: number;
  createdAt: string;
  updatedAt: string;
}

export interface FindingInput {
  kind: FindingKind;
  section: string;
  text: string;
  icd10: string;
  drug: string;
  dose: string;
  frequency: string;
  duration: string;
  pageId: string;
}

export type AuditEntity = 'observation' | 'finding';

export interface AuditEntry {
  id: string;
  entity: AuditEntity;
  entityId: string;
  action: 'update' | 'delete';
  /** The version the old value had. */
  version: number;
  userId: string;
  at: string;
  /** The record as it was before the change, as JSON. */
  old: string;
}

export interface ItemDetail {
  item: InboxItem;
  pages: Page[];
  observations: Observation[];
  findings: Finding[];
  aiEnabled: boolean;
}

export interface DocumentDetail {
  document: HealthDocument;
  visit: Visit;
  pages: Page[];
  observations: Observation[];
  findings: Finding[];
}

export interface InboxGroup {
  personId: string;
  date: string;
  providerName: string;
  providerKind: ProviderKind;
  visitType: VisitType;
  items: InboxItem[];
}

export interface GroupFileInput {
  visit: VisitInput;
  documents: { itemId: string; kind: DocumentKind; title: string }[];
}

export type SearchKind = 'page' | 'finding' | 'visit';

export interface SnippetPart {
  text: string;
  match: boolean;
}

export interface SearchHit {
  kind: SearchKind;
  snippet: SnippetPart[] | null;
  personId?: string;
  personName?: string;
  date?: string;
  visitId?: string;
  visitTitle?: string;
  documentId?: string;
  documentKind?: DocumentKind;
  documentTitle?: string;
  itemId?: string;
  pageId?: string;
  pageNumber?: number;
  providerName?: string;
}

export interface SearchResult {
  query: string;
  hits: SearchHit[];
}

export type MetricCategory =
  | 'glucose'
  | 'lipids'
  | 'kidney'
  | 'liver'
  | 'blood_count'
  | 'thyroid'
  | 'electrolytes'
  | 'inflammation'
  | 'iron_vitamins'
  | 'hepatitis'
  | 'tumor_markers'
  | 'coagulation'
  | 'enzymes'
  | 'urine'
  | 'vitals'
  /** Metrics made from printed names the catalog doesn't know (HW-MET-7). */
  | 'printed';

export interface Metric {
  code: string;
  loinc: string;
  name: string;
  nameVi: string;
  category: MetricCategory;
  unit: string;
  /** Units the engine converts from: canonical = value × factor + offset. */
  conversions?: { from: string; factor: number; offset?: number }[];
  aliases: string[] | null;
  urine?: boolean;
  /** Made from a printed name the catalog doesn't know (HW-MET-7): no LOINC code, conversions or description. */
  printed?: boolean;
}

export interface UploadResult {
  name: string;
  file: StoredFile;
  duplicate: boolean;
}

export interface UploadResponse {
  items: InboxItem[];
  files: UploadResult[];
}

export interface FileInput {
  visitId?: string;
  visit?: VisitInput;
  kind: DocumentKind;
  title: string;
}

export interface Filed {
  visit: Visit;
  document: HealthDocument;
}

/** Engine settings, for every family on the Mac (HW-OPS-9). The API key is never sent back. */
export interface EngineSettings {
  host: string;
  port: number;
  maxFileMb: number;
  maxBatchMb: number;
  /** Empty means the mac-ocr build next to the engine. */
  macOcrBin: string;
  workers: number;
  /** Empty turns AI reading off. */
  aiBaseUrl: string;
  aiTimeoutSeconds: number;
}

export type RestartField = 'host' | 'port' | 'maxBatchMb';

export interface EngineSettingsView {
  settings: EngineSettings;
  aiApiKeySet: boolean;
  aiModel: string;
  aiEffort: string;
  defaultMacOcrBin: string;
  macOcrFound: boolean;
  listening: string;
  restartFields: RestartField[];
  /** Fields whose stored value failed the checks at start; they use the default until the next save. */
  rejected: (keyof EngineSettings | 'aiApiKey')[];
  stamp: { updatedAt: string | null; updatedBy: string; updatedByName: string };
}

export interface EngineSettingsInput extends EngineSettings {
  /** Absent keeps the saved key; empty removes it. */
  aiApiKey?: string;
}

export interface AICheck {
  ok: boolean;
  message: string;
}

/** One rate-limit window of the AI provider (HW-OPS-10). */
export interface AIUsageWindow {
  name: string; // "5h" | "weekly"
  usedPercent: number;
  resetsAt?: string;
}

/** Token totals an API-key account reports instead of windows. */
export interface AIUsageTotals {
  since?: string;
  inputTokens: number;
  outputTokens: number;
  requests: number;
}

export interface AIUsage {
  provider: string;
  authMode?: string;
  source?: string;
  windows: AIUsageWindow[];
  totals?: AIUsageTotals;
}

/** The menu bar quota; enabled is false while AI reading is off. */
export interface AIUsageView {
  enabled: boolean;
  usage?: AIUsage;
  error?: string;
}

/** A phone capture link (HW-IMP-9), shown once. qr is a PNG data URL of url. */
export interface CaptureLink {
  url: string;
  qr: string;
  expiresAt: string;
}

/** Where a trend point came from: a filed paper, typed in without one, or computed (BMI). */
export type PointSource = 'paper' | 'typed' | 'computed';

/**
 * One value on a person's trend (HW-MET-3). `value` and the range are in the
 * metric's canonical unit; `value` is missing for a text result or a unit
 * the engine can't convert. The range and flag are the paper's own (HW-MET-5).
 */
export interface TrendPoint {
  observationId?: string;
  documentId?: string;
  pageId?: string;
  /** The page's number within its document, counting across the document's files. */
  pageNumber?: number;
  pageTitle?: string;
  visitId?: string;
  provider?: string;
  date: string;
  value?: number;
  printed: string;
  printedUnit?: string;
  refLow?: number;
  refHigh?: number;
  refText?: string;
  flag?: string;
  source: PointSource;
}

/** One metric on a person's overview (HW-MET-4). */
export interface OverviewItem {
  metric: Metric;
  latest: TrendPoint;
  previous?: TrendPoint;
  /** Latest minus previous, in the canonical unit, when both are numbers. */
  change?: number;
  count: number;
  /** Up to the last 12 numeric points, oldest first. */
  recent: TrendPoint[];
}

export interface OverviewGroup {
  category: MetricCategory;
  items: OverviewItem[];
}

export interface MetricsOverview {
  person: Person;
  groups: OverviewGroup[];
  /** Approved results not linked to a catalog metric yet. */
  unlinked: number;
}

/** What a metric is, quoted word for word from a trusted source (HW-MET-6). */
export interface MetricAbout {
  quote: string;
  /** Who publishes the source, such as MedlinePlus. */
  source: string;
  /** The title of the quoted page. */
  title: string;
  url: string;
  /** The day the quote was checked against the page, YYYY-MM-DD. */
  fetched: string;
}

export interface MetricTrend {
  person: Person;
  metric: Metric;
  about?: MetricAbout;
  points: TrendPoint[];
}

/** Vitals typed in for one day without a paper (HW-MET-2). An empty unit is the metric's own. */
export interface MeasurementInput {
  date: string;
  values: { metricCode: string; value: string; unit?: string }[];
}
