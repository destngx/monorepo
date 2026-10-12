<script lang="ts">
  import { onMount } from 'svelte';
  import { graphWeaveApi, type WorkflowSummary } from '../../graphweave/api';

  // Props
  let {
    tenantId = $bindable('system'),
    selectedWorkflowId = $bindable(null),
    onSelectWorkflow = (workflowId: string) => {},
  } = $props<{
    tenantId: string;
    selectedWorkflowId: string | null;
    onSelectWorkflow: (workflowId: string) => void;
  }>();

  // Component states
  let workflows = $state<WorkflowSummary[]>([]);
  let tenants = $state<string[]>(['system', 'default']);
  let customTenant = $state('');
  let isAddingTenant = $state(false);
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  // Import workflow states
  let showImportModal = $state(false);
  let importName = $state('');
  let importVersion = $state('1.0.0');
  let importOwner = $state('admin');
  let importDescription = $state('');
  let importTags = $state('research, default');
  let importDefinition = $state('');
  let importError = $state<string | null>(null);
  let importLoading = $state(false);

  async function fetchTenants() {
    try {
      const res = await graphWeaveApi.listTenants();
      const loadedTenants = res.tenants || [];
      tenants = Array.from(new Set(['system', 'default', ...loadedTenants]));
    } catch (err) {
      console.warn('Failed to fetch tenants registry, using fallback list', err);
    }
  }

  async function fetchWorkflows() {
    if (!tenantId) return;
    isLoading = true;
    error = null;
    try {
      const res = await graphWeaveApi.listWorkflows(tenantId);
      workflows = res.workflows || [];
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to fetch workflows';
      workflows = [];
    } finally {
      isLoading = false;
    }
  }

  $effect(() => {
    fetchWorkflows();
  });

  onMount(() => {
    fetchTenants();
    fetchWorkflows();
  });

  function handleAddCustomTenant() {
    if (!customTenant || !customTenant.trim()) return;
    const t = customTenant.trim().toLowerCase().replace(/\s+/g, '_');
    if (!tenants.includes(t)) {
      tenants = [...tenants, t];
    }
    tenantId = t;
    customTenant = '';
    isAddingTenant = false;
  }

  async function handleDelete(workflowId: string) {
    if (!confirm(`Are you sure you want to permanently delete workflow: ${workflowId}?`)) return;
    try {
      await graphWeaveApi.deleteWorkflow(workflowId, tenantId);
      if (selectedWorkflowId === workflowId) selectedWorkflowId = null;
      await fetchWorkflows();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Deletion failed');
    }
  }

  async function handleImport() {
    importError = null;
    importLoading = true;

    try {
      if (!importName || !importVersion || !importOwner || !importDefinition) {
        throw new Error('Name, Version, Owner, and Definition fields are required');
      }

      let parsedDefinition;
      try {
        parsedDefinition = JSON.parse(importDefinition);
      } catch (e) {
        throw new Error('Definition must be a valid JSON string matching GraphWeave Spec', { cause: e });
      }

      const workflowId = `${importName.trim().toLowerCase().replace(/\s+/g, '-')}:${importVersion.trim()}`;

      const payload = {
        tenant_id: tenantId,
        workflow_id: workflowId,
        name: importName.trim(),
        version: importVersion.trim(),
        description: importDescription.trim(),
        owner: importOwner.trim(),
        tags: importTags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
        definition: parsedDefinition,
      };

      await graphWeaveApi.createWorkflow(payload);

      showImportModal = false;
      importName = '';
      importVersion = '1.0.0';
      importDescription = '';
      importDefinition = '';

      await fetchWorkflows();
      onSelectWorkflow(workflowId);
    } catch (err) {
      importError = err instanceof Error ? err.message : 'Import failed';
    } finally {
      importLoading = false;
    }
  }

  function loadExampleTemplate() {
    const example = {
      name: importName || 'dynamic-agent-task',
      version: importVersion || '1.0.0',
      description: 'Simple multi-agent execution with prompt automation',
      limits: { max_hops: 10, timeout_seconds: 300 },
      nodes: [
        {
          id: 'entry',
          type: 'entry',
          display_name: 'Workflow Start',
          config: {
            properties: {
              topic: { type: 'string', description: 'Subject of research' },
              depth: { type: 'string', description: 'Quick or thorough depth' },
            },
            required: ['topic'],
          },
        },
        {
          id: 'agent_research',
          type: 'agent_node',
          display_name: 'Researcher Agent',
          config: {
            system_prompt:
              "You are an AI research assistant. You have access to:\n- load_skill(skill_name): Load skill playbook\n- search(query): Web search\n\nLoad the 'research' skill to guide your methodology. Return a clean analysis with findings.",
            user_prompt_template: 'Perform research on: {topic}. Target depth is: {depth}.',
            input_mapping: {
              topic: '$.entry.topic',
              depth: '$.entry.depth',
            },
            output_key: 'research_output',
          },
        },
        {
          id: 'exit',
          type: 'exit',
          display_name: 'Complete',
          config: {
            output_mapping: {
              final_report: '$.research_output',
            },
          },
        },
      ],
      edges: [
        { from: 'entry', to: 'agent_research' },
        { from: 'agent_research', to: 'exit' },
      ],
    };
    importDefinition = JSON.stringify(example, null, 2);
  }
