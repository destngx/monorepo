import { describe, expect, it } from 'vitest';
import type { GoalPlan, GoalProgress } from '$lib/api/types';
import {
  emptyGoalForm,
  finishLabel,
  formatYears,
  goalDateHint,
  goalFormFor,
  goalInputFrom,
  groupGoals,
  looksLikeEmergencyFund,
  inOrder,
  moveId,
  paceStatus,
  verdictStatus,
} from './goals';

const goal = (overrides: Partial<GoalProgress> = {}): GoalProgress => ({
  id: 'goal-2',
  row: 2,
  name: 'Trip',
  target: 30_000_000,
  priority: 1,
  startingAmount: 0,
  status: 'active',
  etag: 'e1',
  useSlowMoney: false,
  emergencyFund: false,
  targetAmount: 30_000_000,
  targetComputed: false,
  funded: 10_000_000,
  fromPool: 10_000_000,
  fromSlow: 0,
  remaining: 20_000_000,
  percent: 33.3,
  monthsLeft: 10,
  requiredPerMonth: 2_000_000,
  projectedFinish: '2027-06',
  monthsToReach: 8,
  yearsToReach: 0.7,
  longHorizon: false,
  pace: 'on_track',
  ...overrides,
});

describe('goal statuses', () => {
  it('gives every pace and verdict a status tone and label', () => {
    expect(paceStatus('behind')).toEqual({ tone: 'warning', label: 'Behind' });
    expect(paceStatus('paused').tone).toBe('neutral');
    expect(paceStatus('no_deadline').label).toBe('No deadline');
    expect(verdictStatus('short').tone).toBe('serious');
    expect(verdictStatus('fits').tone).toBe('good');
  });

  it('describes the projected finish', () => {
    expect(finishLabel(goal())).toBe('Jun 2027');
    expect(finishLabel(goal({ projectedFinish: null, monthsToReach: null }))).toBe('Not reachable at this pace');
    expect(finishLabel(goal({ monthsToReach: 0, projectedFinish: '2026-10' }))).toBe('Funded now');
    expect(finishLabel(goal({ pace: 'done' }))).toBe('Done');
    expect(formatYears(8.7)).toBe('8.7 years');
    expect(formatYears(null)).toBe('-');
  });
});

describe('goalDateHint', () => {
  it('explains a blank date as no deadline', () => {
    expect(goalDateHint('')).toBe('Blank means no deadline; the goal shows its projected finish only.');
  });

  it('counts savings to the end of the month and points to the month before for early needs', () => {
    expect(goalDateHint('2026-12')).toBe(
      "Saved by the end of Dec 2026. Need it before Dec's wage arrives? Pick Nov 2026.",
    );
  });

  it('steps back across a year boundary', () => {
    expect(goalDateHint('2027-01')).toBe(
      "Saved by the end of Jan 2027. Need it before Jan's wage arrives? Pick Dec 2026.",
    );
  });
});

describe('goal lists', () => {
  it('groups active, paused and done goals keeping priority order', () => {
    const plan = {
      goals: [
        goal({ id: 'a' }),
        goal({ id: 'b', status: 'done' }),
        goal({ id: 'c', status: 'paused' }),
        goal({ id: 'd' }),
      ],
    } as GoalPlan;
    const groups = groupGoals(plan);
    expect(groups.active.map((g) => g.id)).toEqual(['a', 'd']);
    expect(groups.paused.map((g) => g.id)).toEqual(['c']);
    expect(groups.done.map((g) => g.id)).toEqual(['b']);
  });
});

describe('goal form', () => {
  it('recognises the emergency fund like the engine', () => {
    expect(looksLikeEmergencyFund('Quỹ khẩn cấp')).toBe(true);
    expect(looksLikeEmergencyFund('Safety net', 'emergency money')).toBe(true);
    expect(looksLikeEmergencyFund('New car')).toBe(false);
  });

  it('builds the input and names the first problem', () => {
    const form = { ...emptyGoalForm(), name: ' Trip ', target: 30_000_000, targetDate: '2027-06' };
    expect(goalInputFrom(form, false).input).toEqual({
      name: 'Trip',
      target: 30_000_000,
      targetDate: '2027-06',
      priority: undefined,
      startingAmount: undefined,
      status: 'active',
      note: undefined,
      useSlowMoney: undefined,
    });
    expect(goalInputFrom({ ...form, useSlowMoney: true }, false).input?.useSlowMoney).toBe(true);
    expect(goalInputFrom({ ...form, name: '' }, false).error).toMatch(/Name/);
    expect(goalInputFrom({ ...form, target: null }, false).error).toMatch(/Target is required/);
    expect(goalInputFrom({ ...form, target: null }, true).input?.target).toBeUndefined();
    expect(goalInputFrom({ ...form, startingAmount: -1 }, false).error).toMatch(/negative/);
  });

  it('fills the form from a goal', () => {
    expect(goalFormFor(goal({ targetDate: '2027-06', note: 'n', useSlowMoney: true }))).toMatchObject({
      name: 'Trip',
      target: 30_000_000,
      targetDate: '2027-06',
      priority: 1,
      startingAmount: null,
      note: 'n',
      useSlowMoney: true,
    });
  });
});

describe('moveId', () => {
  it('moves one place up or down', () => {
    expect(moveId(['a', 'b', 'c'], 1, 0)).toEqual(['b', 'a', 'c']);
    expect(moveId(['a', 'b', 'c'], 1, 2)).toEqual(['a', 'c', 'b']);
  });

  it('takes the place of the card it lands on, shifting the cards between', () => {
    expect(moveId(['a', 'b', 'c', 'd'], 0, 2)).toEqual(['b', 'c', 'a', 'd']);
    expect(moveId(['a', 'b', 'c', 'd'], 3, 1)).toEqual(['a', 'd', 'b', 'c']);
  });

  it('keeps the order for the same place or an index out of range', () => {
    const ids = ['a', 'b'];
    expect(moveId(ids, 1, 1)).toBe(ids);
    expect(moveId(ids, 0, 5)).toBe(ids);
    expect(moveId(ids, -1, 0)).toBe(ids);
  });
});

describe('inOrder', () => {
  it('shows goals in a pending order while the reorder saves, keeping goals the order misses', () => {
    const goals = [goal({ id: 'a' }), goal({ id: 'b' }), goal({ id: 'c' })];
    expect(inOrder(goals, ['c', 'a', 'b']).map((g) => g.id)).toEqual(['c', 'a', 'b']);
    expect(inOrder(goals, ['b', 'gone']).map((g) => g.id)).toEqual(['b', 'a', 'c']);
    expect(inOrder(goals, null)).toBe(goals);
  });
});
