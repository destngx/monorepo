<script lang="ts">
  import { graphWeaveApi } from '$lib/graphweave/api';

  import WorkflowRegistry from '$lib/components/graphweave/WorkflowRegistry.svelte';
  import NodeList from '$lib/components/graphweave/NodeList.svelte';
  import WorkflowInspector from '$lib/components/graphweave/WorkflowInspector.svelte';
  import ExecutionRunner from '$lib/components/graphweave/ExecutionRunner.svelte';
  // Center column canvas tab mode
  let centerMode = $state('design');

  // Shared reactive workspace states
  let tenantId = $state('system');
  let selectedWorkflowId = $state(null);
  let selectedWorkflowDetail = $state(null);
  let selectedNodeId = $state(null);

  // Execution tracking state (updates the DAG live!)
  let activeNodeId = $state(null);

  // Fetch full details whenever workflow is selected
  $effect(() => {
    if (selectedWorkflowId && tenantId) {
      fetchWorkflowDetails();
    } else {
      selectedWorkflowDetail = null;
      selectedNodeId = null;
    }
  });

  async function fetchWorkflowDetails() {
    if (!selectedWorkflowId) return;
    try {
      const res = await graphWeaveApi.getWorkflow(selectedWorkflowId, tenantId);
      selectedWorkflowDetail = res;

      // Auto-select the first agent_node or orchestrator by default so prompt boxes are immediately visible
      if (res.definition?.nodes && res.definition.nodes.length > 0) {
        const firstAgent = res.definition.nodes.find((n) => n.type === 'agent_node' || n.type === 'orchestrator');
        if (firstAgent) {
          selectedNodeId = firstAgent.id;
        } else {
          // Fallback to first node in the array
          selectedNodeId = res.definition.nodes[0].id;
        }
      }
    } catch (err) {
      console.error('Failed to fetch full workflow structure details', err);
    }
  }

  // Derive active node configuration object for the inspector panel
  let selectedNode = $derived.by(() => {
    if (!selectedWorkflowDetail || !selectedNodeId) return null;
    return selectedWorkflowDetail.definition.nodes.find((n) => n.id === selectedNodeId) || null;
  });

  // Handle Select workflow callback
  function handleSelectWorkflow(workflowId) {
    selectedWorkflowId = workflowId;
    selectedNodeId = null;
  }

  // Save changes from Workflow Inspector back to the API database
  async function handleSaveNode(updatedNode) {
    if (!selectedWorkflowDetail || !selectedWorkflowId) return;

    try {
      const updatedNodes = selectedWorkflowDetail.definition.nodes.map((n) =>
        n.id === updatedNode.id ? updatedNode : n,
      );

      const payload = {
        definition: {
          ...selectedWorkflowDetail.definition,
          nodes: updatedNodes,
        },
      };

      await graphWeaveApi.updateWorkflow(selectedWorkflowId, tenantId, payload);
      await fetchWorkflowDetails();
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Failed to persist prompt and node configurations');
    }
  }

  // Delete node from active workflow definition schema
  async function handleDeleteNode(nodeId) {
    if (!selectedWorkflowDetail || !selectedWorkflowId) return;
    if (!window.confirm(`Are you sure you want to remove node '${nodeId}' from the workflow schema layout?`)) return;

    try {
      const filteredNodes = selectedWorkflowDetail.definition.nodes.filter((n) => n.id !== nodeId);
      const filteredEdges = selectedWorkflowDetail.definition.edges.filter((e) => e.from !== nodeId && e.to !== nodeId);

      const payload = {
        definition: {
          ...selectedWorkflowDetail.definition,
          nodes: filteredNodes,
          edges: filteredEdges,
        },
      };

      if (selectedNodeId === nodeId) selectedNodeId = null;

      await graphWeaveApi.updateWorkflow(selectedWorkflowId, tenantId, payload);
      await fetchWorkflowDetails();
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Node removal failed');
    }
  }

  // Append new node to workflow definition schema
  async function handleAddNode(newNode) {
    if (!selectedWorkflowDetail || !selectedWorkflowId) return;

    try {
      // Append node
      const updatedNodes = [...selectedWorkflowDetail.definition.nodes, newNode];

      // Attempt to auto-connect to make connecting node DAG elements easier
      let updatedEdges = [...selectedWorkflowDetail.definition.edges];
      const count = selectedWorkflowDetail.definition.nodes.length;
      if (count > 0) {
        const prev = selectedWorkflowDetail.definition.nodes[count - 1];
        if (prev.id !== 'exit' && newNode.id !== 'entry') {
          updatedEdges.push({ from: prev.id, to: newNode.id });
        }
      }

      const payload = {
        definition: {
          ...selectedWorkflowDetail.definition,
          nodes: updatedNodes,
          edges: updatedEdges,
        },
      };

      selectedNodeId = newNode.id;

      await graphWeaveApi.updateWorkflow(selectedWorkflowId, tenantId, payload);
      await fetchWorkflowDetails();
    } catch (err) {
      window.alert(err instanceof Error ? err.message : 'Adding node failed');
    }
  }
