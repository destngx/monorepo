# Node Naming and Schema Convention Standard

This document outlines the strict naming and structural conventions for nodes within the GraphWeave ecosystem. Following these standards ensures deterministic DAG execution, clean state interpolation, and reliable dynamic workflow generation.

## 1. Structural Identifiers

| Field                    | Convention                         | Case                  | Example                                      | Rationale                                                  |
| :----------------------- | :--------------------------------- | :-------------------- | :------------------------------------------- | :--------------------------------------------------------- |
| **`node_id`** (Registry) | `{catalog_name}:{version}`         | `snake_case` + SemVer | `summarizer:v1.0.0`<br>`cli_executor:v1.2.0` | Unique global catalog index key.                           |
| **`node_name`**          | Singular role or capability        | `snake_case`          | `summarizer`<br>`fact_checker`               | Represents the generic capability template.                |
| **`alias`** (Workflow)   | `[verb]_[noun]` or `[noun]_[role]` | `snake_case`          | `summarize_inbox`<br>`pdf_ocr_extractor`     | Identifies the unique node instance inside a specific DAG. |
| **`display_name`**       | Human-readable descriptor          | `Title Case`          | `Summarize Inbox`<br>`PDF OCR Extractor`     | Used for UI visualization and graph rendering.             |

---

## 2. Functional Grammar rules for `alias` / `id` (inside a DAG)

To make it instantly clear what a node does just by looking at its alias, we apply specific verb/noun prefixes and postfixes based on the **node type**:

### A. Compute / Cognitive Nodes (Agent Nodes)

**Use Functional POSTFIXES** for cognitive nodes or classifiers. This describes the _role_ or _agent persona_ performing the work.

- **`_analyzer`**: `intent_analyzer`, `sentiment_analyzer`
- **`_classifier`**: `metadata_classifier`, `topic_classifier`
- **`_checker`** / **`_validator`**: `fact_checker`, `format_validator`
- **`_resolver`** / **`_generator`**: `node_resolver`, `edge_generator`
- **`_extractor`**: `entity_extractor`, `summary_extractor`

### B. Execution Nodes (CLI Nodes)

**Use Action PREFIXES** for nodes that run external commands, scripts, or deterministic actions.

- **`run_`**: `run_mac_ocr`, `run_linter`
- **`sync_`**: `sync_obsidian_vault`, `sync_db_cache`
- **`execute_`**: `execute_vitest`, `execute_query`
- **`generate_`**: `generate_markdown_report`

### C. Infrastructure / Control Flow Nodes

- **Control flow (`branch` / `guardrail`)**: Use **`is_[condition]`** or **`[subject]_gate`** (e.g., `is_valid_json`, `availability_gate`, `spam_check_gate`).
- **Graph boundaries**: Reserved exact keywords **`entry`** and **`exit`**.

---

## 3. The `NodeType` Enum vs `Capabilities` Array

The GraphWeave engine separates _how_ a node executes from _what_ a node does.

### The Engine Mechanics: `node_type` (Strict Enum)

You must **never create a new `node_type`** (like `slack_node` or `database_node`) just because you are integrating a new service. The `node_type` field dictates the execution handler in the core Python engine. It is strictly limited to an Enum:

- `agent_node`
- `cli_node`
- `orchestrator`
- `branch`
- `entry`
- `exit`
- `guardrail`
- `skill_loader`

### The Action Layer: `capabilities` (Dynamic)

To support infinite extensibility (e.g., integrating Slack, querying a database, parsing PDFs), we use the `.capabilities` array.

- Represent new integrations as an `agent_node` or `cli_node` equipped with dynamic MCP tools.
- Add arbitrary discovery strings to the array: `capabilities: ["slack", "notification", "database", "postgres"]`.
- This allows the `workflow-generator` and UI registry to discover what actions a node can perform without mutating the core execution engine code.

---

## 4. State & Output Variable Strictness

Predictability in state keys is vital since downstream nodes rely on upstream JSON paths (e.g., `$.intent_analyzer.output`).

1. **`output_key`**:
   Must match `^[a-z_][a-z0-9_]*$`. It cannot contain dots, dashes, or special characters because it is used directly as a Python/JS object key.
2. **`tags`**:
   Must match `^[a-z0-9-]+$` (lowercase kebab-case) for clean search indexing.
3. **`status`**:
   Must be one of `active`, `deprecated`, or `inactive`.
