import { describe, expect, it } from 'vitest';
import { STATUS, STATUS_TONES, usageTone } from './status';

describe('status scale', () => {
  it('Given every tone When shown Then it has its own icon or label, so it never reads by colour alone', () => {
    const marks = STATUS_TONES.map((tone) => `${STATUS[tone].icon}:${STATUS[tone].label}`);
    expect(new Set(marks).size).toBe(STATUS_TONES.length);
  });

  it("Given the classes When an app changes its palette Then they follow, because they use the theme's role scales", () => {
    const classKeys = ['ink', 'edge', 'box', 'badge', 'fill', 'wash'] as const;
    const classes = STATUS_TONES.flatMap((tone) => classKeys.flatMap((key) => STATUS[tone][key].split(' ')));
    for (const name of classes) {
      expect(name, name).toMatch(/-(good|warn|danger|info|ink)-\d+(\/\d+)?$/);
    }
  });
});

describe('usageTone', () => {
  it('Given a share of a limit When rated Then it is good, then warning from 80%, then serious from 100%', () => {
    expect(usageTone(79)).toBe('good');
    expect(usageTone(80)).toBe('warning');
    expect(usageTone(100)).toBe('serious');
    expect(usageTone(60, 50)).toBe('warning');
  });
});
