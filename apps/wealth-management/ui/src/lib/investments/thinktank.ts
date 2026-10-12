import type { InvestmentCommand, InvestmentEvent, InvestmentPhase, NewsReport, ThinkTankContext } from '$lib/api/types';

// Think tank run state (WM-AI-9, WM-INV-18). The engine streams phase
// events; reduce() folds them into one immutable state the page renders.

export type PhaseStatus = 'pending' | 'running' | 'ok' | 'error' | 'stopped';

export interface PhaseState {
  status: PhaseStatus;
  /** Why the main prompt failed when the smaller fallback prompt was used. */
  fallback?: string;
  error?: string;
}

export interface ThinkTankMessage {
  name: string;
  content: string;
  model?: string;
}

export interface ThinkTankActions {
  name: string;
  commands: InvestmentCommand[];
  model?: string;
}

export type RunStatus = 'idle' | 'running' | 'done' | 'error' | 'stopped';

export interface ThinkTankState {
  status: RunStatus;
  phases: Record<InvestmentPhase, PhaseState>;
  context: ThinkTankContext | null;
  thinkTank: ThinkTankMessage | null;
  synthesis: ThinkTankMessage | null;
  actions: ThinkTankActions | null;
  error: string;
  /** News reports sent with this run. */
  news: NewsReport[];
}

export const PHASES: { key: InvestmentPhase; label: string }[] = [
  { key: 'context', label: 'Context' },
  { key: 'think-tank', label: 'Think tank' },
  { key: 'synthesis', label: 'Synthesis' },
  { key: 'actions', label: 'Actions' },
];

const pendingPhases = (): Record<InvestmentPhase, PhaseState> => ({
  context: { status: 'pending' },
  'think-tank': { status: 'pending' },
  synthesis: { status: 'pending' },
  actions: { status: 'pending' },
});

export function initialState(): ThinkTankState {
  return {
    status: 'idle',
    phases: pendingPhases(),
    context: null,
    thinkTank: null,
    synthesis: null,
    actions: null,
    error: '',
    news: [],
  };
}

/** A fresh running state for a new run. */
export function startState(news: NewsReport[] = []): ThinkTankState {
  return { ...initialState(), status: 'running', news };
}

function withPhase(state: ThinkTankState, phase: InvestmentPhase, next: PhaseState): ThinkTankState {
  return { ...state, phases: { ...state.phases, [phase]: next } };
}

/** Marks every running phase with the given status. */
function settleRunning(
  phases: Record<InvestmentPhase, PhaseState>,
  status: PhaseStatus,
  error?: string,
): Record<InvestmentPhase, PhaseState> {
  const next = { ...phases };
  for (const { key } of PHASES) {
    if (next[key].status === 'running') next[key] = { ...next[key], status, ...(error ? { error } : {}) };
  }
  return next;
}

/** Applies one streamed event. Pure: the input state is never mutated. */
export function reduce(state: ThinkTankState, event: InvestmentEvent): ThinkTankState {
  switch (event.type) {
    case 'phase': {
      const current = state.phases[event.phase];
      if (!current) return state;
      switch (event.status) {
        case 'running':
          return withPhase(state, event.phase, { status: 'running' });
        case 'fallback':
          return withPhase(state, event.phase, {
            ...current,
            status: 'running',
            fallback: event.message || 'The main prompt failed.',
          });
        case 'ok':
          return withPhase(state, event.phase, { ...current, status: 'ok' });
        case 'error':
          return withPhase(state, event.phase, {
            ...current,
            status: 'error',
            error: event.message || 'The phase failed.',
          });
      }
      return state;
    }
    case 'context':
      return { ...state, context: event.context };
    case 'message': {
      const message = { name: event.name, content: event.content, model: event.model };
      if (event.phase === 'think-tank') return { ...state, thinkTank: message };
      if (event.phase === 'synthesis') return { ...state, synthesis: message };
      return state;
    }
    case 'actions':
      return { ...state, actions: { name: event.name, commands: event.commands ?? [], model: event.model } };
    case 'done':
      return { ...state, status: 'done' };
    case 'error':
      return fail(state, event.message);
  }
  return state;
}

/** Ends the run with an error: a stream error event, or a request that failed before streaming. */
export function fail(state: ThinkTankState, message: string): ThinkTankState {
  const error = message || 'The analysis failed.';
  return { ...state, status: 'error', error, phases: settleRunning(state.phases, 'error', error) };
}

/** Ends the run because the user stopped it. */
export function stop(state: ThinkTankState): ThinkTankState {
  if (state.status !== 'running') return state;
  return { ...state, status: 'stopped', phases: settleRunning(state.phases, 'stopped') };
}

/** Called when the stream closes: a run still running never got its "done". */
export function finish(state: ThinkTankState): ThinkTankState {
  if (state.status !== 'running') return state;
  return fail(state, 'The engine ended the analysis before it finished.');
}

/** Phase whose error the page shows inline, if any. */
export function failedPhase(state: ThinkTankState): InvestmentPhase | null {
  return PHASES.find(({ key }) => state.phases[key].status === 'error')?.key ?? null;
}

const truncate = (text: string, max: number) =>
  text.length <= max ? text : `${text.slice(0, Math.max(0, max - 1)).trimEnd()}…`;

/**
 * Compact page context for the follow-up chat: the commands first (they are
 * short and most useful), then as much of the synthesis as fits.
 */
export function chatSummary(state: ThinkTankState, max = 4000): string {
  const parts: string[] = ['Investment think tank result (AI analysis of the portfolio, market pulse and news).'];
  const news = state.context?.news ?? state.news;
  if (news.length) {
    parts.push(`News bias: ${news.map((r) => `${r.topic}: ${r.bias}`).join('; ')}.`);
  }
  const commands = state.actions?.commands ?? [];
  if (commands.length) {
    parts.push('Recommended commands:\n' + commands.map((c) => `- ${c.action} ${c.target}: ${c.rationale}`).join('\n'));
  }
  let summary = parts.join('\n\n');
  if (state.synthesis?.content) {
    const header = '\n\nSynthesis:\n';
    const room = max - summary.length - header.length;
    if (room > 200) summary += header + truncate(state.synthesis.content, room);
  }
  return truncate(summary, max);
}
