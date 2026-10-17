import { render } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { mask } from '$lib/state/mask.svelte';
import LineChart from './LineChart.svelte';

afterEach(() => mask.hide());

describe('LineChart', () => {
  it('Given exactly two points When it renders Then each date labels the axis once', () => {
    const { container } = render(LineChart, {
      points: [
        { date: '2026-01-01', value: 100_000 },
        { date: '2026-02-01', value: 262_000 },
      ],
    });
    const dates = [...container.querySelectorAll('svg text')]
      .map((label) => label.textContent ?? '')
      .filter((text) => /^\d{2}\/\d{2}\/\d{4}$/.test(text));
    expect(dates).toEqual(['01/01/2026', '01/02/2026']);
  });
});
