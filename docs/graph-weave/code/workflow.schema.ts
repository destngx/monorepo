import { z } from 'zod';

const NodeTypeEnum = z.enum(['entry', 'exit', 'skill_call', 'branch', 'human_decision']);

const GuardrailConfig = z.object({
  model: z.string().optional(),
  max_tokens: z.number().int().positive().optional(),
  blocked_patterns: z.array(z.string()).optional(),
  pii_detection: z.boolean().optional(),
  required_format: z.string().optional(),
  blocked_topics: z.array(z.string()).optional(),
});

const NodeGuardrails = z.object({
  input: GuardrailConfig.optional(),
  output: GuardrailConfig.optional(),
});

const BaseNode = z.object({
  id: z
    .string()
    .min(1)
    .regex(/^[a-z_]+$/),
  type: NodeTypeEnum,
  display_name: z.string().optional(),
  description: z.string().optional(),
  guardrails: NodeGuardrails.optional(),
});

const EntryNode = BaseNode.extend({
  type: z.literal('entry'),
  config: z.object({
    properties: z.record(z.string(), z.any()),
    required: z.array(z.string()).optional(),
  }),
});

const ExitNode = BaseNode.extend({
  type: z.literal('exit'),
  config: z.object({
    output_mapping: z.record(z.string(), z.any()),
  }),
});

const SkillCallNode = BaseNode.extend({
  type: z.literal('skill_call'),
  config: z.object({
    skill_id: z.string().min(1),
    input_mapping: z.record(z.string(), z.string()).optional(),
    output_key: z.string().min(1),
  }),
});

const BranchNode = BaseNode.extend({
  type: z.literal('branch'),
  config: z.object({
    condition_expression: z.string().min(1),
  }),
});

const HumanDecisionNode = BaseNode.extend({
  type: z.literal('human_decision'),
  config: z.object({
    prompt: z.string(),
    options: z.array(z.string()).min(1).optional(),
  }),
});

const Node = z.discriminatedUnion('type', [EntryNode, ExitNode, SkillCallNode, BranchNode, HumanDecisionNode]);

const Edge = z.object({
  id: z.string().optional(),
  from: z.string().min(1),
  to: z.string().min(1),
  condition: z.string().optional(),
  config: z.object({ label: z.string().optional() }).optional(),
});

const WorkflowLimits = z.object({
  max_hops: z.number().int().positive().default(20),
  max_tokens: z.number().int().positive().default(100000),
  timeout_seconds: z.number().int().positive().default(300),
  stagnation_window: z.number().int().positive().default(3),
  stagnation_threshold: z.number().int().positive().default(2),
});

const WorkflowMetadata = z.object({
  author: z.string().min(1),
  tenant_id: z.string().optional(),
  created_at: z.string().datetime().optional(),
  tags: z.array(z.string()).optional(),
});

export const WorkflowSchema = z
  .object({
    name: z.string().min(1),
    version: z.string().regex(/^\d+\.\d+\.\d+$/),
    description: z.string().optional(),
    metadata: WorkflowMetadata.optional(),
    limits: WorkflowLimits.optional(),
    nodes: z.array(Node).min(2),
    edges: z.array(Edge).min(1),
  })
  .strict()
  .refine(
    (workflow) => {
      const hasEntry = workflow.nodes.some((n) => n.type === 'entry');
      const hasExit = workflow.nodes.some((n) => n.type === 'exit');
      return hasEntry && hasExit;
    },
    { message: 'Workflow must have entry and exit nodes' },
  )
  .refine(
    (workflow) => {
      const nodeIds = new Set(workflow.nodes.map((n) => n.id));
      return workflow.edges.every((edge) => nodeIds.has(edge.from) && nodeIds.has(edge.to));
    },
    { message: 'All edges must reference existing node IDs' },
  )
  .refine(
    (workflow) => {
      const ids = workflow.nodes.map((n) => n.id);
      return new Set(ids).size === ids.length;
    },
    { message: 'Node IDs must be unique' },
  );

export type Workflow = z.infer<typeof WorkflowSchema>;

export function validateWorkflow(data: unknown): {
  valid: boolean;
  workflow?: Workflow;
  errors?: Array<{ message: string }>;
} {
  const result = WorkflowSchema.safeParse(data);

  if (result.success) {
    return { valid: true, workflow: result.data };
  }

  return {
    valid: false,
    errors: result.error.issues.map((issue) => ({
      message: issue.message,
    })),
  };
}

