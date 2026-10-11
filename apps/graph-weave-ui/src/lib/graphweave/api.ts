/**
 * GraphWeave API Client wrapping /graphweave-api endpoint (proxied to port 8001)
 */

export interface HealthStatus {
  status: string;
  services: {
    redis?: { status: string; error?: string };
    ai_gateway?: { status: string; url?: string; response_code?: number; error?: string };
    [key: string]: any;
  };
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

export interface WorkflowDetail extends WorkflowSummary {
  definition: {
    name: string;
    version: string;
    description?: string;
    metadata?: Record<string, any>;
    limits?: {
      max_hops?: number;
      max_tokens?: number;
      timeout_seconds?: number;
      stagnation_window?: number;
      stagnation_threshold?: number;
    };
    nodes: any[];
    edges: any[];
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

export interface ExecutionStatusResponse {
  run_id: string;
  status: string;
  workflow_id: string;
  tenant_id: string;
  events: any[];
  output: Record<string, any>;
  state: any;
  hop_count: number;
}

export interface NodeRegistryItem {
  node_id: string;
  node_name: string;
  tenant_id: string;
  description?: string;
  system_prompt?: string;
  user_prompt_template?: string;
  tags?: string[];
  config?: {
    system_prompt?: string;
    user_prompt_template?: string;
    [key: string]: any;
  };
  [key: string]: any;
}

const API_BASE = '/graphweave-api';

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
    try {
      const errJson = await response.json();
      errorDetail = errJson.message || errJson.detail || JSON.stringify(errJson);
    } catch {
      errorDetail = await response.text();
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
    return request<{ tenant_id: string; workflows: WorkflowSummary[]; count: number }>(`/workflows?${params.toString()}`);
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

  executeWorkflow: (tenantId: string, workflowId: string, inputPayload: Record<string, any>) =>
    request<{ run_id: string; thread_id: string; status: string; workflow_id: string; tenant_id: string }>(
      '/execute',
      {
        method: 'POST',
        body: JSON.stringify({
          tenant_id: tenantId,
          workflow_id: workflowId,
          input: inputPayload,
        }),
      }
    ),

  getExecutionStatus: (runId: string) => request<ExecutionStatusResponse>(`/execute/${runId}/status`),

  cancelExecution: (runId: string) =>
    request<{ run_id: string; status: string; thread_id: string }>(`/execute/${runId}/cancel`, {
      method: 'POST',
    }),

  recoverExecution: (runId: string, threadId: string) =>
    request<any>(`/execute/${runId}/recover`, {
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
