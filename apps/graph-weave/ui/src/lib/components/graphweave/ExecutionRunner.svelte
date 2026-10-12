<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    graphWeaveApi,
    type WorkflowDetail,
    type ExecutionRun,
    type ExecutionStatusResponse,
  } from '../../graphweave/api';

  // Props using Svelte 5 runes
  let {
    tenantId = 'system',
    workflowId = null,
    workflowDetail = null,
    activeNodeId = $bindable(null),
  } = $props<{
    tenantId: string;
    workflowId: string | null;
    workflowDetail: WorkflowDetail | null;
    activeNodeId: string | null;
  }>();

  // Executions lists & status
  let runs = $state<ExecutionRun[]>([]);
  let selectedRunId = $state<string | null>(null);
  let selectedRunDetail = $state<ExecutionStatusResponse | null>(null);
  let isFetchingRuns = $state(false);
  let isFetchingDetail = $state(false);
  let isTriggering = $state(false);

  // Inputs for launching runs
  let useRawJson = $state(false);
  let rawJsonInputs = $state('{\n  "topic": "wealth management automation",\n  "depth": "quick"\n}');
  let formInputs = $state<Record<string, string>>({});
  let launchError = $state<string | null>(null);

  // Active polling timer reference
  let pollInterval: ReturnType<typeof setInterval> | null = null;

  // Derive input fields from workflow detail entry node
  let entryProperties = $derived.by(() => {
    if (!workflowDetail || !workflowDetail.definition?.nodes) return [];
    const entryNode = workflowDetail.definition.nodes.find((n: any) => n.type === 'entry');
    if (!entryNode || !entryNode.config?.properties) return [];

    return Object.entries(entryNode.config.properties).map(([key, val]: [string, any]) => ({
      key,
      type: val.type || 'string',
      description: val.description || '',
      required: entryNode.config?.required?.includes(key) || false,
    }));
  });

  // Reactively initialize form input structures when workflow properties change
  $effect(() => {
    const inputs: Record<string, string> = {};
    entryProperties.forEach((prop) => {
      inputs[prop.key] = '';
    });
    formInputs = inputs;
  });

  // Fetch runs reactively based on workflow selection
  $effect(() => {
    if (workflowId) {
      fetchRuns();
    } else {
      runs = [];
      selectedRunId = null;
      selectedRunDetail = null;
    }
  });

  // Check running status in runs and start polling if needed
  $effect(() => {
    const hasActiveRun = runs.some((r) => r.status === 'running' || r.status === 'queued');
    if (hasActiveRun) {
      startPolling();
    } else if (!selectedRunId || (selectedRunDetail && !['running', 'queued'].includes(selectedRunDetail.status))) {
      stopPolling();
    }
  });

  // Derive the active node from the latest events in selectedRunDetail
  $effect(() => {
    if (!selectedRunDetail || !['running', 'queued'].includes(selectedRunDetail.status)) {
      activeNodeId = null;
      return;
    }

    const events = selectedRunDetail.events || [];
    let currentActive: string | null = null;

    for (let i = events.length - 1; i >= 0; i--) {
      const evt = events[i];
      if (evt.type === 'node.started' && evt.data?.node_id) {
        currentActive = evt.data.node_id;
        break;
      }
    }

    activeNodeId = currentActive;
  });

  onMount(() => {
    if (workflowId) {
      fetchRuns();
    }
  });

  onDestroy(() => {
    stopPolling();
  });

  async function fetchRuns() {
    if (!workflowId) return;
    isFetchingRuns = true;
    try {
      const res = await graphWeaveApi.listExecutions(tenantId, workflowId);
      runs = res.runs || [];
    } catch (err) {
      console.error('Failed to list executions', err);
    } finally {
      isFetchingRuns = false;
    }
  }

  async function selectRun(runId: string) {
    selectedRunId = runId;
    isFetchingDetail = true;
    try {
      const res = await graphWeaveApi.getExecutionStatus(runId);
      selectedRunDetail = res;
    } catch (err) {
      console.error('Failed to get execution status', err);
    } finally {
      isFetchingDetail = false;
    }
  }

  function startPolling() {
    if (pollInterval) return;
    pollInterval = setInterval(async () => {
      if (selectedRunId) {
        try {
          const res = await graphWeaveApi.getExecutionStatus(selectedRunId);
          selectedRunDetail = res;

          // Update status in local runs list
          const runIndex = runs.findIndex((r) => r.run_id === selectedRunId);
          if (runIndex !== -1) {
            runs[runIndex].status = res.status;
            runs[runIndex].hop_count = res.hop_count;
          }

          if (!['running', 'queued'].includes(res.status)) {
            stopPolling();
            fetchRuns(); // refresh whole list once complete
          }
        } catch (e) {
          console.warn('Polling status failed', e);
        }
      } else {
        fetchRuns();
      }
    }, 850);
  }

  function stopPolling() {
    if (pollInterval) {
      clearInterval(pollInterval);
      pollInterval = null;
    }
  }

  async function handleLaunch() {
    if (!workflowId) return;
    launchError = null;
    isTriggering = true;

    let payload: Record<string, any> = {};
    try {
      if (useRawJson) {
        payload = JSON.parse(rawJsonInputs);
      } else {
        // Collect form fields and check required
        entryProperties.forEach((prop) => {
          const val = formInputs[prop.key];
          if (prop.required && (!val || !val.trim())) {
            throw new Error(`Required input parameter is missing: ${prop.key}`);
          }
          payload[prop.key] = val;
        });
      }

      const res = await graphWeaveApi.executeWorkflow(tenantId, workflowId, payload);
      await fetchRuns();

      // Auto select the new run and start monitoring
      selectedRunId = res.run_id;
      await selectRun(res.run_id);
      startPolling();
    } catch (err) {
      launchError = err instanceof Error ? err.message : 'Execution launch failed';
    } finally {
      isTriggering = false;
    }
  }

  async function handleCancel(runId: string) {
    if (!confirm('Are you sure you want to abort this active workflow execution?')) return;
    try {
      await graphWeaveApi.cancelExecution(runId);
      await fetchRuns();
      if (selectedRunId === runId) {
        await selectRun(runId);
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Cancellation failed');
    }
  }

  async function handleRecover(runId: string, threadId: string) {
    try {
      await graphWeaveApi.recoverExecution(runId, threadId);
      alert('Recovery request successfully dispatched! Polling updated thread state.');
      await fetchRuns();
      if (selectedRunId === runId) {
        await selectRun(runId);
      }
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Recovery failed');
    }
  }

  function formatTime(isoStr?: string): string {
    if (!isoStr) return '';
    try {
      const d = new Date(isoStr);
      return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false });
    } catch {
      return '';
    }
  }
