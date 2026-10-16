import { render } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { describe, expect, it } from 'vitest';
import Badge from '$lib/components/Badge.svelte';
import { STATUS, STATUS_TONES, usageTone } from './status';

// WCAG relative luminance and contrast, to hold chart colours to 3:1 on both themes.
function luminance(hex: string): number {
  const channel = (offset: number) => {
    const value = parseInt(hex.slice(offset, offset + 2), 16) / 255;
    return value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(1) + 0.7152 * channel(3) + 0.0722 * channel(5);
}
function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}
const LIGHT_SURFACE = '#fcfcfb';
const DARK_SURFACE = '#1a1a19';

describe('status tokens', () => {
  it('give every tone an icon, a label and every style', () => {
    for (const tone of STATUS_TONES) {
      const token = STATUS[tone];
      expect(token.icon, tone).toBeTruthy();
      expect(token.label, tone).toMatch(/\S/);
      for (const style of [token.ink, token.edge, token.box, token.badge, token.fill, token.wash]) {
        expect(style, tone).toMatch(/\S/);
      }
      expect(token.chart, tone).toMatch(/^#[0-9a-f]{6}$/);
    }
  });

  it('tells the four severities apart by icon, not only by colour', () => {
    const severities = ['good', 'warning', 'serious', 'critical'] as const;
    severities.forEach((tone, i) => {
      for (const other of severities.slice(i + 1)) {
        expect(STATUS[tone].icon, `${tone} vs ${other}`).not.toBe(STATUS[other].icon);
        expect(STATUS[tone].label, `${tone} vs ${other}`).not.toBe(STATUS[other].label);
      }
    });
  });

  it('keep text colours readable in dark mode', () => {
    for (const tone of STATUS_TONES) {
      expect(STATUS[tone].ink, tone).toContain('dark:');
      expect(STATUS[tone].box, tone).toContain('dark:');
    }
  });

  it('draw chart colours at 3:1 or more on the light and the dark surface', () => {
    for (const tone of STATUS_TONES) {
      expect(contrast(STATUS[tone].chart, LIGHT_SURFACE), `${tone} light`).toBeGreaterThanOrEqual(3);
      expect(contrast(STATUS[tone].chart, DARK_SURFACE), `${tone} dark`).toBeGreaterThanOrEqual(3);
    }
  });
});

describe('usageTone', () => {
  it('is good below the warning share, a warning from it and serious at the limit', () => {
    expect(usageTone(79.9)).toBe('good');
    expect(usageTone(80)).toBe('warning');
    expect(usageTone(99)).toBe('warning');
    expect(usageTone(100)).toBe('serious');
    expect(usageTone(65, 60)).toBe('warning');
  });
});

describe('Badge status', () => {
  it('shows the status icon and falls back to the status label', () => {
    const { container } = render(Badge, { status: 'critical' });
    const badge = container.querySelector('[data-status="critical"]');
    expect(badge?.textContent).toBe(STATUS.critical.label);
    expect(badge?.querySelector('svg')).toBeTruthy();
    expect(badge?.className).toContain('bg-rose-500/15');
  });

  it('keeps the caller label next to the icon', () => {
    const children = createRawSnippet(() => ({ render: () => '<span>Pay soon</span>' }));
    const { container } = render(Badge, { status: 'warning', children });
    const badge = container.querySelector('[data-status="warning"]');
    expect(badge?.textContent).toBe('Pay soon');
    expect(badge?.querySelector('svg')).toBeTruthy();
  });

  it('stays a plain hue without a status', () => {
    const children = createRawSnippet(() => ({ render: () => '<span>Tag</span>' }));
    const { container } = render(Badge, { tone: 'violet', children });
    expect(container.querySelector('[data-status]')).toBeNull();
    expect(container.querySelector('svg')).toBeNull();
  });
});
