<script lang="ts">
  interface Node {
    id: string;
    type: string;
    display_name?: string;
    description?: string;
    config?: Record<string, any>;
  }

  interface Edge {
    from: string;
    to: string;
    condition?: string;
    config?: { label?: string };
  }

  // Props using Svelte 5 runes, identical to DagVisualizer for zero-risk drop-in swap
  let { 
    nodes = $bindable([]), 
    edges = $bindable([]), 
    selectedNodeId = $bindable(null),
    activeNodeId = null,
    onSelectNode = (node: any) => {}
  } = $props<{
    nodes: Node[];
    edges: Edge[];
    selectedNodeId: string | null;
    activeNodeId?: string | null;
    onSelectNode?: (node: Node) => void;
  }>();

  // Search filter query state
  let searchQuery = $state('');

  // Topologically sorted ordered nodes
  let orderedNodes = $derived.by(() => {
    if (!nodes || nodes.length === 0) return [];

    const ranks = new Map<string, number>();
    const incoming = new Map<string, string[]>();

    // Initialise adjacency maps
    nodes.forEach((n: Node) => {
      incoming.set(n.id, []);
    });

    edges.forEach((e: Edge) => {
      if (incoming.has(e.to)) incoming.get(e.to)!.push(e.from);
    });

    // Topological rank computation
    let changed = true;
    let iterations = 0;
    
    nodes.forEach((n: Node) => {
      if (n.type === 'entry') ranks.set(n.id, 0);
      else ranks.set(n.id, 1);
    });

    while (changed && iterations < 100) {
      changed = false;
      iterations++;
      
      nodes.forEach((n: Node) => {
        if (n.type === 'entry') return;

        const parents = incoming.get(n.id) || [];
        if (parents.length === 0) {
          if (ranks.get(n.id) !== 1) {
            ranks.set(n.id, 1);
            changed = true;
          }
          return;
        }

        let maxParentRank = -1;
        parents.forEach(p => {
          const pRank = ranks.get(p) ?? 0;
          if (pRank > maxParentRank) maxParentRank = pRank;
        });

        const newRank = maxParentRank + 1;
        if (ranks.get(n.id) !== newRank) {
          ranks.set(n.id, newRank);
          changed = true;
        }
      });
    }

    // Sort list by rank primarily, then by ID alphabetically
    const sorted = [...nodes].sort((a, b) => {
      const rA = ranks.get(a.id) ?? 99;
      const rB = ranks.get(b.id) ?? 99;
      if (rA !== rB) return rA - rB;
      return a.id.localeCompare(b.id);
    });

    // Filter by live text search query
    const q = searchQuery.trim().toLowerCase();
    if (!q) return sorted;

    return sorted.filter(n => 
      n.id.toLowerCase().includes(q) || 
      (n.display_name && n.display_name.toLowerCase().includes(q)) ||
      n.type.toLowerCase().includes(q)
    );
  });

  // Hashing logic to generate beautiful HSL colors consistent with visual theme
  function getNodeColorMeta(nodeId: string, nodeType: string) {
    if (nodeType === 'entry') {
      return { 
        border: 'rgba(16, 185, 129, 0.25)', 
        bg: 'rgba(240, 253, 244, 0.95)', 
        text: 'rgb(6, 95, 70)', 
        accent: 'rgb(16, 185, 129)' 
      };
    }
    if (nodeType === 'exit') {
      return { 
        border: 'rgba(244, 63, 94, 0.25)', 
        bg: 'rgba(255, 241, 242, 0.95)', 
        text: 'rgb(159, 18, 57)', 
        accent: 'rgb(244, 63, 94)' 
      };
    }
    
    // Stable hash seed from node ID string
    let hash = 0;
    for (let i = 0; i < nodeId.length; i++) {
      hash = nodeId.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = Math.abs(hash) % 360;
    
    return {
      border: `hsla(${hue}, 60%, 55%, 0.2)`,
      bg: `hsla(${hue}, 85%, 98%, 0.96)`, 
      text: `hsl(${hue}, 75%, 28%)`, 
      accent: `hsl(${hue}, 70%, 48%)`
    };
  }

  function handleSelectNode(node: Node) {
    selectedNodeId = node.id;
    onSelectNode(node);
  }
</script>

<div class="relative w-full h-full flex flex-col border border-slate-800/80 rounded-2xl overflow-hidden shadow-2xl bg-slate-950/60 p-5 select-none" style="height: 100%; min-height: 480px;">
  
  <!-- Section Title & Search Header -->
  <div class="flex flex-col sm:flex-row items-stretch sm:items-center justify-between gap-4 pb-4 border-b border-slate-900 mb-4 shrink-0">
    <div class="text-left">
      <span class="text-[8px] font-extrabold uppercase tracking-widest text-slate-500 font-sans block">Workflow Flow Path</span>
      <span class="text-[10px] font-extrabold font-display uppercase tracking-wider text-slate-300">Topologically Ordered Step List</span>
    </div>
    
    <!-- Filter Search Box -->
    <div class="flex items-center gap-2 px-3 py-1.5 bg-slate-950 border border-slate-900 rounded-xl shadow-inner w-full sm:w-[220px]">
      <svg width="12" height="12" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" class="text-slate-500 shrink-0">
        <path stroke-linecap="round" stroke-linejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <input 
        type="text" 
        bind:value={searchQuery}
        placeholder="Search steps..."
        class="bg-transparent border-0 p-0 text-[10px] text-white placeholder-slate-550 focus:ring-0 focus:outline-none w-full font-sans"
      />
      {#if searchQuery}
        <button 
          type="button"
          onclick={() => searchQuery = ''}
          class="text-[8px] font-bold text-slate-500 hover:text-white transition-all uppercase tracking-wider font-mono cursor-pointer shrink-0"
        >
          Clear
        </button>
      {/if}
    </div>
  </div>

  {#if !nodes || nodes.length === 0}
    <!-- Empty State -->
    <div class="flex-1 flex flex-col items-center justify-center text-slate-500 z-10 select-none pointer-events-none">
      <div class="w-14 h-14 rounded-2xl bg-slate-950/80 border border-slate-900 flex items-center justify-center text-slate-700 mb-4 shadow-inner">
        <svg width="24" height="24" class="stroke-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 12c0-1.232-.046-2.453-.138-3.662a4.006 4.006 0 00-3.7-3.7 48.656 48.656 0 00-7.324 0 4.006 4.006 0 00-3.7 3.7C4.547 9.547 4.5 10.768 4.5 12s.047 2.453.138 3.662a4.006 4.006 0 003.7 3.7 48.656 48.656 0 007.324 0 4.006 4.006 0 003.7-3.7c.092-1.209.138-2.43.138-3.662z" />
        </svg>
      </div>
      <span class="text-xs font-extrabold uppercase tracking-widest text-slate-400">Empty Graph Steps</span>
      <span class="text-[10px] text-slate-500 mt-1">Activate a workflow definition schema to list execution steps.</span>
    </div>
  {:else if orderedNodes.length === 0}
    <!-- Empty Search Results State -->
    <div class="flex-1 flex flex-col items-center justify-center text-slate-500 z-10 select-none">
      <span class="text-xs font-extrabold uppercase tracking-widest text-slate-500 font-sans">No Steps Match Search</span>
      <button 
        type="button" 
        onclick={() => searchQuery = ''}
        class="mt-2 text-[10px] font-bold text-indigo-400 hover:text-indigo-300 transition-colors uppercase tracking-wider font-mono cursor-pointer"
      >
        Reset Filter
      </button>
    </div>
  {:else}
    <!-- Topologically Ordered Step Timeline -->
    <div class="flex-1 overflow-y-auto custom-scrollbar pr-2 min-h-0 relative">
      
      <!-- Vertical Timeline Track Link -->
      <div class="absolute left-6 top-6 bottom-6 w-0.5 bg-slate-900 z-0"></div>

      <div class="flex flex-col gap-4 relative z-10">
        {#each orderedNodes as node, index}
          {@const colorMeta = getNodeColorMeta(node.id, node.type)}
          {@const isSelected = selectedNodeId === node.id}
          {@const isActive = activeNodeId === node.id}
          {@const outEdges = edges.filter((e: Edge) => e.from === node.id)}

          <div class="flex items-start gap-4">
            
            <!-- Timeline Index Step Circle Indicator -->
            <button 
              type="button"
              onclick={() => handleSelectNode(node)}
              class="w-4.5 h-4.5 rounded-full border flex items-center justify-center shrink-0 mt-3 transition-all duration-300 cursor-pointer shadow-md select-none tap-target hover:scale-110 active:scale-95
                {isSelected ? 'ring-4 ring-indigo-500/20' : isActive ? 'ring-4 ring-emerald-500/25 animate-pulse' : ''}"
              style="border-color: {isSelected ? colorMeta.accent : colorMeta.border}; background-color: {isSelected ? colorMeta.accent : colorMeta.bg}; color: {isSelected ? '#ffffff' : colorMeta.text};"
            >
              {#if node.type === 'entry'}
                <svg width="8" height="8" fill="currentColor" viewBox="0 0 24 24">
                  <path d="M8 5v14l11-7z" />
                </svg>
              {:else if node.type === 'exit'}
                <svg width="8" height="8" fill="none" stroke="currentColor" stroke-width="4.5" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                </svg>
              {:else}
                <span class="text-[8px] font-black font-mono leading-none">{index + 1}</span>
              {/if}
            </button>

            <!-- Premium Interactive Node Card Details -->
            <div 
              role="button"
              tabindex="0"
              onclick={() => handleSelectNode(node)}
              onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); handleSelectNode(node); } }}
              class="flex-1 flex flex-col p-4 rounded-xl border transition-all duration-300 text-left cursor-pointer shadow-sm select-none hover:-translate-y-0.5 hover:shadow-md hover:brightness-105 active:scale-[0.99] active:translate-y-0 active:brightness-95
                {isSelected ? 'ring-2 ring-indigo-500/10' : isActive ? 'ring-2 ring-emerald-500/15' : ''}"
              style="border-color: {isSelected ? colorMeta.accent : colorMeta.border}; background-color: {colorMeta.bg}; border-left: 4px solid {colorMeta.accent}; box-shadow: {isSelected ? `0 0 20px -2px hsla(${isSelected ? '240' : '0'}, 60%, 55%, 0.08)` : 'none'};"
            >
              
              <!-- Card Header -->
              <div class="flex items-center justify-between gap-4">
                <div class="min-w-0">
                  <span class="text-xs font-bold leading-tight font-display tracking-wide block truncate" style="color: {colorMeta.text};">
                    {node.display_name || node.id}
                  </span>
                  
                  <!-- Type Badge Tag & ID display -->
                  <div class="flex items-center gap-2 mt-1 select-none">
                    <span class="text-[7.5px] font-extrabold uppercase font-mono tracking-widest px-1.5 py-0.5 rounded" style="background-color: {colorMeta.border}; color: {colorMeta.accent};">
                      {node.type === 'agent_node' ? 'agent' : node.type === 'cli_node' ? 'cli' : node.type}
                    </span>
                    <span class="text-[8px] font-mono text-slate-500 truncate" title="Unique ID: {node.id}">id: {node.id}</span>
                  </div>
                </div>

                <!-- Selected/Active Pill Badges -->
                <div class="flex items-center gap-1.5 shrink-0 select-none">
                  {#if isActive}
                    <span class="flex items-center gap-1.5 text-[8px] font-extrabold uppercase text-emerald-600 bg-emerald-500/10 border border-emerald-500/20 px-2 py-0.5 rounded-full font-sans animate-pulse">
                      <span class="w-1 h-1 rounded-full bg-emerald-500"></span>
                      Executing
                    </span>
                  {/if}
                  {#if isSelected}
                    <span class="text-[8px] font-extrabold uppercase text-indigo-600 bg-indigo-500/10 border border-indigo-500/20 px-2 py-0.5 rounded-full font-sans">
                      Selected
                    </span>
                  {/if}
                </div>
              </div>

              <!-- Node Description Preview (if available) -->
              {#if node.description}
                <p class="text-[9.5px] text-slate-500 leading-normal font-sans mt-2.5 line-clamp-2 border-l border-slate-200/80 pl-2">
                  {node.description}
                </p>
              {/if}

              <!-- Connected Outflow Branches Timelines -->
              {#if outEdges.length > 0}
                <div class="mt-3.5 pt-3 border-t border-slate-200/60 flex flex-col gap-2 shrink-0">
                  <span class="text-[8px] font-extrabold uppercase tracking-widest text-slate-400 select-none font-sans block">Branches & Routing:</span>
                  <div class="flex flex-wrap gap-2">
                    {#each outEdges as edge}
                      <button
                        type="button"
                        onclick={(e) => {
                          e.stopPropagation(); // Avoid triggering card selection click
                          const targetNode = nodes.find((n: Node) => n.id === edge.to);
                          if (targetNode) handleSelectNode(targetNode);
                        }}
                        class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[9px] font-mono font-bold bg-white border border-slate-200 text-slate-600 hover:bg-indigo-50 hover:border-indigo-200 hover:text-indigo-600 hover:-translate-y-0.5 active:scale-95 active:translate-y-0 transition-all duration-150 cursor-pointer shadow-sm select-none"
                        title="Jump to target node '{edge.to}'"
                      >
                        {#if edge.condition}
                          <span class="text-[7.5px] font-extrabold uppercase tracking-wider text-amber-600 bg-amber-50 border border-amber-200/60 px-1 rounded">if: {edge.condition}</span>
                        {/if}
                        <span>→ {nodes.find((n: Node) => n.id === edge.to)?.display_name || edge.to}</span>
                      </button>
                    {/each}
                  </div>
                </div>
              {/if}

            </div>

          </div>
        {/each}
      </div>

    </div>
  {/if}

</div>

<style>
  /* Custom scrollbar matching styling tokens */
  .custom-scrollbar::-webkit-scrollbar {
    width: 6px;
  }
  .custom-scrollbar::-webkit-scrollbar-track {
    background: transparent;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb {
    background: rgba(15, 23, 42, 0.4);
    border-radius: 99px;
  }
  .custom-scrollbar::-webkit-scrollbar-thumb:hover {
    background: rgba(15, 23, 42, 0.6);
  }
</style>
