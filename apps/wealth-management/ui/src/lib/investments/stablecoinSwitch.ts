// Stablecoin switch (WM-INV-30, D43): is moving a stablecoin balance to a
// place that pays a higher rate worth the conversion fees? This is the
// sheet's former "Stablecoin Yield vs Fees Calculator" tab. Nothing comes
// from the sheet or the engine, so it runs here, on the user's figures.
//
// Interest is simple, rate x days / 365, as in the sheet: over the weeks a
// switch is weighed for, compounding changes nothing that matters. Staying
// grows the whole principal at the current rate. Switching pays the entry
// fee first, grows what is left at the new rate, then pays the exit fee on
// the way out. A fee is a percent of what is converted, or a flat amount.

export type FeeKind = 'percent' | 'amount';

export interface SwitchFee {
  kind: FeeKind;
  /** Percent (0.1 = 0.1%) or an amount in the stablecoin. */
  value: number;
}

export interface SwitchInput {
  principal: number;
  /** Percent a year paid where the money is now. */
  currentRate: number;
  /** Percent a year paid by the new place. */
  newRate: number;
  /** How long the money is held, in whole days. */
  days: number;
  entryFee: SwitchFee;
  exitFee: SwitchFee;
}

export type SwitchField = keyof SwitchInput;

export interface SwitchResult {
  ok: true;
  stay: { value: number; interest: number };
  switch: { value: number; interest: number; entryFee: number; exitFee: number };
  /** Switch value - stay value: what switching gains (negative: loses). */
  net: number;
  /** Switch only when it ends ahead; a tie stays, as in the sheet. */
  decision: 'switch' | 'stay';
  /** The first whole day of holding after which switching is ahead; null when it never is. */
  breakEvenDays: number | null;
}

export type SwitchComparison = SwitchResult | { ok: false; errors: Partial<Record<SwitchField, string>> };

/** Starting figures until the user saves their own: an example, not anyone's balance. */
export const DEFAULT_SWITCH_INPUT: SwitchInput = {
  principal: 1000,
  currentRate: 5,
  newRate: 10,
  days: 30,
  entryFee: { kind: 'percent', value: 0.1 },
  exitFee: { kind: 'percent', value: 0.1 },
};

export const SWITCH_STORAGE_KEY = 'wm.stablecoinSwitch';

const finite = (value: number) => typeof value === 'number' && Number.isFinite(value);

/** What is left after paying a fee on an amount. */
function afterFee(amount: number, fee: SwitchFee): number {
  return fee.kind === 'percent' ? amount * (1 - fee.value / 100) : amount - fee.value;
}

function feeError(fee: SwitchFee): string | undefined {
  if (!finite(fee.value) || fee.value < 0) return 'Enter a fee of 0 or more.';
  if (fee.kind === 'percent' && fee.value >= 100) return 'A percent fee must be under 100%.';
  return undefined;
}

export function compareStablecoinSwitch(input: SwitchInput): SwitchComparison {
  const { principal, currentRate, newRate, days, entryFee, exitFee } = input;
  const errors: Partial<Record<SwitchField, string>> = {};
  if (!finite(principal) || principal <= 0) errors.principal = 'Enter an amount above 0.';
  if (!finite(currentRate) || currentRate < 0) errors.currentRate = 'Enter a rate of 0% or more.';
  if (!finite(newRate) || newRate < 0) errors.newRate = 'Enter a rate of 0% or more.';
  if (!finite(days) || days < 1 || !Number.isInteger(days)) errors.days = 'Enter a whole number of days, 1 or more.';
  const entryError = feeError(entryFee);
  if (entryError) errors.entryFee = entryError;
  const exitError = feeError(exitFee);
  if (exitError) errors.exitFee = exitError;
  if (Object.keys(errors).length) return { ok: false, errors };

  const invested = afterFee(principal, entryFee);
  if (invested <= 0) return { ok: false, errors: { entryFee: 'The entry fee takes the whole amount.' } };
  const grown = invested * (1 + ((newRate / 100) * days) / 365);
  if (exitFee.kind === 'amount' && exitFee.value > grown) {
    return { ok: false, errors: { exitFee: 'The exit fee is more than switching ends with.' } };
  }
  const stay = principal * (1 + ((currentRate / 100) * days) / 365);
  const switched = afterFee(grown, exitFee);
  const net = switched - stay;

  // Switch minus stay is a straight line in days: start + slope x days.
  const exitShare = exitFee.kind === 'percent' ? 1 - exitFee.value / 100 : 1;
  const start = afterFee(invested, exitFee) - principal;
  const slope = (invested * (newRate / 100) * exitShare - principal * (currentRate / 100)) / 365;
  // Fees are never negative, so start <= 0: switching gets ahead only when the slope is up.
  const breakEvenDays = slope > 0 ? Math.floor(-start / slope) + 1 : null;

  return {
    ok: true,
    stay: { value: stay, interest: stay - principal },
    switch: { value: switched, interest: grown - invested, entryFee: principal - invested, exitFee: grown - switched },
    net,
    decision: net > 0 ? 'switch' : 'stay',
    breakEvenDays,
  };
}

function parseFee(raw: unknown, fallback: SwitchFee): SwitchFee {
  if (typeof raw !== 'object' || raw === null) return fallback;
  const { kind, value } = raw as Record<string, unknown>;
  if ((kind !== 'percent' && kind !== 'amount') || typeof value !== 'number' || !Number.isFinite(value))
    return fallback;
  return { kind, value };
}

/** The saved input, field by field; anything missing or malformed takes its default. */
export function parseSwitchInput(raw: string | null | undefined): SwitchInput {
  let saved: unknown;
  try {
    saved = raw ? JSON.parse(raw) : null;
  } catch {
    return DEFAULT_SWITCH_INPUT;
  }
  if (typeof saved !== 'object' || saved === null || Array.isArray(saved)) return DEFAULT_SWITCH_INPUT;
  const record = saved as Record<string, unknown>;
  const number = (key: 'principal' | 'currentRate' | 'newRate' | 'days') => {
    const value = record[key];
    return typeof value === 'number' && Number.isFinite(value) ? value : DEFAULT_SWITCH_INPUT[key];
  };
  return {
    principal: number('principal'),
    currentRate: number('currentRate'),
    newRate: number('newRate'),
    days: number('days'),
    entryFee: parseFee(record.entryFee, DEFAULT_SWITCH_INPUT.entryFee),
    exitFee: parseFee(record.exitFee, DEFAULT_SWITCH_INPUT.exitFee),
  };
}
