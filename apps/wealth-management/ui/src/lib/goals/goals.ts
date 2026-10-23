import type { GoalInput, GoalPace, GoalPlan, GoalProgress, GoalStatus, PlanVerdict } from '$lib/api/types';
import { formatMonth } from '$lib/format';
import type { StatusTone } from '$lib/status';

/** A goal's pace as a status badge: always an icon and a label, never colour alone. */
export function paceStatus(pace: GoalPace): { tone: StatusTone; label: string } {
  switch (pace) {
    case 'done':
      return { tone: 'good', label: 'Done' };
    case 'funded':
      return { tone: 'good', label: 'Funded' };
    case 'on_track':
      return { tone: 'good', label: 'On track' };
    case 'behind':
      return { tone: 'warning', label: 'Behind' };
    case 'no_deadline':
      return { tone: 'info', label: 'No deadline' };
    case 'paused':
      return { tone: 'neutral', label: 'Paused' };
  }
}

/** The plan check verdict as a status. */
export function verdictStatus(verdict: PlanVerdict): { tone: StatusTone; label: string } {
  switch (verdict) {
    case 'fits':
      return { tone: 'good', label: 'Your savings cover your dated goals' };
    case 'short':
      return { tone: 'serious', label: 'Your savings fall short of your dated goals' };
    case 'no_deadlines':
      return { tone: 'info', label: 'No goal has a target date' };
  }
}

/** "8.7 years" / "1 year"; "-" when unknown. */
export function formatYears(years: number | null | undefined): string {
  if (years === null || years === undefined) return '-';
  return years === 1 ? '1 year' : `${years} years`;
}

/** When a goal is projected to finish, in words. */
export function finishLabel(goal: GoalProgress): string {
  if (goal.pace === 'done') return 'Done';
  if (goal.pace === 'paused') return 'Paused';
  if (goal.projectedFinish === null) return 'Not reachable at this pace';
  if (goal.monthsToReach === 0) return 'Funded now';
  return formatMonth(goal.projectedFinish);
}

/**
 * The target date hint. The engine counts a month's savings as all there by the
 * end of that month, so money needed early in a month belongs to the month before.
 */
export function goalDateHint(targetDate: string): string {
  if (!targetDate) return 'Blank means no deadline; the goal shows its projected finish only.';
  const [year, month] = targetDate.split('-').map(Number);
  const before = month === 1 ? `${year - 1}-12` : `${year}-${String(month - 1).padStart(2, '0')}`;
  return `Saved by the end of ${formatMonth(targetDate)}. Need it before ${formatMonth(targetDate, true)}'s wage arrives? Pick ${formatMonth(before)}.`;
}

/** Active goals first, then paused, then done, each in priority order (the engine's order). */
export function groupGoals(plan: GoalPlan): { active: GoalProgress[]; paused: GoalProgress[]; done: GoalProgress[] } {
  return {
    active: plan.goals.filter((goal) => goal.status === 'active'),
    paused: plan.goals.filter((goal) => goal.status === 'paused'),
    done: plan.goals.filter((goal) => goal.status === 'done'),
  };
}

/** Moves one id up (-1) or down (+1); out-of-range moves return the same order. */
export function moveId(ids: string[], index: number, delta: -1 | 1): string[] {
  const target = index + delta;
  if (index < 0 || index >= ids.length || target < 0 || target >= ids.length) return ids;
  const next = [...ids];
  [next[index], next[target]] = [next[target], next[index]];
  return next;
}

/**
 * Mirrors the engine's emergency fund rule (name or note mentions emergency, khẩn cấp or dự phòng,
 * accents ignored) so the form knows the target may stay blank; the engine decides.
 */
export function looksLikeEmergencyFund(name: string, note = ''): boolean {
  const folded = `${name} ${note}`
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .replace(/đ/gi, 'd')
    .toLowerCase();
  return /\b(emergency|khan cap|du phong)/.test(folded);
}

/** The goal form; amounts are null while blank. */
export interface GoalForm {
  name: string;
  target: number | null;
  targetDate: string;
  priority: number | null;
  startingAmount: number | null;
  status: GoalStatus;
  note: string;
}

export function emptyGoalForm(): GoalForm {
  return { name: '', target: null, targetDate: '', priority: null, startingAmount: null, status: 'active', note: '' };
}

export function goalFormFor(goal: GoalProgress): GoalForm {
  return {
    name: goal.name,
    target: goal.target,
    targetDate: goal.targetDate ?? '',
    priority: goal.priority || null,
    startingAmount: goal.startingAmount || null,
    status: goal.status,
    note: goal.note ?? '',
  };
}

/** Builds the API input, or the first problem to show. The engine validates again. */
export function goalInputFrom(form: GoalForm, emergencyFund: boolean): { input?: GoalInput; error?: string } {
  const name = form.name.trim();
  if (!name) return { error: 'Name is required.' };
  const target = form.target ?? undefined;
  if (target !== undefined && target <= 0) return { error: 'Target must be more than 0.' };
  if (target === undefined && !emergencyFund)
    return { error: 'Target is required (only the emergency fund computes it).' };
  if (form.targetDate && !/^\d{4}-\d{2}$/.test(form.targetDate)) return { error: 'Target date must be a month.' };
  if (form.startingAmount !== null && form.startingAmount < 0) return { error: 'Starting amount cannot be negative.' };
  return {
    input: {
      name,
      target,
      targetDate: form.targetDate || undefined,
      priority: form.priority ?? undefined,
      startingAmount: form.startingAmount ?? undefined,
      status: form.status,
      note: form.note.trim() || undefined,
    },
  };
}
