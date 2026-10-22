import { describe, expect, it } from 'vitest';
import { DEFAULT_SWITCH_INPUT, compareStablecoinSwitch, parseSwitchInput, type SwitchInput } from './stablecoinSwitch';

const input = (overrides: Partial<SwitchInput> = {}): SwitchInput => ({
  principal: 1000,
  currentRate: 5,
  newRate: 10,
  days: 73,
  entryFee: { kind: 'percent', value: 0.1 },
  exitFee: { kind: 'percent', value: 0.1 },
  ...overrides,
});

describe('stablecoin switch: yield vs fees (WM-INV-30)', () => {
  it('Given percent fees When comparing Then staying earns simple interest and switching pays both fees around the new rate', () => {
    const result = compareStablecoinSwitch(input());

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    // 1000 x (1 + 5% x 73/365)
    expect(result.stay.value).toBeCloseTo(1010, 6);
    expect(result.stay.interest).toBeCloseTo(10, 6);
    // 1000 x (1 - 0.1%) x (1 + 10% x 73/365) x (1 - 0.1%)
    expect(result.switch.value).toBeCloseTo(1017.96102, 5);
    expect(result.switch.interest).toBeCloseTo(19.98, 6);
    expect(result.switch.entryFee).toBeCloseTo(1, 6);
    expect(result.switch.exitFee).toBeCloseTo(1.01898, 5);
    expect(result.net).toBeCloseTo(7.96102, 5);
    expect(result.decision).toBe('switch');
  });

  it('Given flat fees When comparing Then each fee comes off as an amount', () => {
    const result = compareStablecoinSwitch(
      input({ entryFee: { kind: 'amount', value: 2 }, exitFee: { kind: 'amount', value: 1 } }),
    );

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    // (1000 - 2) x 1.02 - 1
    expect(result.switch.value).toBeCloseTo(1016.96, 6);
    expect(result.switch.entryFee).toBe(2);
    expect(result.switch.exitFee).toBe(1);
    expect(result.net).toBeCloseTo(6.96, 6);
  });

  it('Given fees larger than the extra interest When comparing Then the decision is to stay', () => {
    const result = compareStablecoinSwitch(input({ days: 5 }));

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.net).toBeLessThan(0);
    expect(result.decision).toBe('stay');
  });

  it('Given an exact tie When comparing Then the decision is to stay, as in the sheet', () => {
    const result = compareStablecoinSwitch(
      input({ newRate: 5, entryFee: { kind: 'percent', value: 0 }, exitFee: { kind: 'amount', value: 0 } }),
    );

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.net).toBe(0);
    expect(result.decision).toBe('stay');
  });

  it('Given fees and a higher rate When asking how long to hold Then break-even is the first whole day switching is ahead', () => {
    const result = compareStablecoinSwitch(input());

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.breakEvenDays).toBe(15);
    const before = compareStablecoinSwitch(input({ days: 14 }));
    const at = compareStablecoinSwitch(input({ days: 15 }));
    expect(before.ok && before.net).toBeLessThan(0);
    expect(at.ok && at.net).toBeGreaterThan(0);
  });

  it('Given no fees and a higher rate When asking how long to hold Then switching is ahead from the first day', () => {
    const result = compareStablecoinSwitch(
      input({ entryFee: { kind: 'percent', value: 0 }, exitFee: { kind: 'amount', value: 0 } }),
    );

    expect(result.ok && result.breakEvenDays).toBe(1);
  });

  it('Given a new rate no better than the current one When asking how long to hold Then switching never pays off', () => {
    const result = compareStablecoinSwitch(input({ newRate: 5 }));

    expect(result.ok).toBe(true);
    if (!result.ok) return;
    expect(result.breakEvenDays).toBeNull();
    expect(result.decision).toBe('stay');
  });

  it.each<[string, Partial<SwitchInput>, string]>([
    ['no principal', { principal: 0 }, 'principal'],
    ['a negative rate', { currentRate: -1 }, 'currentRate'],
    ['a negative new rate', { newRate: -1 }, 'newRate'],
    ['no days', { days: 0 }, 'days'],
    ['part of a day', { days: 1.5 }, 'days'],
    ['a fee of 100%', { entryFee: { kind: 'percent', value: 100 } }, 'entryFee'],
    ['a negative fee', { exitFee: { kind: 'amount', value: -1 } }, 'exitFee'],
    ['a flat entry fee eating the principal', { entryFee: { kind: 'amount', value: 1000 } }, 'entryFee'],
    ['a value that is not a number', { principal: Number.NaN }, 'principal'],
  ])('Given %s When comparing Then it names the bad field instead of a result', (_, overrides, field) => {
    const result = compareStablecoinSwitch(input(overrides));

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(Object.keys(result.errors)).toContain(field);
  });

  it('Given a flat exit fee above what switching ends with When comparing Then it says so', () => {
    const result = compareStablecoinSwitch(input({ exitFee: { kind: 'amount', value: 5000 } }));

    expect(result.ok).toBe(false);
    if (result.ok) return;
    expect(result.errors.exitFee).toMatch(/more than/);
  });
});

describe('stablecoin switch: saved input', () => {
  it('Given a saved input When reading it back Then every field is restored', () => {
    const saved = input({ entryFee: { kind: 'amount', value: 0.2 } });

    expect(parseSwitchInput(JSON.stringify(saved))).toEqual(saved);
  });

  it.each([
    ['nothing saved', null],
    ['broken JSON', '{'],
    ['another shape', '[1, 2]'],
  ])('Given %s When reading it back Then the defaults are used', (_, raw) => {
    expect(parseSwitchInput(raw)).toEqual(DEFAULT_SWITCH_INPUT);
  });

  it('Given an older save missing a field When reading it back Then only that field takes its default', () => {
    const parsed = parseSwitchInput(JSON.stringify({ principal: 250, exitFee: { kind: 'bogus', value: 1 } }));

    expect(parsed.principal).toBe(250);
    expect(parsed.exitFee).toEqual(DEFAULT_SWITCH_INPUT.exitFee);
    expect(parsed.days).toBe(DEFAULT_SWITCH_INPUT.days);
  });
});
