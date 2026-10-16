import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import ExpenseSplitCard from './ExpenseSplitCard.svelte';

afterEach(() => mask.hide());

const months = (total: number) => ({ category: '', values: [], total });
const flow = (expense: number) => ({ month: '2026-01', income: 0, expense, cashFlow: -expense, savingsRate: 0 });
const split = {
  fixed: months(5_000_000),
  essential: months(3_000_000),
  flexible: months(2_000_000),
  untagged: months(0),
  needsShareOfIncome: 40,
  needsShareComfortable: 50,
  untaggedCategories: [],
  room: [
    {
      category: 'Food',
      nature: 'essential' as const,
      typical: 6_000_000,
      lean: 4_500_000,
      perMonth: 1_500_000,
      perYear: 18_000_000,
    },
  ],
  roomPerMonth: 1_000_000,
  roomPerYear: 12_000_000,
};
const text = (id: string) => screen.getByTestId(id).textContent?.replace(/\s+/g, ' ').trim();

describe('ExpenseSplitCard', () => {
  it('splits spending into needs and wants with the room to save', () => {
    mask.toggle();
    render(ExpenseSplitCard, { props: { split, months: [flow(1)] } });
    expect(text('split-legend')).toBe(
      'Fixed (set by contracts) 5.000.000 ₫ · 50.0% Essential (needed, can be lowered) 3.000.000 ₫ · 30.0% Flexible (wants) 2.000.000 ₫ · 20.0%',
    );
    expect(text('room-total')).toBe(
      'Living like your leaner months would free about 1.000.000 ₫ a month, 12.000.000 ₫ a year.',
    );
    expect(text('room')).toBe('Food Essential typically 6.000.000 ₫, 4.500.000 ₫ in leaner months 1.500.000 ₫ / month');
  });

  it('masks amounts but keeps the shares', () => {
    render(ExpenseSplitCard, { props: { split, months: [flow(1)] } });
    expect(text('split-legend')).toContain(`Fixed (set by contracts) ${MASK} · 50.0%`);
  });

  it('asks for more months before judging the room, and lists untagged categories', () => {
    render(ExpenseSplitCard, {
      props: {
        split: { ...split, room: [], roomPerMonth: 0, roomPerYear: 0, untaggedCategories: ['Gifts'] },
        months: [flow(1), flow(0)],
      },
    });
    expect(screen.getByText('Needs 4 completed months with spending to find your leaner months.')).toBeTruthy();
    expect(text('untagged')).toContain('Not tagged yet: Gifts.');
  });
});
