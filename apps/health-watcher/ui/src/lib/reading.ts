import type {
  Finding,
  FindingInput,
  InboxItem,
  Observation,
  ObservationInput,
  Person,
  Proposal,
  VisitInput,
} from './api/types';
import { blankVisit } from './visits';

export type Tone = 'neutral' | 'busy' | 'warn' | 'ok' | 'error';

export interface ReadingState {
  label: string;
  detail: string;
  tone: Tone;
}

/** Where an inbox item is in OCR and AI reading, for a status chip (HW-READ-6). */
export function readingState(item: InboxItem): ReadingState {
  const ocr = item.ocr;
  if (ocr && ocr.failed > 0) {
    return { label: 'Text reading failed', detail: ocr.error ?? '', tone: 'error' };
  }
  if (ocr && ocr.done < ocr.total) {
    const progress = ocr.total > 1 ? ` (${ocr.done} of ${ocr.total} files)` : '';
    if (ocr.error) return { label: `Retrying text reading${progress}`, detail: ocr.error, tone: 'warn' };
    return { label: `Reading text on the Mac${progress}`, detail: '', tone: 'busy' };
  }
  const r = item.reading;
  switch (r?.status) {
    case undefined:
    case 'waiting':
      return { label: 'Reading text on the Mac', detail: '', tone: 'busy' };
    case 'off':
      return {
        label: 'AI reading is off',
        detail:
          'File it by hand, or ask an engine admin to set the AI address in Engine settings. It is read as soon as AI is on.',
        tone: 'neutral',
      };
    case 'needs_check':
      return { label: 'Check the masked pages', detail: r.reason ?? '', tone: 'warn' };
    case 'queued':
      if (r.error) return { label: `Retrying (attempt ${r.attempts + 1})`, detail: r.error, tone: 'warn' };
      return { label: 'AI is reading', detail: '', tone: 'busy' };
    case 'done':
      return { label: 'Read by AI', detail: r.reason ?? '', tone: 'ok' };
    case 'failed':
      return { label: 'AI reading failed', detail: r.error ?? '', tone: 'error' };
    case 'skipped':
      return { label: 'Not read', detail: r.reason ?? '', tone: 'neutral' };
  }
}

/** A new visit prefilled from the reading's proposal. The person is the
 * proposal's, then the backlog person, then the only person in the tenant. */
export function proposalVisit(p: Proposal | undefined, people: Person[], backlogPersonId: string): VisitInput {
  const known = (id: string | undefined) => (id && people.some((x) => x.id === id) ? id : '');
  const personId = known(p?.personId) || known(backlogPersonId) || (people.length === 1 ? people[0].id : '');
  const v = blankVisit(personId);
  if (!p) return v;
  return {
    ...v,
    date: p.date || v.date,
    type: p.visitType ?? v.type,
    providerName: p.providerName ?? '',
    providerKind: p.providerKind ?? '',
  };
}

export function observationInput(o: Observation): ObservationInput {
  return {
    analyte: o.analyte,
    metricCode: o.metricCode ?? '',
    value: o.value,
    unit: o.unit,
    reference: o.refText,
    flag: o.flag,
    sampleDate: o.sampleDate ?? '',
    pageId: o.pageId ?? '',
  };
}

export function findingInput(f: Finding): FindingInput {
  return {
    kind: f.kind,
    section: f.section ?? '',
    text: f.text,
    icd10: f.icd10 ?? '',
    drug: f.drug ?? '',
    dose: f.dose ?? '',
    frequency: f.frequency ?? '',
    duration: f.duration ?? '',
    date: f.date ?? '',
    pageId: f.pageId ?? '',
  };
}

/** The line under a finding's text: a prescription's drug, dose, frequency and duration, or a vaccination's vaccine and dose. */
export function findingDetails(f: Finding): string {
  const parts =
    f.kind === 'prescription'
      ? [f.drug, f.dose, f.frequency, f.duration]
      : f.kind === 'vaccination'
        ? [f.drug, f.dose]
        : [];
  return parts.filter(Boolean).join(' · ');
}

/** The record an audit entry kept, or {} when it can't be read. */
export function oldRecord(json: string): Record<string, unknown> {
  try {
    const v: unknown = JSON.parse(json);
    return v && typeof v === 'object' && !Array.isArray(v) ? (v as Record<string, unknown>) : {};
  } catch {
    return {};
  }
}
