import { describe, expect, it } from 'vitest';
import type { InvestmentEvent, NewsReport, ThinkTankContext } from '$lib/api/types';
import {
  chatSummary,
  failedPhase,
  fail,
  finish,
  initialState,
  reduce,
  startState,
  stop,
  type ThinkTankState,
} from './thinktank';

const context: ThinkTankContext = {
  accounts: [],
  moneyLent: { owedToYou: 0, youOwe: 0, open: [] },
  recentTransactions: [],
  budget: { income: 30_000_000, expenses: 20_000_000 },
  searches: null,
  searchContext: 'N/A: not available',
};

const news: NewsReport = {
  topic: 'Gold',
  query: 'gold news',
  bias: 'BULLISH',
  counts: { positive: 2, negative: 0, neutral: 1 },
  articles: [],
  provider: 'tavily',
  generatedAt: '2026-10-13T08:00:00Z',
};

const run = (events: InvestmentEvent[], from: ThinkTankState = startState()) => events.reduce(reduce, from);

const happyPath: InvestmentEvent[] = [
  { type: 'phase', phase: 'context', status: 'running' },
  { type: 'context', context },
  { type: 'phase', phase: 'context', status: 'ok' },
  { type: 'phase', phase: 'think-tank', status: 'running' },
  { type: 'phase', phase: 'think-tank', status: 'ok' },
  { type: 'message', phase: 'think-tank', name: 'Think tank', content: '## Debate', model: 'gpt-a' },
  { type: 'phase', phase: 'synthesis', status: 'running' },
  { type: 'phase', phase: 'synthesis', status: 'ok' },
  { type: 'message', phase: 'synthesis', name: 'Synthesis', content: 'Hold cash.', model: 'gpt-b' },
  { type: 'phase', phase: 'actions', status: 'running' },
  { type: 'phase', phase: 'actions', status: 'ok' },
  {
    type: 'actions',
    phase: 'actions',
    name: 'Actions',
    commands: [{ action: 'ALLOCATE', target: 'VN30 ETF', rationale: 'Cheap.' }],
    model: 'gpt-c',
  },
  { type: 'done' },
];

describe('think tank reducer', () => {
  it('starts idle with every phase pending', () => {
    const state = initialState();
    expect(state.status).toBe('idle');
    expect(Object.values(state.phases).every((p) => p.status === 'pending')).toBe(true);
  });

  it('given a full stream, ends done with every output', () => {
    const state = run(happyPath);
    expect(state.status).toBe('done');
    expect(Object.values(state.phases).map((p) => p.status)).toEqual(['ok', 'ok', 'ok', 'ok']);
    expect(state.context).toEqual(context);
    expect(state.thinkTank).toEqual({ name: 'Think tank', content: '## Debate', model: 'gpt-a' });
    expect(state.synthesis?.model).toBe('gpt-b');
    expect(state.actions?.commands).toHaveLength(1);
    expect(state.error).toBe('');
    expect(failedPhase(state)).toBeNull();
  });

  it('does not mutate the previous state', () => {
    const before = startState();
    const after = reduce(before, { type: 'phase', phase: 'context', status: 'running' });
    expect(before.phases.context.status).toBe('pending');
    expect(after.phases.context.status).toBe('running');
  });

  it('given a fallback, keeps the reason after the phase succeeds', () => {
    const state = run([
      { type: 'phase', phase: 'synthesis', status: 'running' },
      { type: 'phase', phase: 'synthesis', status: 'fallback', message: 'reply is empty' },
    ]);
    expect(state.phases.synthesis).toEqual({ status: 'running', fallback: 'reply is empty' });
    const ok = reduce(state, { type: 'phase', phase: 'synthesis', status: 'ok' });
    expect(ok.phases.synthesis).toEqual({ status: 'ok', fallback: 'reply is empty' });
  });

  it('given a phase error then an error event, ends in error on that phase', () => {
    const state = run([
      ...happyPath.slice(0, 4),
      { type: 'phase', phase: 'think-tank', status: 'fallback', message: 'timeout' },
      { type: 'phase', phase: 'think-tank', status: 'error', message: 'the model call failed' },
      { type: 'error', message: 'think-tank phase: the model call failed' },
    ]);
    expect(state.status).toBe('error');
    expect(state.error).toBe('think-tank phase: the model call failed');
    expect(state.phases['think-tank']).toEqual({
      status: 'error',
      fallback: 'timeout',
      error: 'the model call failed',
    });
    expect(state.phases.synthesis.status).toBe('pending');
    expect(failedPhase(state)).toBe('think-tank');
  });

  it('given an error event without a phase error, fails the running phase', () => {
    const state = run([
      { type: 'phase', phase: 'context', status: 'running' },
      { type: 'error', message: 'boom' },
    ]);
    expect(state.phases.context).toEqual({ status: 'error', error: 'boom' });
    expect(state.status).toBe('error');
  });

  it('given a request failure before streaming, records the error', () => {
    const state = fail(startState(), 'AI is not configured');
    expect(state.status).toBe('error');
    expect(state.error).toBe('AI is not configured');
  });

  it('given a stream that closes without done, fails the run', () => {
    const state = finish(run(happyPath.slice(0, 5)));
    expect(state.status).toBe('error');
    expect(state.error).toMatch(/before it finished/);
    expect(finish(run(happyPath)).status).toBe('done');
  });

  it('given a stop, marks running phases stopped and keeps finished output', () => {
    const state = stop(run(happyPath.slice(0, 7)));
    expect(state.status).toBe('stopped');
    expect(state.phases['think-tank'].status).toBe('ok');
    expect(state.phases.synthesis.status).toBe('stopped');
    expect(state.thinkTank?.content).toBe('## Debate');
    expect(stop(run(happyPath)).status).toBe('done');
  });
});

describe('chat summary', () => {
  it('lists news bias, commands and the synthesis', () => {
    const summary = chatSummary(run(happyPath, startState([news])));
    expect(summary).toContain('Gold: BULLISH');
    expect(summary).toContain('- ALLOCATE VN30 ETF: Cheap.');
    expect(summary).toContain('Synthesis:\nHold cash.');
  });

  it('stays under the limit, truncating the synthesis first', () => {
    const long = run([
      ...happyPath.slice(0, 8),
      { type: 'message', phase: 'synthesis', name: 'Synthesis', content: 'x'.repeat(10_000) },
      ...happyPath.slice(9),
    ]);
    const summary = chatSummary(long, 4000);
    expect(summary.length).toBeLessThanOrEqual(4000);
    expect(summary).toContain('ALLOCATE VN30 ETF');
    expect(summary.endsWith('…')).toBe(true);
  });
});
