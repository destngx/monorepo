import type { IconName, StatusTone } from '@libs/ui-kit';

/** herdr's agent states. */
export type AgentStatus = 'idle' | 'working' | 'blocked' | 'done' | 'unknown';

/** One agent pane, as the engine's /api/agents returns it. */
export interface Agent {
  paneId: string;
  workspaceId: string;
  workspace: string;
  tabId: string;
  /** The agent program herdr detected: claude, codex, and so on. */
  kind: string;
  title: string;
  status: AgentStatus;
  focused: boolean;
  dir: string;
}

export interface StatusInfo {
  tone: StatusTone;
  label: string;
  icon: IconName;
}

const STATUSES: Record<AgentStatus, StatusInfo> = {
  blocked: { tone: 'warning', label: 'Needs you', icon: 'alert' },
  working: { tone: 'info', label: 'Working', icon: 'refresh' },
  done: { tone: 'good', label: 'Done', icon: 'check' },
  idle: { tone: 'neutral', label: 'Idle', icon: 'clock' },
  unknown: { tone: 'neutral', label: 'Unknown', icon: 'info' },
};

/** The order the list shows states in: what needs you first. */
export const STATUS_ORDER: readonly AgentStatus[] = ['blocked', 'working', 'done', 'idle', 'unknown'];

export function statusOf(status: AgentStatus): StatusInfo {
  return STATUSES[status] ?? STATUSES.unknown;
}

/** The name a person knows the agent by: its terminal title, else the program, else the pane id. */
export function agentName(agent: Agent): string {
  return agent.title.trim() || agent.kind.trim() || agent.paneId;
}

export interface AgentGroup {
  status: AgentStatus;
  agents: Agent[];
}

/** Agents grouped by state in STATUS_ORDER, sorted by name; empty groups are left out. */
export function groupAgents(agents: readonly Agent[]): AgentGroup[] {
  return STATUS_ORDER.map((status) => ({
    status,
    agents: agents
      .filter((a) => (STATUSES[a.status] ? a.status : 'unknown') === status)
      .sort((a, b) => agentName(a).localeCompare(agentName(b)) || a.paneId.localeCompare(b.paneId)),
  })).filter((group) => group.agents.length > 0);
}

/** One line for the page headline: "2 need you · 1 working · 3 idle". */
export function summary(agents: readonly Agent[]): string {
  if (agents.length === 0) return 'No agents running';
  return groupAgents(agents)
    .map(({ status, agents: group }) => {
      const n = group.length;
      if (status === 'blocked') return `${n} ${n === 1 ? 'needs' : 'need'} you`;
      return `${n} ${statusOf(status).label.toLowerCase()}`;
    })
    .join(' · ');
}