</script>

<div class="grid grid-cols-1 lg:grid-cols-12 gap-6 relative z-10 items-stretch h-full min-h-0">
  <!-- Left Panel: Trigger & Runs History (5 Cols) -->
  <div class="lg:col-span-5 space-y-6 flex flex-col h-full min-h-0 overflow-hidden">
    <!-- Launch Box Card -->
    <div class="p-5 rounded-2xl glass-slab select-none">
      <div class="flex items-center justify-between mb-4 border-b border-slate-900/60 pb-3">
        <h3 class="text-xs font-extrabold font-display uppercase tracking-widest text-slate-200">Trigger Execution</h3>
        <button
          onclick={() => (useRawJson = !useRawJson)}
          class="text-[9px] font-extrabold text-indigo-400 hover:text-indigo-300 uppercase tracking-widest cursor-pointer hover:underline focus:outline-none"
        >
          {useRawJson ? 'Use Form UI' : 'Use Raw JSON'}
        </button>
      </div>

      {#if launchError}
        <div
          class="mb-4 bg-rose-950/20 border border-rose-900/40 text-rose-350 p-2.5 rounded-xl text-[10px] font-mono leading-relaxed select-text text-left"
        >
          <strong>Launch Error:</strong>
          {launchError}
        </div>
      {/if}

      {#if !workflowId}
        <div class="py-6 text-center text-slate-500 text-xs">Select a workflow first to load execution controls.</div>
      {:else}
        <div class="space-y-4 text-left select-text">
          {#if useRawJson}
            <div class="space-y-1.5">
              <label for="json-input" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500"
                >Inputs Payload (JSON)</label
              >
              <textarea
                id="json-input"
                bind:value={rawJsonInputs}
                rows="5"
                class="w-full rounded-xl px-3 py-2 text-xs text-indigo-300 font-mono focus:outline-none focus:border-indigo-500 glass-input"
              ></textarea>
            </div>
          {:else if entryProperties.length === 0}
            <div class="py-4 text-center text-slate-500 text-xs italic">
              No input properties declared. Dispatches immediate zero-arg run loop.
            </div>
          {:else}
            <div class="space-y-3.5">
              {#each entryProperties as prop}
                <div class="space-y-1.5">
                  <label for="input-{prop.key}" class="flex items-center justify-between select-none">
                    <span class="text-[10px] font-bold uppercase tracking-widest text-slate-400">
                      {prop.key}
                      {#if prop.required}
                        <span class="text-rose-500 font-bold ml-0.5">*</span>
                      {/if}
                    </span>
                    <span class="text-[8px] text-slate-500 font-mono uppercase tracking-wider">{prop.type}</span>
                  </label>
                  <input
                    type="text"
                    id="input-{prop.key}"
                    bind:value={formInputs[prop.key]}
                    placeholder={prop.description || `Value for ${prop.key}...`}
                    class="w-full rounded-xl px-3.5 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 glass-input"
                  />
                </div>
              {/each}
            </div>
          {/if}

          <button
            onclick={handleLaunch}
            disabled={isTriggering}
            class="w-full py-2.5 px-4 mt-3 text-xs font-bold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl shadow shadow-indigo-600/10 disabled:opacity-50 disabled:cursor-not-allowed transition-all cursor-pointer flex items-center justify-center gap-2 select-none"
          >
            {#if isTriggering}
              <svg
                width="14"
                height="14"
                class="w-3.5 h-3.5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2"
                />
              </svg>
              Initializing Agent Graph...
            {:else}
              <svg
                width="14"
                height="14"
                class="w-3.5 h-3.5"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z"
                />
              </svg>
              Trigger Execution Run
            {/if}
          </button>
        </div>
      {/if}
    </div>

    <!-- Execution History Runs -->
    <div class="p-5 rounded-2xl glass-slab flex-1 flex flex-col select-none overflow-hidden min-h-0">
      <div class="flex items-center justify-between mb-4 border-b border-slate-900/60 pb-3">
        <h3 class="text-xs font-extrabold font-display uppercase tracking-widest text-slate-200">Execution History</h3>
        <button
          onclick={fetchRuns}
          class="p-1.5 rounded-lg border border-slate-850 bg-slate-950/20 text-slate-400 hover:text-slate-250 transition-all cursor-pointer flex items-center justify-center shadow-sm"
          title="Refresh List"
          disabled={isFetchingRuns}
        >
          <svg
            width="14"
            height="14"
            class="w-3.5 h-3.5 {isFetchingRuns ? 'animate-spin text-indigo-400' : ''}"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2"
            />
          </svg>
        </button>
      </div>

      {#if !workflowId}
        <div class="py-12 text-center text-slate-650 text-xs">Select a workflow to view past execution history.</div>
      {:else if runs.length === 0}
        <div class="py-12 text-center text-slate-600 text-xs italic">No past runs logged. Trigger a run to begin.</div>
      {:else}
        <div class="space-y-2.5 overflow-y-auto pr-1 custom-scrollbar min-h-0 flex-1">
          {#each runs as run}
            {@const isSelected = selectedRunId === run.run_id}
            {@const isRunning = ['running', 'queued'].includes(run.status)}

            <div
              role="button"
              tabindex="0"
              onclick={() => selectRun(run.run_id)}
              onkeydown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault();
                  selectRun(run.run_id);
                }
              }}
              class="w-full flex items-center justify-between p-3.5 rounded-xl border text-left transition-all duration-200 cursor-pointer
                {isSelected
                ? 'bg-indigo-950/15 border-indigo-500 shadow shadow-indigo-500/5'
                : 'bg-slate-950/40 border-slate-850 hover:bg-slate-900/40 hover:border-slate-800'}"
            >
              <div class="min-w-0 pr-2 select-text">
                <div class="flex items-center gap-2">
                  <span class="text-[10px] text-slate-200 font-bold font-mono truncate block max-w-[125px]"
                    >{run.run_id}</span
                  >
                  <span
                    class="px-2 py-0.5 rounded text-[8px] font-extrabold uppercase tracking-widest border shrink-0
                    {run.status === 'completed' || run.status === 'success'
                      ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400 glow-entry'
                      : run.status === 'failed'
                        ? 'bg-rose-500/10 border-rose-500/20 text-rose-455 glow-exit'
                        : run.status === 'cancelled'
                          ? 'bg-slate-900 border-slate-850 text-slate-500'
                          : 'bg-amber-500/10 border-amber-500/20 text-amber-400 glow-branch animate-pulse'}"
                  >
                    {run.status}
                  </span>
                </div>
                <div class="flex items-center gap-2 mt-1.5 text-[9px] text-slate-500 font-mono select-none">
                  <span>Hops: <span class="text-slate-400">{run.hop_count}</span></span>
                  <span>•</span>
                  <span class="truncate">Thread: {run.thread_id.slice(0, 8)}...</span>
                </div>
              </div>

              <!-- History Action Trigger buttons -->
              <div class="shrink-0 flex items-center gap-1.5 select-none">
                {#if isRunning}
                  <button
                    onclick={(e) => {
                      e.stopPropagation();
                      handleCancel(run.run_id);
                    }}
                    class="p-1.5 rounded-lg border border-slate-850 bg-slate-950/30 text-rose-400 hover:bg-rose-500/15 hover:border-rose-500/30 cursor-pointer shadow-sm flex items-center justify-center"
                    title="Abort Execution"
                  >
                    <svg
                      width="14"
                      height="14"
                      class="w-3.5 h-3.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2.5"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                      />
                    </svg>
                  </button>
                {:else if run.status === 'failed' && run.thread_id}
                  <button
                    onclick={(e) => {
                      e.stopPropagation();
                      handleRecover(run.run_id, run.thread_id);
                    }}
                    class="p-1.5 rounded-lg border border-slate-850 bg-slate-950/30 text-indigo-400 hover:bg-indigo-500/15 hover:border-indigo-500/30 cursor-pointer shadow-sm flex items-center justify-center"
                    title="Recover Checkpoint"
                  >
                    <svg
                      width="14"
                      height="14"
                      class="w-3.5 h-3.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2.5"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2"
                      />
                    </svg>
                  </button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  </div>

  <!-- Right Panel: Live Terminal Trace Logs (7 Cols) -->
  <div class="lg:col-span-7 flex flex-col h-full min-h-0 items-stretch overflow-hidden">
    <!-- Log Header -->
    <div
      class="flex items-center justify-between p-4.5 border border-slate-900 bg-slate-950 rounded-t-2xl shadow-lg relative select-none"
    >
      <div class="absolute top-0 inset-x-0 h-[2px] bg-gradient-to-r from-indigo-500 via-indigo-600 to-violet-600"></div>

      <div class="flex items-center gap-2.5">
        <span
          class="w-2 h-2 rounded-full
          {selectedRunDetail?.status === 'running' || selectedRunDetail?.status === 'queued'
            ? 'bg-amber-500 animate-pulse glow-branch'
            : selectedRunDetail?.status === 'completed' || selectedRunDetail?.status === 'success'
              ? 'bg-emerald-500 glow-entry'
              : selectedRunDetail?.status === 'failed'
                ? 'bg-rose-500 glow-exit'
                : 'bg-slate-700'}"
        ></span>
        <h3 class="text-xs font-extrabold font-display uppercase tracking-widest text-slate-200">
          Terminal Trace Logs
        </h3>
      </div>

      {#if selectedRunDetail}
        <div class="flex items-center gap-3 text-[9px] font-mono text-slate-500 select-all">
          <span>Run: <span class="text-indigo-400 font-bold">{selectedRunDetail.run_id.slice(0, 13)}...</span></span>
          {#if ['running', 'queued'].includes(selectedRunDetail.status)}
            <span class="text-indigo-400 font-extrabold animate-pulse">● LIVE STEAM</span>
          {/if}
        </div>
      {/if}
    </div>

    <!-- Retro Terminal Shell Screen -->
    <div
      class="flex-1 p-5 border border-slate-900 border-t-0 bg-[#020306]/98 rounded-b-2xl font-mono text-xs text-slate-300 overflow-y-auto space-y-4 shadow-inner custom-scrollbar select-text text-left min-h-0"
    >
      {#if isFetchingDetail}
        <div class="py-32 text-center text-slate-500 space-y-3 select-none">
          <svg
            width="24"
            height="24"
            class="w-6 h-6 animate-spin text-indigo-500 mx-auto"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            stroke-width="2.5"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2"
            />
          </svg>
          <span class="text-[10px] uppercase font-bold tracking-widest">Streaming live timeline trace data...</span>
        </div>
      {:else if !selectedRunDetail}
        <div class="py-32 text-center text-slate-500 max-w-xs mx-auto space-y-3.5 select-none">
          <div
            class="w-12 h-12 rounded-xl bg-slate-950/80 border border-slate-900 flex items-center justify-center text-slate-700 mx-auto shadow-inner"
          >
            <svg width="24" height="24" class="stroke-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          </div>
          <div>
            <span class="font-extrabold text-slate-400 block text-xs uppercase tracking-widest">Terminal Standby</span>
            <p class="text-[9px] text-slate-650 mt-1 leading-normal">
              Select an execution run from the history tree or deploy a new run parameters model to initialize log
              streams.
            </p>
          </div>
        </div>
      {:else}
        <!-- Live timeline events logs -->
        <div class="space-y-4">
          <!-- Terminal initialize header -->
          <div class="flex items-start gap-3 border-l-2 border-indigo-500/40 pl-3.5 py-0.5">
            <span class="text-slate-600 text-[10px] select-none pt-0.5">00:00:00</span>
            <div class="flex-1">
              <span class="text-indigo-400 font-extrabold uppercase tracking-widest text-[8px] block"
                >System container start</span
              >
              <span class="text-slate-200 font-bold"
                >Successfully spawned process run ID: <span class="text-indigo-300 font-mono"
                  >{selectedRunDetail.run_id}</span
                ></span
              >
            </div>
          </div>

          {#each selectedRunDetail.events || [] as event}
            {@const time = formatTime(event.timestamp)}

            {#if event.type === 'request.started'}
              <div class="flex items-start gap-3 border-l-2 border-indigo-500/60 pl-3.5 py-0.5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-indigo-400 font-extrabold uppercase tracking-widest text-[8px] block"
                    >Scope validation success</span
                  >
                  <span class="text-slate-300"
                    >Tenant: <span class="text-indigo-300 font-bold">{event.data?.tenant_id}</span> • Graph:
                    <span class="text-indigo-300 font-bold">{event.data?.workflow_id}</span></span
                  >
                </div>
              </div>
            {:else if event.type === 'node.started'}
              <div
                class="flex items-start gap-3 border-l-2 border-violet-500 pl-3.5 py-1 bg-violet-950/10 rounded-r-xl pr-3 border-r border-t border-b border-slate-900/30"
              >
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <div class="flex items-center justify-between select-none">
                    <span class="text-violet-400 font-extrabold uppercase tracking-widest text-[8px]"
                      >Node traversal start</span
                    >
                    <span
                      class="px-1.5 py-0.25 bg-slate-950 border border-slate-850 text-[7px] text-slate-500 rounded font-sans font-bold uppercase tracking-wider"
                      >{event.data?.node_type}</span
                    >
                  </div>
                  <span class="text-slate-200 font-bold block mt-0.5">Visiting: {event.data?.node_id}</span>

                  {#if event.data?.input && Object.keys(event.data.input).length > 0}
                    <details class="mt-2 cursor-pointer select-none">
                      <summary
                        class="text-[8px] text-slate-500 hover:text-slate-400 uppercase font-bold tracking-widest"
                        >Inspect inputs dictionary</summary
                      >
                      <pre
                        class="mt-1.5 p-2 bg-black/60 border border-slate-900 rounded-xl text-[10px] text-indigo-300 overflow-x-auto whitespace-pre-wrap leading-relaxed select-text">{JSON.stringify(
                          event.data.input,
                          null,
                          2,
                        )}</pre>
                    </details>
                  {/if}
                </div>
              </div>
            {:else if event.type === 'agent.request'}
              <div class="flex items-start gap-3 border-l-2 border-slate-700 pl-3.5 py-0.5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1 text-slate-400 leading-relaxed">
                  <span class="text-slate-500 font-extrabold uppercase tracking-widest text-[8px] block"
                    >LLM network query</span
                  >
                  <span
                    >Model dispatch request to client <span class="text-slate-200 font-bold"
                      >{event.data?.provider}/{event.data?.model}</span
                    >
                    with <span class="text-indigo-400 font-bold">{event.data?.tool_count || 0}</span> allowed MCP tools.</span
                  >
                </div>
              </div>
            {:else if event.type === 'agent.tool_calls'}
              <div class="flex items-start gap-3 border-l-2 border-slate-700 pl-3.5 py-0.5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1 text-slate-400">
                  <span class="text-amber-500/80 font-extrabold uppercase tracking-widest text-[8px] block"
                    >Model routing loop</span
                  >
                  <span
                    >Execution graph contains <span class="text-amber-400 font-bold">{event.data?.count}</span> tool orchestration
                    calls. Processing...</span
                  >
                </div>
              </div>
            {:else if event.type === 'tool.started'}
              <div class="flex items-start gap-3 border-l-2 border-amber-500/50 pl-3.5 py-1 bg-amber-950/5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-amber-500/70 font-extrabold uppercase tracking-widest text-[7.5px] block"
                    >MCP tool trigger</span
                  >
                  <span class="text-slate-300"
                    >Tool: <span class="text-amber-400 font-bold font-mono">{event.data?.tool}</span>
                    <span class="text-slate-600 font-mono">({event.data?.id?.slice(0, 6)})</span></span
                  >
                  {#if event.data?.input}
                    <pre
                      class="mt-1.5 p-2 bg-black/40 border border-slate-900 rounded-xl text-[10px] text-slate-400 overflow-x-auto whitespace-pre-wrap leading-relaxed select-text">arguments: {JSON.stringify(
                        event.data.input,
                        null,
                        2,
                      )}</pre>
                  {/if}
                </div>
              </div>
            {:else if event.type === 'tool.completed'}
              <div class="flex items-start gap-3 border-l-2 border-emerald-600 pl-3.5 py-1 bg-emerald-950/5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-emerald-400 font-extrabold uppercase tracking-widest text-[7.5px] block"
                    >MCP tool success</span
                  >
                  <span class="text-slate-300"
                    >Tool: <span class="text-emerald-300 font-bold font-mono">{event.data?.tool}</span> completed successfully.</span
                  >
                  {#if event.data?.result}
                    <details class="mt-1.5 cursor-pointer select-none">
                      <summary
                        class="text-[8px] text-slate-500 hover:text-slate-400 font-bold uppercase tracking-widest"
                        >Inspect result dictionary</summary
                      >
                      <pre
                        class="mt-1.5 p-2 bg-black/60 border border-slate-900 rounded-xl text-[10px] text-emerald-300 overflow-x-auto whitespace-pre-wrap leading-relaxed select-text">{JSON.stringify(
                          event.data.result,
                          null,
                          2,
                        )}</pre>
                    </details>
                  {/if}
                </div>
              </div>
            {:else if event.type === 'tool.failed'}
              <div class="flex items-start gap-3 border-l-2 border-rose-600 pl-3.5 py-1.5 bg-rose-950/10">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-rose-455 font-extrabold uppercase tracking-widest text-[7.5px] block"
                    >MCP tool failure</span
                  >
                  <span class="text-slate-300"
                    >Tool: <span class="text-rose-400 font-bold font-mono">{event.data?.tool}</span> failed executing tasks.</span
                  >
                  <p class="text-rose-300 mt-1 font-mono leading-relaxed select-text">{event.data?.error}</p>
                </div>
              </div>
            {:else if event.type === 'node.completed'}
              <div
                class="flex items-start gap-3 border-l-2 border-emerald-500 pl-3.5 py-1 bg-emerald-950/10 rounded-r-xl pr-3 border-r border-t border-b border-slate-900/30"
              >
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-emerald-400 font-extrabold uppercase tracking-widest text-[8px] block"
                    >Node traversal completed</span
                  >
                  <span class="text-slate-200 font-bold block mt-0.5">Visited: {event.data?.node_id}</span>
                  <span class="text-slate-500 font-mono text-[8px] block mt-1"
                    >Traversing hop order depth: {event.data?.hop_count}</span
                  >

                  {#if event.data?.output && Object.keys(event.data.output).length > 0}
                    <details class="mt-2 cursor-pointer select-none">
                      <summary
                        class="text-[8px] text-slate-500 hover:text-slate-400 uppercase font-bold tracking-widest"
                        >Inspect outputs context</summary
                      >
                      <pre
                        class="mt-1.5 p-2 bg-black/60 border border-slate-900 rounded-xl text-[10px] text-indigo-300 overflow-x-auto whitespace-pre-wrap leading-relaxed select-text">{JSON.stringify(
                          event.data.output,
                          null,
                          2,
                        )}</pre>
                    </details>
                  {/if}
                </div>
              </div>
            {:else if event.type === 'node.failed'}
              <div
                class="flex items-start gap-3 border-l-2 border-rose-500 pl-3.5 py-1 bg-rose-950/20 rounded-r-xl pr-3 border-r border-t border-b border-slate-900/30"
              >
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-rose-455 font-extrabold uppercase tracking-widest text-[8px] block"
                    >Node traversal failure</span
                  >
                  <span class="text-slate-200 font-bold block mt-0.5">Visited: {event.data?.node_id}</span>
                  <pre
                    class="mt-2 p-2.5 bg-black/75 border border-rose-950/50 rounded-xl text-[10px] text-rose-350 overflow-x-auto whitespace-pre-wrap leading-relaxed select-text">{event
                      .data?.error}</pre>
                </div>
              </div>
            {:else if event.type === 'edge_route'}
              <div class="flex items-start gap-3 border-l-2 border-indigo-500/40 pl-3.5 py-0.5">
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1 text-slate-500">
                  <span class="text-indigo-400/80 font-extrabold uppercase tracking-widest text-[7.5px] block"
                    >Edge gate routed</span
                  >
                  <span
                    >Connection traversed: <span class="text-indigo-300 font-semibold font-mono"
                      >{event.data?.from}</span
                    >
                    ➔ <span class="text-indigo-300 font-semibold font-mono">{event.data?.to}</span></span
                  >
                </div>
              </div>
            {:else if event.type === 'request.completed' || event.type === 'workflow.completed'}
              <div
                class="flex items-start gap-3 border-l-2 border-emerald-500 pl-3.5 py-1.5 bg-emerald-950/15 rounded-r-xl pr-3 border-r border-t border-b border-slate-900/30"
              >
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-emerald-400 font-extrabold uppercase tracking-widest text-[9px] block"
                    >Orchestrator core finalized: SUCCESS</span
                  >
                  <span class="text-slate-300 text-xs">Run container successfully processed all dynamic nodes.</span>

                  {#if event.data?.output && Object.keys(event.data.output).length > 0}
                    <div class="mt-3.5 space-y-1.5 select-text">
                      <span class="text-[8px] font-extrabold text-slate-500 uppercase tracking-widest block select-none"
                        >Final return payload values</span
                      >
                      <pre
                        class="p-3.5 bg-black/85 border border-emerald-900/40 rounded-xl text-xs text-emerald-300 overflow-x-auto whitespace-pre-wrap leading-relaxed shadow-lg font-mono">{JSON.stringify(
                          event.data.output,
                          null,
                          2,
                        )}</pre>
                    </div>
                  {/if}
                </div>
              </div>
            {:else if event.type === 'request.failed'}
              <div
                class="flex items-start gap-3 border-l-2 border-rose-500 pl-3.5 py-1.5 bg-rose-950/20 rounded-r-xl pr-3 border-r border-t border-b border-slate-900/30"
              >
                <span class="text-slate-600 text-[10px] select-none pt-0.5">{time}</span>
                <div class="flex-1">
                  <span class="text-rose-455 font-extrabold uppercase tracking-widest text-[9px] block"
                    >Orchestrator core finalized: FAILURE</span
                  >
                  <p class="text-rose-300 mt-1.5 leading-relaxed font-mono select-text">{event.data?.error}</p>
                </div>
              </div>
            {/if}
          {/each}

          <!-- Polling active blinking dot loading bar inside retro screen -->
          {#if ['running', 'queued'].includes(selectedRunDetail.status)}
            <div class="flex items-center gap-2.5 pl-6 py-2 text-indigo-400 animate-pulse text-[11px] select-none">
              <svg
                width="14"
                height="14"
                class="w-3.5 h-3.5 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                stroke-width="2.5"
              >
                <path
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  d="M4 4v5h.582m15.356 2A8.001 8.001 0 1121.21 8H18.2"
                />
              </svg>
              <span class="font-bold tracking-wide">Awaiting active data packet traversals...</span>
            </div>
          {/if}
        </div>
      {/if}
    </div>
  </div>
</div>
