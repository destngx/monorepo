import { describe, expect, it } from 'vitest';
import { Folds } from './folds.svelte';

describe('Folds', () => {
  it('starts folded, opens one page and folds or opens them all', () => {
    const f = new Folds();
    expect(f.isOpen('a')).toBe(false);
    expect(f.allFolded(['a', 'b'])).toBe(true);
    f.toggle('a');
    expect(f.isOpen('a')).toBe(true);
    expect(f.allFolded(['a', 'b'])).toBe(false);

    f.toggleAll(['a', 'b']);
    expect(f.allFolded(['a', 'b'])).toBe(true);
    f.toggleAll(['a', 'b']);
    expect(f.isOpen('a') && f.isOpen('b')).toBe(true);

    f.toggle('b');
    f.open('b');
    expect(f.isOpen('b')).toBe(true);
    expect(f.allFolded([])).toBe(false);
  });

  it('can start open', () => {
    const f = new Folds({ startOpen: true });
    expect(f.isOpen('a')).toBe(true);
    f.toggle('a');
    expect(f.isOpen('a')).toBe(false);
  });

  it('can start open only the pages it is told to', () => {
    const f = new Folds({ startOpen: (id) => id === 'check' });
    expect(f.isOpen('check')).toBe(true);
    expect(f.isOpen('a')).toBe(false);
    f.toggle('check');
    expect(f.isOpen('check')).toBe(false);
  });
});
