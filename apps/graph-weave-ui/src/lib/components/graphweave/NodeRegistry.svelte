<script lang="ts">
  import { onMount } from 'svelte';
  import { graphWeaveApi, type NodeRegistryItem } from '../../graphweave/api';

  // Props using Svelte 5 runes
  let {
    tenantId = 'system',
    onSelectTemplate = null
  } = $props<{
    tenantId: string;
    onSelectTemplate?: ((node: NodeRegistryItem) => void) | null;
  }>();

  // Component states
  let nodes = $state<NodeRegistryItem[]>([]);
  let searchQuery = $state('');
  let tagFilter = $state('');
  let isLoading = $state(false);
  let error = $state<string | null>(null);

  // Edit / Create States
  let showEditModal = $state(false);
  let isEditing = $state(false);
  let modalError = $state<string | null>(null);
  let modalLoading = $state(false);

  // Form Fields
  let formNodeId = $state('');
  let formNodeName = $state('');
  let formDescription = $state('');
  let formSystemPrompt = $state('');
  let formUserPromptTemplate = $state('');
  let formTags = $state('');

  // Fetch nodes list reactively on tenantId change
  $effect(() => {
    if (tenantId) {
      fetchNodes();
    }
  });

  onMount(() => {
    fetchNodes();
  });

  async function fetchNodes() {
    isLoading = true;
    error = null;
    try {
      const res = await graphWeaveApi.listNodes(tenantId, searchQuery || undefined, tagFilter || undefined);
      nodes = res.nodes || [];
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to fetch custom nodes';
      nodes = [];
    } finally {
      isLoading = false;
    }
  }

  function openCreateModal() {
    isEditing = false;
    formNodeId = '';
    formNodeName = '';
    formDescription = '';
    formSystemPrompt = 'You are an expert AI agent specialized in financial wealth intelligence.';
    formUserPromptTemplate = 'Analyze the following asset context: {input_query}';
    formTags = 'finance, custom';
    modalError = null;
    showEditModal = true;
  }

  function openEditModal(node: NodeRegistryItem) {
    isEditing = true;
    formNodeId = node.node_id;
    formNodeName = node.node_name;
    formDescription = node.description || '';
    
    const cfg = node.config || {};
    formSystemPrompt = cfg.system_prompt || node.system_prompt || '';
    formUserPromptTemplate = cfg.user_prompt_template || node.user_prompt_template || '';
    
    formTags = (node.tags || []).join(', ');
    modalError = null;
    showEditModal = true;
  }

  async function handleSaveNode() {
    modalError = null;
    modalLoading = true;

    try {
      if (!formNodeId || !formNodeName) {
        throw new Error('Node ID and Node Name fields are required');
      }

      if (isEditing) {
        const payload = {
          name: formNodeName.trim(),
          description: formDescription.trim(),
          tags: formTags.split(',').map(t => t.trim()).filter(Boolean),
          config: {
            system_prompt: formSystemPrompt,
            user_prompt_template: formUserPromptTemplate,
            input_mapping: {},
            output_schema: {},
            output_key: ''
          }
        };
        await graphWeaveApi.updateNode(formNodeId, tenantId, payload);
      } else {
        const payload = {
          tenant_id: tenantId,
          node_id: formNodeId.trim().toLowerCase().replace(/\s+/g, '-'),
          node_name: formNodeName.trim(),
          version: '1.0.0',
          name: formNodeName.trim(),
          type: 'agent_node',
          description: formDescription.trim(),
          config: {
            system_prompt: formSystemPrompt,
            user_prompt_template: formUserPromptTemplate,
            input_mapping: {},
            output_schema: {},
            output_key: ''
          },
          input_contract: { required: [], optional: [] },
          output_contract: { produced: [] },
          capabilities: [],
          tags: formTags.split(',').map(t => t.trim()).filter(Boolean),
          owner: 'system',
          status: 'active',
          reuse_eligible: true,
          provenance: {}
        };
        await graphWeaveApi.createNode(payload);
      }

      showEditModal = false;
      await fetchNodes();
    } catch (err) {
      modalError = err instanceof Error ? err.message : 'Failed to save node';
    } finally {
      modalLoading = false;
    }
  }

  async function handleDeleteNode(nodeId: string) {
    if (!confirm(`Are you sure you want to permanently delete custom node: ${nodeId}?`)) return;
    try {
      await graphWeaveApi.deleteNode(nodeId, tenantId);
      await fetchNodes();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Deletion failed');
    }
  }

  // Filter list locally for extra responsiveness
  let filteredNodes = $derived.by(() => {
    let result = nodes;
    if (searchQuery.trim()) {
      const q = searchQuery.toLowerCase();
      result = result.filter(n => 
        n.node_name.toLowerCase().includes(q) || 
        n.node_id.toLowerCase().includes(q) || 
        (n.description && n.description.toLowerCase().includes(q))
      );
    }
    if (tagFilter.trim()) {
      const tag = tagFilter.toLowerCase().trim();
      result = result.filter(n => (n.tags || []).some(t => t.toLowerCase() === tag));
    }
    return result;
  });
