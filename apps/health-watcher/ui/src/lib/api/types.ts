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
  files: StoredFile[];
  createdAt: string;
}

export interface Visit {
  id: string;
  personId: string;
  date: string;
  type: VisitType;
  title: string;
  note: string;
  provider: Provider | null;
  documentCount: number;
  /** Approved values the paper flagged high or low (HW-FIND-1). */
  flaggedCount: number;
  documents?: HealthDocument[];
  createdAt: string;
}

export interface VisitInput {
  personId: string;
  date: string;
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

export type InboxSource = 'web' | 'shortcut';

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

export interface PageMask {
  status: MaskStatus;
  reasons: string[] | null;
  boxes: Quad[] | null;
  manual: Quad[] | null;
  maskedLines: number;
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
  mask?: PageMask;
}

export type RecordStatus = 'draft' | 'approved' | 'rejected';

export interface Observation {
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
  'diagnosis' | 'prescription' | 'procedure' | 'imaging_report' | 'allergy' | 'vaccination' | 'note';

export interface Finding {
  id: string;
  status: RecordStatus;
  itemId?: string;
  documentId?: string;
  personId?: string;
  pageId?: string;
  pageNumber?: number;
  kind: FindingKind;
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
  | 'vitals';

export interface Metric {
  code: string;
  loinc: string;
  name: string;
  nameVi: string;
  category: MetricCategory;
  unit: string;
  aliases: string[] | null;
  urine?: boolean;
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
