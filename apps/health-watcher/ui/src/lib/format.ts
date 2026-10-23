import type { DocumentKind, FindingKind, MetricCategory, ProviderKind, Role, VisitType } from './api/types';

/** `2025-03-04` as `04/03/2025`; empty for empty or malformed input. */
export function formatDate(iso: string | undefined | null): string {
  const match = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso ?? '');
  return match ? `${match[3]}/${match[2]}/${match[1]}` : '';
}

/** A visit's dates as dd/MM/yyyy, with the last day of a stay when there is one. */
export function formatDateRange(start: string, end?: string): string {
  return end && end !== start ? `${formatDate(start)} – ${formatDate(end)}` : formatDate(start);
}

/** Today's date in Asia/Ho_Chi_Minh as YYYY-MM-DD, the engine's calendar. */
export function todayISO(now: Date = new Date()): string {
  return new Intl.DateTimeFormat('en-CA', { timeZone: 'Asia/Ho_Chi_Minh' }).format(now);
}

/** Whole years between a birth date and a day (both YYYY-MM-DD); null when unknown. */
export function ageOn(birthDate: string | undefined, day: string): number | null {
  if (!birthDate || !/^\d{4}-\d{2}-\d{2}$/.test(birthDate)) return null;
  const [by, bm, bd] = birthDate.split('-').map(Number);
  const [y, m, d] = day.split('-').map(Number);
  let age = y - by;
  if (m < bm || (m === bm && d < bd)) age--;
  return age >= 0 ? age : null;
}

/** File size in B, KB or MB. */
export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const visitTypeLabels: Record<VisitType, string> = {
  checkup: 'Check-up',
  lab: 'Lab test',
  imaging: 'Imaging',
  dental: 'Dental',
  specialist: 'Specialist',
  emergency: 'Emergency',
  inpatient: 'Inpatient',
  vaccination: 'Vaccination',
  other: 'Other',
};

export const documentKindLabels: Record<DocumentKind, string> = {
  lab_result: 'Lab result',
  imaging_report: 'Imaging report',
  imaging_film: 'Imaging film',
  prescription: 'Prescription',
  discharge_summary: 'Discharge summary',
  dental_record: 'Dental record',
  record_book_page: 'Record book page',
  invoice: 'Invoice',
  other: 'Other',
};

export const providerKindLabels: Record<ProviderKind, string> = {
  hospital: 'Hospital',
  clinic: 'Clinic',
  dental: 'Dental',
  lab: 'Lab',
  pharmacy: 'Pharmacy',
  other: 'Other',
};

export const findingKindLabels: Record<FindingKind, string> = {
  diagnosis: 'Diagnosis',
  exam: 'Exam result',
  prescription: 'Prescription',
  procedure: 'Procedure',
  imaging_report: 'Imaging report',
  advice: "Doctor's advice",
  allergy: 'Allergy',
  vaccination: 'Vaccination',
  note: 'Note',
};

export const metricCategoryLabels: Record<MetricCategory, string> = {
  glucose: 'Blood sugar',
  lipids: 'Lipids',
  kidney: 'Kidney',
  liver: 'Liver',
  blood_count: 'Blood count',
  thyroid: 'Thyroid',
  electrolytes: 'Electrolytes',
  inflammation: 'Inflammation',
  iron_vitamins: 'Iron and vitamins',
  hepatitis: 'Hepatitis',
  tumor_markers: 'Tumour markers',
  coagulation: 'Coagulation',
  enzymes: 'Enzymes',
  urine: 'Urine',
  vitals: 'Vital signs',
};

/** What a printed flag means: H high, L low, * abnormal. */
export function flagLabel(flag: string): string {
  if (flag === 'H') return 'High';
  if (flag === 'L') return 'Low';
  return flag ? 'Abnormal' : '';
}

export const roleLabels: Record<Role, string> = {
  owner: 'Owner',
  editor: 'Editor',
  viewer: 'Viewer',
};

export const roleHints: Record<Role, string> = {
  owner: 'Manages members and settings, and edits records.',
  editor: 'Adds, edits and files records.',
  viewer: 'Only reads records.',
};

/** Entries of a label map in its declared order, for <select> options. */
export function options<K extends string>(labels: Record<K, string>): { value: K; label: string }[] {
  return (Object.keys(labels) as K[]).map((value) => ({ value, label: labels[value] }));
}

export function isImage(mime: string): boolean {
  return mime.startsWith('image/') && mime !== 'image/heic';
}
