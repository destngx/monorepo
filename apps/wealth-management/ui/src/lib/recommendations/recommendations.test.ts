import { describe, expect, it } from 'vitest';
import type { Recommendation, RecommendationSummary } from '$lib/api/types';
import { maskMoney } from '$lib/ai/moneyMask';
import { settled, type Settled } from '$lib/api/load';
import {
  addDays,
  checksSkipped,
  groupByStatus,
  headline,
  recordNote,
  sourceLabel,
  STATUS_TONE,
  storedOrChecked,
} from './recommendations';

const item = (id: string, status: Recommendation['status']): Recommendation => ({
  id,
  row: 2,
  created: '2026-10-01',
  source: 'budget-review',
  title: `Item ${id}`,
  action: 'Do it',
  impactVnd: null,
  evidencePath: '/budget',
  evidence: 'facts',
  status,
  page: '/budget',
  etag: 'e',
});

const summary = (open: number, openImpactVnd: number): RecommendationSummary => ({
  open,
  snoozed: 0,
  done: 0,
  dismissed: 0,
  openImpactVnd,
  openWithoutImpact: 0,
});

describe('recommendations', () => {
  it('groups items by status in order', () => {
    const groups = groupByStatus([item('a', 'done'), item('b', 'open'), item('c', 'snoozed'), item('d', 'open')]);
    expect(groups.open.map((entry) => entry.id)).toEqual(['b', 'd']);
    expect(groups.snoozed).toHaveLength(1);
    expect(groups.done).toHaveLength(1);
    expect(groups.dismissed).toHaveLength(0);
  });

  it('writes the headline with the yearly worth, for MaskedText to mask', () => {
    expect(headline(summary(0, 0))).toBe('No open recommendations');
    expect(headline(summary(1, 0))).toBe('1 open recommendation');
    expect(headline(summary(3, 1_200_000))).toMatch(/^3 open recommendations, worth about 1\.200\.000.* a year$/);
    expect(maskMoney(headline(summary(3, 1_200_000)))).toBe('3 open recommendations, worth about •••••• a year');
  });

  it('labels reviews and engine checks', () => {
    expect(sourceLabel('year-review')).toBe('Year review');
    expect(sourceLabel('rule:goal-plan-short')).toBe('Goal plan check');
    expect(sourceLabel('rule:new-one')).toBe('Engine check');
  });

  it('never shows a status by colour alone: every status maps to a token', () => {
    expect(Object.keys(STATUS_TONE).sort()).toEqual(['dismissed', 'done', 'open', 'snoozed']);
  });

  it('adds calendar days for a snooze', () => {
    expect(addDays(new Date(2026, 9, 28), 7)).toBe('2026-11-04');
  });

  it('summarises what a review wrote', () => {
    expect(recordNote(undefined)).toBe('');
    expect(recordNote({ created: [item('a', 'open')], duplicates: 1, checked: [], ignoredOutcomes: 0 })).toBe(
      'Recommendations: 1 added, 1 already on the list.',
    );
    expect(recordNote({ created: [], duplicates: 0, checked: [], ignoredOutcomes: 0 })).toBe(
      'No new recommendations from this review.',
    );
    expect(recordNote({ created: [], duplicates: 0, checked: [], ignoredOutcomes: 0, error: 'Could not save' })).toBe(
      'Could not save',
    );
  });
});

describe('storedOrChecked', () => {
  const failed = (error: string): Promise<Settled<string>> => Promise.resolve({ ok: false, error });

  it('Given the stored read loads When the refresh never answers Then the stored list shows without waiting', async () => {
    expect(await storedOrChecked(settled('stored'), new Promise(() => undefined))).toEqual({
      ok: true,
      value: 'stored',
    });
  });

  it('Given the stored read fails When the refresh answers Then its list shows', async () => {
    expect(await storedOrChecked(failed('read down'), settled('checked'))).toEqual({ ok: true, value: 'checked' });
  });

  it('Given both fail When settled Then the read failure is kept', async () => {
    expect(await storedOrChecked(failed('read down'), failed('no write access'))).toEqual({
      ok: false,
      error: 'read down',
    });
  });
});

describe('checksSkipped', () => {
  const failure = (status: number, sheetsAuth: 'expired' | null = null) => ({
    kind: 'auth' as const,
    message: 'forbidden',
    sheetsAuth,
    hint: null,
    status,
  });

  it('Given a 403 without a Sheets credential problem Then the checks were skipped for write access', () => {
    expect(checksSkipped(failure(403))).toBe(true);
    expect(checksSkipped(failure(401))).toBe(true);
  });

  it('Given an expired Sheets token, another error or no failure Then it is not the write-access case', () => {
    expect(checksSkipped(failure(401, 'expired'))).toBe(false);
    expect(checksSkipped({ ...failure(500), kind: 'other' })).toBe(false);
    expect(checksSkipped(null)).toBe(false);
  });
});
