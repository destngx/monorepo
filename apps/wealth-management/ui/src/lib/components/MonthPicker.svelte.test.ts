import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import MonthPicker from './MonthPicker.svelte';

describe('MonthPicker', () => {
  beforeEach(() => {
    vi.useFakeTimers({ toFake: ['Date'] });
    vi.setSystemTime(new Date('2026-10-23T03:00:00Z'));
  });
  afterEach(() => vi.useRealTimers());

  it('Given no month When a year and a month are picked Then the value is yyyy-mm and the panel closes', async () => {
    const props = $state({ id: 'when', value: '', placeholder: 'No deadline' });
    render(MonthPicker, props);

    const button = screen.getByRole('button', { name: 'No deadline' });
    await fireEvent.click(button);
    expect(screen.getByRole('group', { name: 'Months of 2026' })).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'Next year' }));
    await fireEvent.click(screen.getByRole('button', { name: 'Mar 2027' }));

    expect(props.value).toBe('2027-03');
    expect(screen.queryByRole('dialog')).toBeNull();
    expect(screen.getByRole('button', { name: 'Mar 2027' }).getAttribute('aria-expanded')).toBe('false');
  });

  it('Given a month When opened Then it starts on that year with the month marked, and Clear empties it', async () => {
    const props = $state({ id: 'when', value: '2030-12', clearLabel: 'No deadline' });
    render(MonthPicker, props);

    await fireEvent.click(screen.getByRole('button', { name: 'Dec 2030' }));
    expect(screen.getByRole('button', { name: 'Dec 2030', pressed: true })).toBeTruthy();

    await fireEvent.click(screen.getByRole('button', { name: 'No deadline' }));
    expect(props.value).toBe('');
  });

  it('Given the panel is open When Escape is pressed Then it closes without picking', async () => {
    const props = $state({ id: 'when', value: '' });
    render(MonthPicker, props);

    await fireEvent.click(screen.getByRole('button', { name: 'Pick a month' }));
    await fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' });

    expect(screen.queryByRole('dialog')).toBeNull();
    expect(props.value).toBe('');
  });
});