</script>

<div class="space-y-4">
  <!-- Compact Tenant Scope & Actions -->
  <div
    class="flex flex-col gap-2.5 p-4 rounded-xl border border-slate-900 bg-slate-950/40 backdrop-blur-xl relative z-10"
  >
    <div class="flex flex-col gap-1 text-left select-none">
      <label for="tenant-sel" class="text-[8px] font-extrabold uppercase tracking-widest text-slate-500"
        >Scope Context</label
      >

      {#if isAddingTenant}
        <div class="flex items-center gap-1.5 mt-1 select-text">
          <input
            type="text"
            bind:value={customTenant}
            placeholder="tenant_id"
            class="rounded-lg px-2.5 py-1.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 font-mono w-28 glass-input min-h-[32px]"
            onkeydown={(e) => e.key === 'Enter' && handleAddCustomTenant()}
          />
          <button
            onclick={handleAddCustomTenant}
            class="px-2.5 py-1.5 text-[10px] font-bold text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg cursor-pointer transition-colors shadow shadow-indigo-600/10 min-h-[32px]"
          >
            Add
          </button>
          <button
            onclick={() => (isAddingTenant = false)}
            class="text-[10px] text-slate-550 hover:text-slate-350 px-1 cursor-pointer min-h-[32px]"
          >
            Cancel
          </button>
        </div>
      {:else}
        <div class="flex items-center gap-2 mt-1">
          <select
            id="tenant-sel"
            bind:value={tenantId}
            class="bg-slate-950/80 border border-slate-850 rounded-lg px-3 py-1.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 cursor-pointer flex-1 min-h-[36px]"
          >
            {#each tenants as tenant}
              <option value={tenant}>{tenant}</option>
            {/each}
          </select>

          <button
            onclick={() => (isAddingTenant = true)}
            class="p-2 rounded-lg border border-slate-800 bg-slate-950/40 text-slate-400 hover:text-slate-200 hover:border-slate-700 transition-colors cursor-pointer flex items-center justify-center min-h-[36px] w-[36px]"
            title="Add Custom Tenant"
            aria-label="Add Tenant"
          >
            <svg width="16" height="16" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
            </svg>
          </button>
        </div>
      {/if}
    </div>

    <!-- Sync and Register Actions -->
    <div class="flex items-center gap-2 select-none pt-1 border-t border-slate-900/60 mt-1">
      <button
        onclick={fetchWorkflows}
        class="flex-1 px-2.5 py-2 text-[10px] font-bold text-slate-300 border border-slate-850 bg-slate-950/30 hover:bg-slate-900 rounded-lg hover:border-slate-700 transition-all cursor-pointer flex items-center justify-center gap-1.5 min-h-[32px]"
        disabled={isLoading}
      >
        <svg
          width="12"
          height="12"
          class="text-slate-400 {isLoading ? 'animate-spin text-indigo-400' : ''}"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2" />
        </svg>
        Sync
      </button>

      <button
        onclick={() => (showImportModal = true)}
        class="flex-1 px-2.5 py-2 text-[10px] font-extrabold text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg shadow-sm transition-all cursor-pointer flex items-center justify-center gap-1.5 min-h-[32px]"
      >
        <svg width="12" height="12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"
          />
        </svg>
        Register
      </button>
    </div>
  </div>

  <!-- Explorer Header -->
  <div class="flex items-center justify-between px-1 select-none">
    <div class="text-left">
      <h3 class="text-[9px] font-extrabold font-display uppercase tracking-widest text-slate-500">Registered Graphs</h3>
    </div>
    <span
      class="text-[8px] bg-slate-950 border border-slate-900 px-2 py-0.5 rounded-md font-mono text-indigo-400 font-extrabold"
    >
      {workflows.length}
    </span>
  </div>

  <!-- Loading & Statuses -->
  {#if isLoading}
    <div class="py-8 text-center glass-slab rounded-xl select-none">
      <svg
        width="20"
        height="20"
        class="animate-spin text-indigo-500 mx-auto mb-2"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2.5"
      >
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2" />
      </svg>
      <span class="text-[10px] text-slate-400 font-semibold tracking-wide block">Syncing Registry...</span>
    </div>
  {:else if error}
    <div class="py-6 px-4 text-center glass-slab rounded-xl border-rose-500/10">
      <svg
        width="24"
        height="24"
        class="text-rose-500/40 mx-auto mb-1.5 stroke-1"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
        />
      </svg>
      <span class="text-[9px] font-extrabold uppercase tracking-widest block text-rose-400">Sync Error</span>
      <span
        class="text-[10px] text-slate-500 font-mono leading-relaxed block mb-3 break-all max-h-16 overflow-y-auto custom-scrollbar select-text"
        >{error}</span
      >
      <button
        onclick={fetchWorkflows}
        class="px-3 py-1.5 text-[9px] font-bold text-indigo-400 border border-indigo-500/20 bg-indigo-500/5 rounded-lg hover:bg-indigo-500/10 cursor-pointer min-h-[28px] w-full"
        >Retry Sync</button
      >
    </div>
  {:else if workflows.length === 0}
    <div class="py-10 px-4 text-center glass-slab rounded-xl select-none">
      <svg
        width="32"
        height="32"
        class="mb-2 stroke-1 text-slate-700 mx-auto"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
        />
      </svg>
      <span class="text-[10px] font-extrabold uppercase tracking-widest text-slate-500">No active workflows</span>
      <button
        onclick={() => (showImportModal = true)}
        class="mt-3 px-3 py-1.5 text-[9px] font-bold text-indigo-400 border border-indigo-500/20 bg-indigo-500/5 hover:bg-indigo-500/10 hover:border-indigo-500/40 rounded-lg transition-all cursor-pointer w-full"
      >
        Load Example Spec
      </button>
    </div>
  {:else}
    <!-- Compact Vertical Explorer List -->
    <div class="flex flex-col gap-2 max-h-[30vh] overflow-y-auto custom-scrollbar pr-1 select-none">
      {#each workflows as wf}
        {@const isSelected = selectedWorkflowId === wf.workflow_id}

        <div
          role="button"
          tabindex="0"
          onclick={() => onSelectWorkflow(wf.workflow_id)}
          onkeydown={(e) => {
            if (e.key === 'Enter' || e.key === ' ') {
              e.preventDefault();
              onSelectWorkflow(wf.workflow_id);
            }
          }}
          class="group flex flex-col p-3 rounded-xl transition-all duration-200 cursor-pointer relative overflow-hidden backdrop-blur-xl border text-left
            {isSelected
            ? 'border-indigo-500/60 bg-indigo-950/15 shadow-sm shadow-indigo-500/5'
            : 'border-slate-900/60 bg-slate-950/20 hover:border-slate-800'}"
        >
          <!-- Neon left indicator bar for active item -->
          {#if isSelected}
            <div class="absolute left-0 inset-y-0 w-[3px] bg-gradient-to-b from-indigo-500 to-violet-600"></div>
          {/if}

          <div class="flex items-center justify-between gap-2.5 {isSelected ? 'pl-1' : ''}">
            <div class="min-w-0">
              <h4 class="text-xs font-bold text-slate-200 group-hover:text-white transition-colors truncate">
                {wf.name}
              </h4>
              <span class="text-[9px] text-slate-500 font-mono mt-0.5 block truncate select-all">{wf.workflow_id}</span>
            </div>

            <div class="shrink-0 flex items-center gap-1.5">
              <span
                class="px-1.5 py-0.5 rounded text-[8px] font-mono bg-slate-950 text-slate-400 border border-slate-900 shadow-inner"
              >
                v{wf.version}
              </span>

              <!-- Compact Delete Button -->
              <button
                onclick={(e) => {
                  e.stopPropagation();
                  handleDelete(wf.workflow_id);
                }}
                class="p-1 rounded-lg border border-slate-900 hover:border-rose-500/30 hover:bg-rose-500/10 text-slate-650 hover:text-rose-455 transition-colors cursor-pointer flex items-center justify-center shadow-sm"
                title="Delete Workflow"
                aria-label="Delete Workflow"
              >
                <svg width="12" height="12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            </div>
          </div>

          <!-- Condensed Metadata Footer -->
          <div
            class="mt-2 flex items-center justify-between text-[8px] text-slate-500 border-t border-slate-900/40 pt-1.5 {isSelected
              ? 'pl-1'
              : ''}"
          >
            <span class="truncate max-w-[90px]">{wf.owner}</span>
            <span
              class="uppercase font-semibold tracking-wider {wf.status === 'active'
                ? 'text-emerald-500'
                : 'text-amber-500'}"
            >
              {wf.status}
            </span>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Modal Overlay for Registering Workflows -->
{#if showImportModal}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-fade-in">
    <div
      class="relative w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl flex flex-col max-h-[92vh] overflow-hidden"
    >
      <!-- Top Neon Header -->
      <div class="absolute top-0 inset-x-0 h-[3px] bg-gradient-to-r from-indigo-500 via-indigo-600 to-violet-600"></div>

      <!-- Header -->
      <div class="px-6 py-5 border-b border-slate-800/80 flex items-center justify-between select-none">
        <h3 class="text-sm font-extrabold font-display uppercase tracking-widest text-slate-200">
          Register Orchestrator Graph
        </h3>
        <button
          onclick={() => (showImportModal = false)}
          class="p-2 text-slate-500 hover:text-slate-350 cursor-pointer tap-target flex items-center justify-center rounded-lg"
          aria-label="Close"
          title="Close"
        >
          <svg
            width="24"
            height="24"
            class="w-5.5 h-5.5"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto p-6 space-y-4.5 text-left custom-scrollbar">
        {#if importError}
          <div
            class="bg-rose-950/20 border border-rose-900/40 text-rose-350 p-3.5 rounded-xl text-xs leading-relaxed font-mono"
          >
            <strong>Spec Error:</strong>
            {importError}
          </div>
        {/if}

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label for="imp-name" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
              >Workflow Name (id)</label
            >
            <input
              type="text"
              id="imp-name"
              bind:value={importName}
              placeholder="e.g. market-analyzer"
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 min-h-[44px] glass-input"
            />
          </div>
          <div class="space-y-1.5">
            <label for="imp-version" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
              >Semantic Version</label
            >
            <input
              type="text"
              id="imp-version"
              bind:value={importVersion}
              placeholder="e.g. 1.0.0"
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 font-mono min-h-[44px] glass-input"
            />
          </div>
        </div>

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <div class="space-y-1.5">
            <label for="imp-owner" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
              >Workflow Owner</label
            >
            <input
              type="text"
              id="imp-owner"
              bind:value={importOwner}
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 min-h-[44px] glass-input"
            />
          </div>
          <div class="space-y-1.5">
            <label for="imp-tags" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
              >Tags (comma separated)</label
            >
            <input
              type="text"
              id="imp-tags"
              bind:value={importTags}
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 min-h-[44px] glass-input"
            />
          </div>
        </div>

        <div class="space-y-1.5">
          <label for="imp-desc" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
            >Workflow Description</label
          >
          <input
            type="text"
            id="imp-desc"
            bind:value={importDescription}
            placeholder="e.g. Multi-agent dynamic analysis pipeline with LLM reasoning loop."
            class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 min-h-[44px] glass-input"
          />
        </div>

        <!-- JSON Specification Field -->
        <div class="space-y-1.5">
          <div class="flex justify-between items-center select-none">
            <label for="imp-def" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
              >JSON Specification Definition</label
            >
            <button
              type="button"
              onclick={loadExampleTemplate}
              class="text-[9px] font-extrabold text-indigo-400 hover:text-indigo-300 uppercase tracking-widest cursor-pointer py-1"
            >
              Load Example Template
            </button>
          </div>
          <textarea
            id="imp-def"
            bind:value={importDefinition}
            rows="10"
            class="w-full bg-[#020307]/90 border border-slate-850 rounded-xl p-3.5 text-xs text-indigo-300 font-mono focus:outline-none focus:border-indigo-500 leading-relaxed custom-scrollbar shadow-inner"
            placeholder="Paste raw JSON matching GraphWeave specification..."
          ></textarea>
        </div>
      </div>

      <!-- Footer -->
      <div class="px-6 py-4 border-t border-slate-800 bg-slate-950/20 flex justify-end gap-3 items-center select-none">
        <button
          onclick={() => (showImportModal = false)}
          class="px-4 py-2 text-xs font-bold text-slate-400 hover:text-slate-250 cursor-pointer min-h-[44px]"
        >
          Cancel
        </button>

        <button
          onclick={handleImport}
          disabled={importLoading || !importName || !importDefinition}
          class="px-5 py-2 text-xs font-extrabold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed transition-all min-h-[44px] shadow shadow-indigo-600/10"
        >
          {#if importLoading}
            Registering...
          {:else}
            Register Blueprint
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}