export function getNodesByType(workflow: Workflow, type: string) {
  return workflow.nodes.filter((n) => n.type === type);
}

export function getOutgoingEdges(workflow: Workflow, nodeId: string) {
  return workflow.edges.filter((e) => e.from === nodeId);
}

export function getIncomingEdges(workflow: Workflow, nodeId: string) {
  return workflow.edges.filter((e) => e.to === nodeId);
}

export function findNode(workflow: Workflow, nodeId: string) {
  return workflow.nodes.find((n) => n.id === nodeId);
}

export function validateWorkflowIntegrity(workflow: Workflow): { issues: string[]; valid: boolean } {
  const issues: string[] = [];
  const reachableNodes = new Set<string>();

  const entryNodes = workflow.nodes.filter((n) => n.type === 'entry');
  const queue = [...entryNodes.map((n) => n.id)];

  queue.forEach((id) => {
    reachableNodes.add(id);
  });

  while (queue.length > 0) {
    const nodeId = queue.shift();
    if (!nodeId) break;

    const outgoing = workflow.edges.filter((e) => e.from === nodeId);
    outgoing.forEach((edge) => {
      if (!reachableNodes.has(edge.to)) {
        reachableNodes.add(edge.to);
        queue.push(edge.to);
      }
    });
  }

  workflow.nodes.forEach((node) => {
    if (node.type !== 'entry' && !reachableNodes.has(node.id)) {
      issues.push(`Unreachable node: ${node.id}`);
    }
  });

  workflow.nodes.forEach((node) => {
    if (node.type === 'exit') return;
    const outgoing = workflow.edges.filter((e) => e.from === node.id);
    if (outgoing.length === 0) {
      issues.push(`Dead-end node: ${node.id}`);
    }
  });

  const exitNodes = workflow.nodes.filter((n) => n.type === 'exit');
  exitNodes.forEach((exitNode) => {
    if (!reachableNodes.has(exitNode.id)) {
      issues.push(`Exit unreachable: ${exitNode.id}`);
    }
  });

  workflow.nodes.forEach((node) => {
    if (node.type === 'skill_call' && !node.config.skill_id) {
      issues.push(`skill_call missing skill_id: ${node.id}`);
    }
  });

  return {
    issues,
    valid: issues.length === 0,
  };
}

export function evaluateCondition(condition: string, state: Record<string, unknown>): boolean {
  const expr = condition.replace(/^\$\./, '');

  const operators = ['===', '!==', '==', '!=', '<=', '>=', '<', '>', '&&', '||'];
  const pattern = new RegExp(`(${operators.join('|')})`);
  const matched = expr.match(pattern);

  if (!matched) {
    const value = getNestedValue(state, expr.trim());
    return Boolean(value);
  }

  const [operator] = matched;
  const parts = expr.split(new RegExp(`\\s*${operator.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*`));
  const [left, right] = [parts[0], parts[1]];

  const leftVal = getNestedValue(state, left.trim());
  const rightVal = parseValue(right.trim());

  switch (operator) {
    case '==':
      return leftVal == rightVal;
    case '!=':
      return leftVal != rightVal;
    case '===':
      return leftVal === rightVal;
    case '!==':
      return leftVal !== rightVal;
    case '<':
      return leftVal < rightVal;
    case '>':
      return leftVal > rightVal;
    case '<=':
      return leftVal <= rightVal;
    case '>=':
      return leftVal >= rightVal;
    default:
      throw new Error(`Unknown operator: ${operator}`);
  }
}

function getNestedValue(obj: unknown, path: string): unknown {
  const keys = path.split('.');
  let current = obj;

  for (const key of keys) {
    if (current === null || current === undefined) return undefined;
    if (typeof current !== 'object') return undefined;
    current = (current as Record<string, unknown>)[key];
  }

  return current;
}

function parseValue(val: string): unknown {
  if (val.startsWith("'") && val.endsWith("'")) {
    return val.slice(1, -1);
  }
  if (val.startsWith('"') && val.endsWith('"')) {
    return val.slice(1, -1);
  }

  if (!isNaN(parseFloat(val))) {
    return parseFloat(val);
  }

  if (val === 'true') return true;
  if (val === 'false') return false;
  if (val === 'null') return null;

  return val;
}