</script>

<!-- Consolidated 3-Column Workspace Grid Layout -->
<div
  class="flex-1 flex flex-col xl:flex-row gap-6 p-6 h-[calc(100vh-73px)] w-full items-stretch relative z-10 max-w-[1700px] mx-auto overflow-hidden min-h-0"
>
  <!-- LEFT COLUMN PANEL (Workflows + Active nodes + Collapsible catalog Templates) -->
  <aside
    class="w-full xl:w-[320px] shrink-0 flex flex-col gap-5 h-full overflow-y-auto custom-scrollbar select-none pr-1"
  >
    <!-- Section 1: Workflow registry vertical select explorer -->
    <div class="glass-slab p-5 rounded-2xl flex flex-col gap-4">
      <WorkflowRegistry bind:tenantId bind:selectedWorkflowId onSelectWorkflow={handleSelectWorkflow} />
    </div>
  </aside>

  <!-- CENTER SPACE (Flex-1, DAG graphical canvas + live execution toggle) -->
  <section class="flex-1 flex flex-col gap-6 min-w-0 h-full overflow-hidden">
    {#if !selectedWorkflowId}
      <!-- Empty Onboarding Guide -->
      <div
        class="flex-1 flex items-center justify-center py-24 text-center glass-slab rounded-2xl p-8 border-indigo-500/10"
      >
        <div class="max-w-md space-y-6">
          <div
            class="w-16 h-16 rounded-2xl bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center mx-auto text-indigo-400 shadow-lg shadow-indigo-500/15"
          >
            <svg width="32" height="32" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="1.5">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M9 20l-5.447-2.724A2 2 0 012.553 15.5V8.5a2 2 0 011-1.736L9 3.5m0 16.5V10m0 10l5.447-2.724a2 2 0 001-1.736V8.5a2 2 0 00-1-1.736L9 3.5m0 6.5L3.5 7.5m5.5 2.5l5.5-2.5M9 3.5l5.5 2.5m0 0L9 8.5M9 8.5L3.5 6"
              />
            </svg>
          </div>
          <div class="space-y-2">
            <h2 class="text-base font-extrabold font-display text-white tracking-wider uppercase">
              Orchestration Canvas Unloaded
            </h2>
            <p class="text-xs text-slate-400 leading-relaxed max-w-sm mx-auto">
              Please choose an active AI multi-agent orchestration path from the left sidebar registry to mount the
              canvas visualizer and run loop debugger.
            </p>
          </div>

          <div
            class="p-4 rounded-xl bg-slate-950/40 border border-slate-900 text-[10px] text-slate-500 font-mono text-left max-w-sm mx-auto leading-relaxed"
          >
            <span class="text-indigo-400 font-bold block mb-1">Developer Quickstart:</span>
            1. Select 'system' or 'default' context registry scope.<br />
            2. Register a new graph blueprint definition using GraphWeave JSON Spec.<br />
            3. Edit node prompts, parameters, and ReAct loops live in the inspector.
          </div>
        </div>
      </div>
    {:else if selectedWorkflowDetail}
      <!-- Design Canvas vs Execution mode segmented selector header -->
      <div
        class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-900 pb-4 relative z-10 select-none shrink-0"
      >
        <div class="segmented-control">
          <button
            onclick={() => (centerMode = 'design')}
            class="px-4 py-2 text-[10px] font-extrabold uppercase tracking-widest rounded-xl transition-all duration-200 cursor-pointer tap-target flex items-center justify-center gap-1.5
                {centerMode === 'design'
              ? 'bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 shadow-sm'
              : 'text-slate-500 hover:text-slate-300 border border-transparent'}"
          >
            <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z"
              />
            </svg>
            Design Canvas
          </button>

          <button
            onclick={() => (centerMode = 'execution')}
            class="px-4 py-2 text-[10px] font-extrabold uppercase tracking-widest rounded-xl transition-all duration-200 cursor-pointer tap-target flex items-center justify-center gap-1.5
                {centerMode === 'execution'
              ? 'bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 shadow-sm'
              : 'text-slate-500 hover:text-slate-350 border border-transparent'}"
          >
            <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
              />
              <path stroke-linecap="round" stroke-linejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            Live Execution
          </button>
        </div>

        <!-- Scope Context summary tags -->
        <div class="flex items-center gap-3">
          <div
            class="flex items-center gap-2.5 px-3.5 py-1.5 rounded-xl bg-slate-950/40 border border-slate-900 shadow-inner"
          >
            <span class="w-1.5 h-1.5 rounded-full bg-indigo-500 animate-pulse"></span>
            <span class="text-[8px] font-extrabold text-slate-500 uppercase tracking-widest font-sans"
              >Active Graph:</span
            >
            <span class="text-[9px] font-bold font-mono text-indigo-300 uppercase tracking-wider"
              >{selectedWorkflowDetail.name} ({selectedWorkflowDetail.version})</span
            >
          </div>
        </div>
      </div>

      <!-- Mode panels -->
      {#if centerMode === 'design'}
        <!-- Design mode: Full interactive DAG visualizer -->
        <div class="flex-1 flex flex-col gap-3 min-h-0 h-full overflow-hidden">
          <div class="flex items-center justify-between px-1 shrink-0">
            <div class="text-left">
              <h3 class="text-[10px] font-extrabold font-display uppercase tracking-widest text-slate-400">
                Interactive DAG Visualizer
              </h3>
              <span class="text-[9px] text-slate-400 leading-normal block mt-0.5"
                >Click any active node card below to open its behaviors and prompt setups in the right inspector.</span
              >
            </div>
            <span
              class="text-[8.5px] bg-slate-950 border border-slate-900 px-3 py-1 rounded-full font-mono text-indigo-400 font-bold"
            >
              Total Nodes: {selectedWorkflowDetail.definition?.nodes?.length || 0}
            </span>
          </div>

          <div class="flex-1 min-h-0 relative">
            <NodeList
              bind:nodes={selectedWorkflowDetail.definition.nodes}
              bind:edges={selectedWorkflowDetail.definition.edges}
              bind:selectedNodeId
              {activeNodeId}
            />
          </div>
        </div>
      {:else}
        <!-- Execution mode: Read-only DAG flow monitor + interactive runner console -->
        <div class="flex-1 flex flex-col gap-4 animate-fade-in min-h-0 h-full overflow-hidden">
          <div class="flex-[3] flex flex-col gap-2 min-h-0">
            <div class="flex items-center justify-between px-1 select-none shrink-0">
              <span class="text-[10px] font-extrabold font-display uppercase tracking-widest text-slate-400"
                >Live Orchestration Flow Map</span
              >
              {#if activeNodeId}
                <div
                  class="flex items-center gap-2 text-[10px] text-emerald-400 font-semibold animate-pulse font-mono bg-emerald-950/20 px-3 py-1 rounded-full border border-emerald-900/30 shadow-sm shadow-emerald-500/5"
                >
                  <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-heart-pulse"></span>
                  <span>Processing Node: {activeNodeId}</span>
                </div>
              {/if}
            </div>

            <div class="flex-1 min-h-0 relative">
              <NodeList
                bind:nodes={selectedWorkflowDetail.definition.nodes}
                bind:edges={selectedWorkflowDetail.definition.edges}
                selectedNodeId={null}
                {activeNodeId}
              />
            </div>
          </div>

          <!-- Action dashboard terminal console -->
          <div class="flex-[4] min-h-0 flex flex-col">
            <ExecutionRunner
              {tenantId}
              workflowId={selectedWorkflowId}
              workflowDetail={selectedWorkflowDetail}
              bind:activeNodeId
            />
          </div>
        </div>
      {/if}
    {:else}
      <!-- Loading spinner for active workflow details -->
      <div class="flex-1 flex flex-col items-center justify-center py-24 select-none">
        <svg
          width="24"
          height="24"
          class="animate-spin text-indigo-500 mb-2"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="2.5"
        >
          <path stroke-linecap="round" stroke-linejoin="round" d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2" />
        </svg>
        <span class="text-[10px] text-slate-500 font-semibold tracking-wider uppercase"
          >Loading Orchestrator Graph...</span
        >
      </div>
    {/if}
  </section>

  <!-- RIGHT COLUMN PANEL (Node Configuration inspector) -->
  <aside class="w-full xl:w-[380px] shrink-0 flex flex-col h-full overflow-hidden">
    <WorkflowInspector
      node={selectedNode}
      {tenantId}
      onSave={handleSaveNode}
      onDelete={handleDeleteNode}
      onAddNode={handleAddNode}
    />
  </aside>
</div>
