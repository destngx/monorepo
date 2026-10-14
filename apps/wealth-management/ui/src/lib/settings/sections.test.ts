import type { EngineSetting } from '$lib/api/types';
import { describe, expect, it } from 'vitest';
import { choiceOptions, sectionFromUrl, sectionsWithChanges, splitAdvanced } from './sections';

const setting = (key: string, group: string, extra: Partial<EngineSetting> = {}): EngineSetting => ({
  key,
  group,
  label: key,
  description: '',
  kind: 'text',
  secret: false,
  isSet: false,
  ...extra,
});

describe('sectionFromUrl', () => {
  const at = (query: string) => new URL(`http://127.0.0.1:5173/ui/settings${query}`);

  it('opens the section named in the address', () => {
    expect(sectionFromUrl(at('?tab=market'))).toBe('market');
  });

  it('falls back to General for a missing or unknown section', () => {
    expect(sectionFromUrl(at(''))).toBe('general');
    expect(sectionFromUrl(at('?tab=nope'))).toBe('general');
  });

  it('returns from a Google sign-in to the section it started from', () => {
    expect(sectionFromUrl(at('?google=connected&purpose=sheets'))).toBe('sheets');
    expect(sectionFromUrl(at('?google=error&purpose=test'))).toBe('general');
  });
});

describe('splitAdvanced', () => {
  it('keeps the order and folds advanced settings away', () => {
    const { main, advanced } = splitAdvanced([
      setting('A', 'ai'),
      setting('B', 'ai', { advanced: true }),
      setting('C', 'ai'),
    ]);
    expect(main.map((s) => s.key)).toEqual(['A', 'C']);
    expect(advanced.map((s) => s.key)).toEqual(['B']);
  });
});

describe('sectionsWithChanges', () => {
  it('marks every section holding a changed setting', () => {
    const settings = [setting('AI_MODEL', 'ai'), setting('TAVILY_API_KEY', 'search'), setting('LOG_LEVEL', 'general')];
    expect([...sectionsWithChanges(settings, { TAVILY_API_KEY: 'k' })]).toEqual(['ai']);
    expect(sectionsWithChanges(settings, {}).size).toBe(0);
  });
});

describe('choiceOptions', () => {
  it('lists the options and keeps a stored value that is no longer offered', () => {
    expect(choiceOptions(['a', 'b'], 'b')).toEqual(['a', 'b']);
    expect(choiceOptions(['a', 'b'], 'legacy')).toEqual(['legacy', 'a', 'b']);
    expect(choiceOptions(['a'], '')).toEqual(['a']);
  });
});
