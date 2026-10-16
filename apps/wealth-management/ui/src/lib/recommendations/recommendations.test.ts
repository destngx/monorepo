import { describe, expect, it } from 'vitest';
import type { Recommendation, RecommendationSummary } from '$lib/api/types';
import { addDays, groupByStatus, headline, recordNote, sourceLabel, STATUS_TONE } from './recommendations';

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

  it('writes the headline with the yearly worth, masked on request', () => {
    expect(headline(summary(0, 0))).toBe('No open recommendations');
    expect(headline(summary(1, 0))).toBe('1 open recommendation');
    expect(headline(summary(3, 1_200_000))).toMatch(/^3 open recommendations, worth about 1\.200\.000.* a year$/);
    expect(headline(summary(3, 1_200_000), true)).toBe('3 open recommendations, worth about •••••• a year');
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