</script>

<div class="space-y-3">
  
  <!-- Search and Actions Bar -->
  <div class="flex flex-col gap-2 p-3 rounded-xl border border-slate-900 bg-slate-950/40 backdrop-blur-xl select-none shadow-sm relative z-10">
    <div class="flex items-center gap-1.5 w-full">
      <!-- Search Input -->
      <div class="relative flex-1">
        <span class="absolute inset-y-0 left-0 flex items-center pl-2.5 text-slate-500 pointer-events-none">
          <svg width="12" height="12" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
          </svg>
        </span>
        <input 
          type="text" 
          bind:value={searchQuery}
          placeholder="Search templates..."
          class="w-full rounded-lg pl-8 pr-2 py-1.5 text-[10px] text-slate-200 focus:outline-none focus:border-indigo-500 placeholder-slate-550 glass-input min-h-[30px]"
        />
      </div>

      <!-- Add Action -->
      <button 
        onclick={openCreateModal}
        class="p-2 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white shadow transition-all cursor-pointer flex items-center justify-center min-h-[30px] w-[30px]"
        title="Create Catalog Node"
        aria-label="Create Node"
      >
        <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
          <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
        </svg>
      </button>
    </div>
  </div>

  <!-- Main Nodes list -->
  {#if isLoading}
    <div class="py-6 text-center glass-slab rounded-xl select-none">
      <svg width="20" height="20" class="animate-spin text-indigo-500 mx-auto mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
        <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2" />
      </svg>
      <span class="text-[9px] text-slate-400 font-semibold tracking-wide">Syncing templates...</span>
    </div>
  {:else if error}
    <div class="py-6 text-center glass-slab rounded-xl border-rose-500/10">
      <span class="text-[9px] font-extrabold uppercase tracking-widest text-rose-455 block mb-1">Catalog Connection Failure</span>
      <span class="text-[9px] text-slate-550 font-mono block truncate px-2 select-text">{error}</span>
    </div>
  {:else if filteredNodes.length === 0}
    <div class="py-8 text-center glass-slab rounded-xl select-none">
      <span class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">No custom node templates</span>
      <button 
        onclick={openCreateModal}
        class="mt-2.5 px-3 py-1.5 text-[9px] font-bold text-indigo-400 border border-indigo-500/25 bg-indigo-500/5 hover:bg-indigo-500/10 rounded-lg transition-all cursor-pointer w-full"
      >
        Add Template
      </button>
    </div>
  {:else}
    <!-- Registry vertical list of small items -->
    <div class="flex flex-col gap-2 max-h-[30vh] overflow-y-auto custom-scrollbar pr-1">
      {#each filteredNodes as node}
        <div 
          role="button"
          tabindex="0"
          onclick={() => {
            if (onSelectTemplate) {
              onSelectTemplate(node);
            } else {
              openEditModal(node);
            }
          }}
          onkeydown={(e) => { 
            if (e.key === 'Enter' || e.key === ' ') { 
              e.preventDefault(); 
              if (onSelectTemplate) {
                onSelectTemplate(node);
              } else {
                openEditModal(node);
              }
            } 
          }}
          class="flex flex-col p-3 rounded-xl relative overflow-hidden cursor-pointer select-none glass-slab glass-slab-hover border-slate-900/60 text-left"
        >
          <!-- Node Header -->
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0">
              <h4 class="text-xs font-bold text-slate-200 leading-tight truncate">{node.node_name}</h4>
              <span class="text-[8px] text-slate-550 font-mono block truncate select-all">{node.node_id}</span>
            </div>
            
            <div class="flex items-center gap-1 shrink-0">
              {#if onSelectTemplate}
                <button
                  onclick={(e) => { e.stopPropagation(); onSelectTemplate(node); }}
                  class="p-1 rounded bg-indigo-500/10 hover:bg-indigo-500/20 border border-indigo-500/25 text-indigo-400 hover:text-indigo-300 transition-colors cursor-pointer flex items-center justify-center font-bold text-[8px] uppercase px-1.5 min-h-[22px]"
                  title="Apply Template"
                >
                  Apply
                </button>
              {/if}
              <button
                onclick={(e) => { e.stopPropagation(); openEditModal(node); }}
                class="p-1 rounded border border-slate-900 hover:border-indigo-500/30 hover:bg-indigo-500/5 text-slate-650 hover:text-indigo-450 transition-colors cursor-pointer flex items-center justify-center"
                title="Edit Prompts"
                aria-label="Edit Prompts"
              >
                <svg width="10" height="10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
              <button
                onclick={(e) => { e.stopPropagation(); handleDeleteNode(node.node_id); }}
                class="p-1 rounded border border-slate-900 hover:border-rose-500/30 hover:bg-rose-500/5 text-slate-650 hover:text-rose-455 transition-colors cursor-pointer flex items-center justify-center"
                title="Delete Node"
                aria-label="Delete Node"
              >
                <svg width="10" height="10" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
              </button>
            </div>
          </div>

          <!-- Description (one line) -->
          {#if node.description}
            <p class="text-[10px] text-slate-400 mt-1 leading-normal truncate">
              {node.description}
            </p>
          {/if}

          <!-- Type Indicator & Tags footer -->
          <div class="mt-2 flex items-center justify-between text-[8px] border-t border-slate-900/40 pt-1.5">
            <span class="text-indigo-400 font-extrabold uppercase tracking-widest">{node.type || 'agent_node'}</span>
            {#if node.tags && node.tags.length > 0}
              <span class="text-slate-500 truncate max-w-[100px]">{node.tags.join(', ')}</span>
            {/if}
          </div>

        </div>
      {/each}
    </div>
  {/if}

  <!-- Node Edit / Create Modal -->
  {#if showEditModal}
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-md animate-fade-in">
      <div class="relative w-full max-w-2xl bg-slate-900 border border-slate-800 rounded-2xl shadow-2xl flex flex-col max-h-[95vh] overflow-hidden">
        
        <div class="absolute top-0 inset-x-0 h-[3px] bg-gradient-to-r from-indigo-500 via-indigo-600 to-violet-600"></div>

        <!-- Header -->
        <div class="px-6 py-5 border-b border-slate-800 flex items-center justify-between select-none">
          <h3 class="text-sm font-extrabold font-display uppercase tracking-widest text-slate-200">
            {isEditing ? 'Configure Catalog Prompts' : 'Register Reusable Catalog Node'}
          </h3>
          <button onclick={() => showEditModal = false} class="text-slate-500 hover:text-slate-350 cursor-pointer p-1.5 rounded-lg hover:bg-slate-950/40 transition-colors flex items-center justify-center" title="Close" aria-label="Close">
            <svg width="22" height="22" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
              <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <!-- Body -->
        <div class="flex-1 overflow-y-auto p-6 space-y-4.5 text-left custom-scrollbar">
          {#if modalError}
            <div class="bg-rose-950/20 border border-rose-900/40 text-rose-350 p-3.5 rounded-xl text-xs leading-relaxed font-mono">
              <strong>Save Error:</strong> {modalError}
            </div>
          {/if}

          <div class="grid grid-cols-2 gap-4">
            <div class="space-y-1.5">
              <label for="nd-id" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">Node Identifier (unique ID)</label>
              <input 
                type="text" 
                id="nd-id"
                bind:value={formNodeId}
                disabled={isEditing}
                placeholder="e.g. portfolio-evaluator"
                class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 font-mono disabled:opacity-50 glass-input"
              />
            </div>
            <div class="space-y-1.5">
              <label for="nd-name" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">Display Name</label>
              <input 
                type="text" 
                id="nd-name"
                bind:value={formNodeName}
                placeholder="e.g. Portfolio Evaluator Node"
                class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 glass-input"
              />
            </div>
          </div>

          <div class="space-y-1.5">
            <label for="nd-desc" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">Description / Agent Purpose</label>
            <input 
              type="text" 
              id="nd-desc"
              bind:value={formDescription}
              placeholder="Performs financial allocation calculations, weights optimization, and scores risks."
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 glass-input"
            />
          </div>

          <div class="space-y-1.5">
            <label for="nd-sys" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">System Instruction Prompt (Behavior)</label>
            <textarea 
              id="nd-sys"
              bind:value={formSystemPrompt}
              rows="6"
              class="w-full rounded-xl p-3.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 leading-relaxed glass-input custom-scrollbar"
              placeholder="Define agent character rules, limits, tool execution contracts..."
            ></textarea>
          </div>

          <div class="space-y-1.5">
            <label for="nd-user" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">User Query Template (Context)</label>
            <textarea 
              id="nd-user"
              bind:value={formUserPromptTemplate}
              rows="4"
              class="w-full rounded-xl p-3.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 leading-relaxed glass-input custom-scrollbar"
              placeholder={"Interpolate context variables with braces: Evaluate risk metrics on {assets}..."}
            ></textarea>
          </div>

          <div class="space-y-1.5">
            <label for="nd-tags" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block select-none">Node Tags (comma separated)</label>
            <input 
              type="text" 
              id="nd-tags"
              bind:value={formTags}
              placeholder="e.g. math, portfolio, tools"
              class="w-full rounded-xl px-3 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 glass-input"
            />
          </div>
        </div>

        <!-- Footer -->
        <div class="px-6 py-4 border-t border-slate-800 bg-slate-950/20 flex justify-end gap-3 select-none">
          <button 
            onclick={() => showEditModal = false}
            class="px-4 py-2 text-xs font-bold text-slate-400 hover:text-slate-250 cursor-pointer"
          >
            Cancel
          </button>
          
          <button 
            onclick={handleSaveNode}
            disabled={modalLoading || !formNodeId || !formNodeName}
            class="px-5 py-2 text-xs font-extrabold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow shadow-indigo-600/10"
          >
            {#if modalLoading}
              Saving...
            {:else}
              Save Catalog Node
            {/if}
          </button>
        </div>

      </div>
    </div>
  {/if}

</div>
