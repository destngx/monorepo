import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import NeedsYouCard from './NeedsYouCard.svelte';
import { needsYou, needsYouItem } from './needsYou.testdata';

vi.mock('$app/paths', () => ({ base: '/ui' }));

const rows = (container: HTMLElement) => [...container.querySelectorAll('li[data-kind]')] as HTMLElement[];

describe('NeedsYouCard', () => {
  afterEach(() => mask.hide());

  it('leads with the answer and lists the items in the engine order, each with its status icon and label', () => {
    const { container } = render(NeedsYouCard, { inbox: needsYou() });

    expect(screen.getByRole('heading', { level: 2 }).textContent).toContain('4 things need you');
    expect(container.querySelector('section')?.getAttribute('data-status')).toBe('critical');
    expect(rows(container).map((row) => row.dataset.kind)).toEqual([
      'bill-overdue',
      'card-due',
      'goal-plan-short',
      'recommendation',
    ]);
    const card = rows(container)[1];
    expect(card.querySelector('[data-status="serious"]')?.textContent).toContain('Serious');
    expect(card.querySelector('[data-status] svg')).toBeTruthy();
  });

  it('links each item to the page where it is acted on', () => {
    const { container } = render(NeedsYouCard, { inbox: needsYou() });
    const links = rows(container).map((row) => row.querySelector('a')?.getAttribute('href'));
    expect(links).toEqual(['/ui/bills', '/ui/bills', '/ui/accounts/goals', '/ui/recommendations?page=%2Fbudget']);
  });

  it('masks amounts, also inside the engine wording, until revealed', () => {
    const { container } = render(NeedsYouCard, { inbox: needsYou() });
    const goal = rows(container)[2];
    expect(goal.textContent).toContain(`Goal plan is short by ${MASK} a month`);
    expect(goal.textContent).not.toContain('2.000.000');
    expect(goal.textContent).toContain('a month');

    mask.toggle();
    const revealed = render(NeedsYouCard, { inbox: needsYou() });
    expect(rows(revealed.container)[2].textContent).toContain('Goal plan is short by 2.000.000 ₫ a month');
  });

  it('shows a calm state when nothing needs the user', () => {
    const { container } = render(NeedsYouCard, { inbox: needsYou({ items: [] }) });
    expect(screen.getByRole('heading', { level: 2 }).textContent).toContain('Nothing needs you');
    expect(container.querySelector('section')?.getAttribute('data-status')).toBe('good');
    expect(rows(container)).toHaveLength(0);
  });

  it('shows six rows first and the rest on demand', async () => {
    const items = Array.from({ length: 8 }, (_, index) => needsYouItem({ id: `item-${index}` }));
    const { container } = render(NeedsYouCard, { inbox: needsYou({ items }) });
    expect(rows(container)).toHaveLength(6);
    await fireEvent.click(screen.getByText('Show all 8'));
    expect(rows(container)).toHaveLength(8);
  });
});
