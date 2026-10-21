import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it } from 'vitest';
import ConfidenceBadge from './ConfidenceBadge.svelte';

describe('ConfidenceBadge', () => {
  beforeEach(() => {
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
  });

  it('Given an overall confidence When clicked Then the reasons and details open in a popup', async () => {
    render(ConfidenceBadge, {
      overall: 8,
      reasons: [
        { text: 'Read from the material with 48% confidence', effect: 48 },
        { text: 'No account: pick one before approving', effect: -40 },
        { text: 'May be missing from the sheet' },
      ],
      notes: ['No time on the material; matched by date only.'],
    });

    const badge = screen.getByRole('button', { name: /Confidence 8%/ });
    expect(screen.queryByText('No account: pick one before approving')).toBeNull();

    await fireEvent.click(badge);

    expect(screen.getByText('No account: pick one before approving')).toBeTruthy();
    expect(screen.getByText('−40')).toBeTruthy();
    expect(screen.getByText('48%')).toBeTruthy();
    expect(screen.getByText('May be missing from the sheet')).toBeTruthy();
    expect(screen.getByText(/No time on the material/)).toBeTruthy();
  });
});
