import { describe, expect, it } from 'vitest';
import type { InboxItem, Observation, Person, Reading } from './api/types';
import { findingDetails, findingInput, observationInput, oldRecord, proposalVisit, readingState } from './reading';

function item(over: Partial<InboxItem> = {}): InboxItem {
  return {
    id: 'i1',
    status: 'new',
    source: 'web',
    note: '',
    uploadedBy: 'u1',
    files: [],
    createdAt: '2026-10-21T00:00:00Z',
    ocr: { total: 1, done: 1, failed: 0 },
    ...over,
  };
}

function reading(over: Partial<Reading>): Reading {
  return { itemId: 'i1', status: 'done', attempts: 0, pagesSent: 1, updatedAt: '2026-10-21T00:00:00Z', ...over };
}

const person = (id: string): Person => ({ id, displayName: id, createdAt: '', visitCount: 0 });

describe('readingState', () => {
  it('shows OCR progress until every file has pages', () => {
    const s = readingState(item({ ocr: { total: 3, done: 1, failed: 0 } }));
    expect(s.tone).toBe('busy');
    expect(s.label).toContain('1 of 3');
  });
  it('shows an OCR failure with its error', () => {
    const s = readingState(item({ ocr: { total: 1, done: 0, failed: 1, error: 'mac-ocr crashed' } }));
    expect(s).toMatchObject({ tone: 'error', detail: 'mac-ocr crashed' });
  });
  it('shows an OCR retry as a warning while it is still queued', () => {
    expect(readingState(item({ ocr: { total: 1, done: 0, failed: 0, error: 'timeout' } })).tone).toBe('warn');
  });
  it('asks for a check when masked pages need one', () => {
    const s = readingState(item({ reading: reading({ status: 'needs_check', reason: 'Page 1: no name' }) }));
    expect(s).toMatchObject({ tone: 'warn', detail: 'Page 1: no name' });
  });
  it('shows a retrying reading with the provider error', () => {
    const s = readingState(item({ reading: reading({ status: 'queued', error: 'provider down', attempts: 2 }) }));
    expect(s.tone).toBe('warn');
    expect(s.detail).toContain('provider down');
  });
  it('maps the other statuses to a tone', () => {
    expect(readingState(item({ reading: reading({ status: 'done' }) })).tone).toBe('ok');
    expect(readingState(item({ reading: reading({ status: 'failed' }) })).tone).toBe('error');
    expect(readingState(item({ reading: reading({ status: 'off' }) })).tone).toBe('neutral');
    expect(readingState(item({ reading: reading({ status: 'skipped', reason: 'No text' }) })).detail).toBe('No text');
  });
});

describe('proposalVisit', () => {
  it('prefills the visit from the proposal', () => {
    const v = proposalVisit(
      {
        personId: 'p2',
        date: '2025-03-14',
        visitType: 'lab',
        providerName: 'Lab A',
        providerKind: 'lab',
        confidence: 90,
      },
      [person('p1'), person('p2')],
      '',
    );
    expect(v).toMatchObject({
      personId: 'p2',
      date: '2025-03-14',
      type: 'lab',
      providerName: 'Lab A',
      providerKind: 'lab',
    });
  });
  it('falls back to the backlog person, then the only person', () => {
    expect(proposalVisit(undefined, [person('p1'), person('p2')], 'p2').personId).toBe('p2');
    expect(proposalVisit({ confidence: 0 }, [person('p1')], '').personId).toBe('p1');
    expect(proposalVisit({ confidence: 0 }, [person('p1'), person('p2')], '').personId).toBe('');
  });
  it('ignores a person who is no longer in the tenant', () => {
    expect(proposalVisit({ personId: 'gone', confidence: 50 }, [person('p1'), person('p2')], '').personId).toBe('');
  });
});

describe('record inputs', () => {
  it('turns an observation into its edit form', () => {
    const o = {
      analyte: 'Glucose',
      metricCode: 'glucose',
      value: '6.4',
      unit: 'mmol/L',
      refText: '3.9 - 6.1',
      flag: 'H',
      sampleDate: '2025-03-14',
      pageId: 'pg1',
    } as Observation;
    expect(observationInput(o)).toEqual({
      analyte: 'Glucose',
      metricCode: 'glucose',
      value: '6.4',
      unit: 'mmol/L',
      reference: '3.9 - 6.1',
      flag: 'H',
      sampleDate: '2025-03-14',
      pageId: 'pg1',
    });
  });
  it('fills blank finding fields', () => {
    expect(findingInput({ kind: 'diagnosis', text: 'Flu' } as never)).toMatchObject({
      section: '',
      icd10: '',
      drug: '',
      pageId: '',
    });
  });
  it('keeps the section of an exam result, so a status change does not drop it', () => {
    expect(findingInput({ kind: 'exam', section: 'Khám Mắt', text: 'MP: 10/10' } as never)).toMatchObject({
      kind: 'exam',
      section: 'Khám Mắt',
    });
  });
});

describe('vaccination findings', () => {
  const dose = {
    kind: 'vaccination',
    text: 'Lần 1: 15/01/2025',
    drug: 'Vaccine X',
    dose: '1ml',
    date: '2025-01-15',
  } as never;
  it('keeps the vaccine, dose and date, so approving a draft does not drop them', () => {
    expect(findingInput(dose)).toMatchObject({ drug: 'Vaccine X', dose: '1ml', date: '2025-01-15' });
    expect(findingInput({ kind: 'note', text: 'x' } as never).date).toBe('');
  });
  it('shows the vaccine and dose under the text', () => {
    expect(findingDetails(dose)).toBe('Vaccine X · 1ml');
  });
  it('shows a prescription line and nothing for other kinds', () => {
    expect(findingDetails({ kind: 'prescription', drug: 'Metformin', frequency: '2 lần/ngày' } as never)).toBe(
      'Metformin · 2 lần/ngày',
    );
    expect(findingDetails({ kind: 'diagnosis', drug: 'stale' } as never)).toBe('');
  });
});

describe('oldRecord', () => {
  it('parses the audited JSON and survives garbage', () => {
    expect(oldRecord('{"value":"6.4"}')).toEqual({ value: '6.4' });
    expect(oldRecord('not json')).toEqual({});
    expect(oldRecord('[1]')).toEqual({});
  });
});
