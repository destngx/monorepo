/**
 * GraphWeave API Client. In production the UI is served by the API itself, so calls are same-origin;
 * in vite dev they go through the /graphweave-api proxy to port 8001.
 */

export interface ServiceHealth {
  status: string;
  error?: string;
  url?: string;
  response_code?: number;
}

export interface HealthStatus {
  status: string;
  services: Record<string, ServiceHealth | undefined>;
}

export interface WorkflowSummary {
  workflow_id: string;
  name: string;
  version: string;
  description?: string;
  tags: string[];
  owner: string;
  status: 'active' | 'archived' | 'draft' | 'deprecated';
  created_at?: string;
  updated_at?: string;
}

/** JSON-schema-style field on an entry node's `config.properties`. */
export interface EntryProperty {
  type?: string;
  description?: string;
}

export interface WorkflowNodeConfig {
  system_prompt?: string;
  user_prompt_template?: string;
  output_key?: string;
  max_iterations?: number;
  condition_expression?: string;
  command?: string;
  properties?: Record<string, EntryProperty>;
  required?: string[];
  [key: string]: unknown;
}

export interface WorkflowNode {
  id: string;
  type: string;
  display_name?: string;
  description?: string;
  alias?: string;
  node_id?: string;
  node_hash?: string;
  tenant_id?: string;
  config?: WorkflowNodeConfig;
  overrides?: Record<string, unknown>;
  guardrails?: Record<string, unknown>;
  retry_config?: Record<string, unknown>;
}

export interface WorkflowEdge {
  from: string;
  to: string;
  condition?: string;
  config?: { label?: string };
}

export interface WorkflowDetail extends WorkflowSummary {
  definition: {
    name: string;
    version: string;
    description?: string;
    metadata?: Record<string, unknown>;
    limits?: {
      max_hops?: number;
      max_tokens?: number;
      timeout_seconds?: number;
      stagnation_window?: number;
      stagnation_threshold?: number;
    };
    nodes: WorkflowNode[];
    edges: WorkflowEdge[];
  };
}

export interface ExecutionRun {
  run_id: string;
  thread_id: string;
  workflow_id: string;
  tenant_id: string;
  status: string;
  hop_count: number;
}

/** Payload of an execution event; which fields are set depends on the event type. */
export interface ExecutionEventData {
  tenant_id?: string;
  workflow_id?: string;
  node_id?: string;
  node_type?: string;
  hop_count?: number;
  provider?: string;
  model?: string;
  tool_count?: number;
  count?: number;
  tool?: string;
  id?: string;
  from?: string;
  to?: string;
  error?: string;
  input?: Record<string, unknown>;
  output?: Record<string, unknown>;
  result?: unknown;
}

export interface ExecutionEvent {
  type: string;
  timestamp?: string;
  data?: ExecutionEventData;
}

export interface ExecutionStatusResponse {
  run_id: string;
  status: string;
  workflow_id: string;
  tenant_id: string;
  events: ExecutionEvent[];
  output: Record<string, unknown>;
  state: Record<string, unknown> | null;
  hop_count: number;
}

export interface ExecuteResponse {
  run_id: string;
  thread_id: string;
  status: string;
  workflow_id: string;
  tenant_id: string;
  output: Record<string, unknown>;
  state: Record<string, unknown> | null;
  events: ExecutionEvent[];
}

export interface NodeRegistryConfig {
  system_prompt?: string;
  user_prompt_template?: string;
  output_key?: string;
  command?: string;
  provider?: string;
  model?: string;
  temperature?: number;
  input_mapping?: Record<string, unknown>;
  output_schema?: Record<string, unknown>;
  max_iterations?: number;
  condition_expression?: string;
}

export interface NodeRegistryItem {
  node_id: string;
  node_name: string;
  tenant_id: string;
  name?: string;
  type?: string;
  version?: string;
  description?: string;
  system_prompt?: string;
  user_prompt_template?: string;
  tags?: string[];
  capabilities?: string[];
  owner?: string;
  status?: string;
  config?: NodeRegistryConfig;
}

const API_BASE = import.meta.env.DEV ? '/graphweave-api' : '';

