import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import RiskCard from './RiskCard.svelte';
import RuleBasedNote from './RuleBasedNote.svelte';

describe('rule-based disclaimer', () => {
  it('is said once by the page note', () => {
    render(RuleBasedNote);
    expect(screen.getByTestId('rule-based-note').textContent).toContain('not a forecast');
  });

  it('is not repeated by the cards', () => {
    const { container } = render(RiskCard, { risk: null });
    expect(container.textContent).not.toMatch(/forecast|heuristic/i);
  });
});
