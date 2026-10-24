import { describe, expect, it } from 'vitest';
import { agentName, groupAgents, statusOf, summary, type Agent } from './agents';

function agent(over: Partial<Agent>): Agent {
  return {
    paneId: 'w1:p1',
    workspaceId: 'w1',
    workspace: 'example',
    tabId: 'w1:t1',
    kind: 'claude',
    title: 'Fix the form',
    status: 'idle',
    focused: false,
    dir: '/home/someone/example',
    ...over,
  };
}

describe('statusOf', () => {
  it('names each state for a person, with a tone that is not colour alone', () => {
    expect(statusOf('blocked')).toEqual({ tone: 'warning', label: 'Needs you', icon: 'alert' });
    expect(statusOf('working')).toEqual({ tone: 'info', label: 'Working', icon: 'refresh' });
    expect(statusOf('done')).toEqual({ tone: 'good', label: 'Done', icon: 'check' });
    expect(statusOf('idle')).toEqual({ tone: 'neutral', label: 'Idle', icon: 'clock' });
    expect(statusOf('unknown')).toEqual({ tone: 'neutral', label: 'Unknown', icon: 'info' });
  });
});

describe('agentName', () => {
  it('uses the title, then the program, then the pane id', () => {
    expect(agentName(agent({ title: 'Fix the form' }))).toBe('Fix the form');
    expect(agentName(agent({ title: ' ', kind: 'codex' }))).toBe('codex');
    expect(agentName(agent({ title: '', kind: '', paneId: 'w2:p3' }))).toBe('w2:p3');
  });
});

describe('groupAgents', () => {
  it('puts agents that need you first, then working, done and idle, by name within each', () => {
    const groups = groupAgents([
      agent({ paneId: 'a:1', title: 'B idle', status: 'idle' }),
      agent({ paneId: 'a:2', title: 'Working', status: 'working' }),
      agent({ paneId: 'a:3', title: 'A idle', status: 'idle' }),
      agent({ paneId: 'a:4', title: 'Blocked', status: 'blocked' }),
      agent({ paneId: 'a:5', title: 'Odd', status: 'unknown' }),
    ]);
    expect(groups.map((g) => [g.status, g.agents.map((a) => a.title)])).toEqual([
      ['blocked', ['Blocked']],
      ['working', ['Working']],
      ['idle', ['A idle', 'B idle']],
      ['unknown', ['Odd']],
    ]);
  });
});

describe('summary', () => {
  it('counts what needs attention first and skips empty states', () => {
    const list = [
      agent({ status: 'blocked' }),
      agent({ status: 'blocked' }),
      agent({ status: 'working' }),
      agent({ status: 'idle' }),
    ];
    expect(summary(list)).toBe('2 need you · 1 working · 1 idle');
    expect(summary([agent({ status: 'blocked' })])).toBe('1 needs you');
    expect(summary([])).toBe('No agents running');
  });
});