/** Picks the most useful message out of a FastAPI or GraphWeave error body. */
function describeErrorBody(body: unknown): string {
  if (body && typeof body === 'object') {
    const { message, detail } = body as { message?: unknown; detail?: unknown };
    if (typeof message === 'string' && message) return message;
    if (typeof detail === 'string' && detail) return detail;
  }
  return JSON.stringify(body);
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${API_BASE}${path}`;
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options?.headers || {}),
    },
  });

  if (!response.ok) {
    let errorDetail = '';
    const body = await response.text();
    try {
      errorDetail = describeErrorBody(JSON.parse(body));
    } catch {
      errorDetail = body;
    }
    throw new Error(`API Error [${response.status}]: ${errorDetail}`);
  }

  if (response.status === 204) {
    return {} as T;
  }

  return response.json() as Promise<T>;
}

export const graphWeaveApi = {
  // Health
  getHealth: () => request<HealthStatus>('/health'),

  // Workflows CRUD
  listTenants: () => request<{ tenants: string[]; count: number }>('/workflows/tenants'),

  listWorkflows: (tenantId: string, filters?: { status?: string; owner?: string; tags?: string }) => {
    const params = new URLSearchParams({ tenant_id: tenantId });
    if (filters?.status) params.append('status', filters.status);
    if (filters?.owner) params.append('owner', filters.owner);
    if (filters?.tags) params.append('tags', filters.tags);
    return request<{ tenant_id: string; workflows: WorkflowSummary[]; count: number }>(
      `/workflows?${params.toString()}`,
    );
  },

  getWorkflow: (workflowId: string, tenantId: string) =>
    request<WorkflowDetail>(`/workflows/${encodeURIComponent(workflowId)}?tenant_id=${encodeURIComponent(tenantId)}`),

  createWorkflow: (workflow: Partial<WorkflowDetail>) =>
    request<WorkflowDetail>('/workflows', {
      method: 'POST',
      body: JSON.stringify(workflow),
    }),

  updateWorkflow: (workflowId: string, tenantId: string, updates: Partial<WorkflowDetail>) =>
    request<WorkflowDetail>(`/workflows/${encodeURIComponent(workflowId)}?tenant_id=${encodeURIComponent(tenantId)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    }),

  deleteWorkflow: (workflowId: string, tenantId: string) =>
    request<void>(`/workflows/${encodeURIComponent(workflowId)}?tenant_id=${encodeURIComponent(tenantId)}`, {
      method: 'DELETE',
    }),

  // Executions
  listExecutions: (tenantId?: string, workflowId?: string) => {
    const params = new URLSearchParams();
    if (tenantId) params.append('tenant_id', tenantId);
    if (workflowId) params.append('workflow_id', workflowId);
    const queryStr = params.toString();
    return request<{ runs: ExecutionRun[]; count: number }>(`/execute${queryStr ? `?${queryStr}` : ''}`);
  },

  executeWorkflow: (tenantId: string, workflowId: string, inputPayload: Record<string, unknown>) =>
    request<{ run_id: string; thread_id: string; status: string; workflow_id: string; tenant_id: string }>('/execute', {
      method: 'POST',
      body: JSON.stringify({
        tenant_id: tenantId,
        workflow_id: workflowId,
        input: inputPayload,
      }),
    }),

  getExecutionStatus: (runId: string) => request<ExecutionStatusResponse>(`/execute/${runId}/status`),

  cancelExecution: (runId: string) =>
    request<{ run_id: string; status: string; thread_id: string }>(`/execute/${runId}/cancel`, {
      method: 'POST',
    }),

  recoverExecution: (runId: string, threadId: string) =>
    request<ExecuteResponse>(`/execute/${runId}/recover`, {
      method: 'POST',
      body: JSON.stringify({ thread_id: threadId }),
    }),

  // Reusable Standalone Nodes
  listNodes: (tenantId: string, name?: string, tags?: string) => {
    const params = new URLSearchParams({ tenant_id: tenantId });
    if (name) params.append('node_name', name);
    if (tags) params.append('tags', tags);
    return request<{ nodes: NodeRegistryItem[]; count: number }>(`/nodes/?${params.toString()}`);
  },

  getNode: (nodeId: string, tenantId: string) =>
    request<NodeRegistryItem>(`/nodes/${encodeURIComponent(nodeId)}?tenant_id=${encodeURIComponent(tenantId)}`),

  createNode: (node: Partial<NodeRegistryItem>) =>
    request<NodeRegistryItem>('/nodes/', {
      method: 'POST',
      body: JSON.stringify(node),
    }),

  updateNode: (nodeId: string, tenantId: string, updates: Partial<NodeRegistryItem>) =>
    request<NodeRegistryItem>(`/nodes/${encodeURIComponent(nodeId)}?tenant_id=${encodeURIComponent(tenantId)}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    }),

  deleteNode: (nodeId: string, tenantId: string) =>
    request<{ deleted: boolean }>(`/nodes/${encodeURIComponent(nodeId)}?tenant_id=${encodeURIComponent(tenantId)}`, {
      method: 'DELETE',
    }),
};
