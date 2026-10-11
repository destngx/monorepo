<script lang="ts">
  import { onMount } from 'svelte';

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

  // Props using Svelte 5 runes
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

  // Camera state using Svelte 5 runes
  let scale = $state(1);
  let panX = $state(0);
  let panY = $state(0);

  let viewportEl = $state<HTMLElement | null>(null);
  let isDragging = $state(false);
  let dragStart = { x: 0, y: 0 };
  let panStart = { x: 0, y: 0 };

  function zoomIn() {
    scale = Math.min(2.5, scale + 0.1);
  }

  function zoomOut() {
    scale = Math.max(0.4, scale - 0.1);
  }

  function resetZoom() {
    scale = 1;
    panX = 0;
    panY = 0;
  }

  function fitToViewport() {
    if (!viewportEl || !nodes || nodes.length === 0) return;
    
    const vWidth = viewportEl.clientWidth;
    const vHeight = viewportEl.clientHeight;
    if (vWidth === 0 || vHeight === 0) return;
    
    const lWidth = computedLayout.width;
    const lHeight = computedLayout.height;
    
    // Fit with 32px safety margins on all sides
    const padding = 32;
    const scaleX = (vWidth - padding * 2) / lWidth;
    const scaleY = (vHeight - padding * 2) / lHeight;
    
    // Target optimal fit scale factor, capping at 1.0 to prevent pixelation on tiny graphs
    const fitScale = Math.min(1, Math.max(0.35, Math.min(scaleX, scaleY)));
    scale = fitScale;
    
    // Center layout nicely in the viewport
    panX = (vWidth - lWidth * fitScale) / 2;
    panY = (vHeight - lHeight * fitScale) / 2;
  }

  // Auto-fit reactively when the nodes array changes (meaning a new workflow is loaded)
  $effect(() => {
    if (nodes && nodes.length > 0 && viewportEl) {
      setTimeout(fitToViewport, 50);
    }
  });

  function handleMouseDown(e: MouseEvent) {
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('a') || target.closest('input') || target.closest('select')) {
      return;
    }
    isDragging = true;
    dragStart = { x: e.clientX, y: e.clientY };
    panStart = { x: panX, y: panY };
  }

  function handleTouchStart(e: TouchEvent) {
    const target = e.target as HTMLElement;
    if (target.closest('button') || target.closest('a') || target.closest('input') || target.closest('select')) {
      return;
    }
    if (e.touches.length === 1) {
      isDragging = true;
      dragStart = { x: e.touches[0].clientX, y: e.touches[0].clientY };
      panStart = { x: panX, y: panY };
    }
  }

  onMount(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging) return;
      panX = panStart.x + (e.clientX - dragStart.x);
      panY = panStart.y + (e.clientY - dragStart.y);
    };

    const handleMouseUp = () => {
      isDragging = false;
    };

    const handleTouchMove = (e: TouchEvent) => {
      if (!isDragging || e.touches.length !== 1) return;
      panX = panStart.x + (e.touches[0].clientX - dragStart.x);
      panY = panStart.y + (e.touches[0].clientY - dragStart.y);
    };

    const handleTouchEnd = () => {
      isDragging = false;
    };

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      const zoomFactor = 0.05;
      const direction = e.deltaY < 0 ? 1 : -1;
      scale = Math.min(2.5, Math.max(0.4, scale + direction * zoomFactor));
    };

    const handleResize = () => {
      fitToViewport();
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    window.addEventListener('touchmove', handleTouchMove, { passive: false });
    window.addEventListener('touchend', handleTouchEnd);
    window.addEventListener('resize', handleResize);

    if (viewportEl) {
      viewportEl.addEventListener('wheel', handleWheel, { passive: false });
      // Schedule initial layout fit
      setTimeout(fitToViewport, 50);
    }

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('touchmove', handleTouchMove);
      window.removeEventListener('touchend', handleTouchEnd);
      window.removeEventListener('resize', handleResize);
      if (viewportEl) {
        viewportEl.removeEventListener('wheel', handleWheel);
      }
    };
  });

  // Layout constants for graph canvas spacing
  const COLUMN_WIDTH = 295;
  const ROW_HEIGHT = 135;
  const PADDING_X = 65;
  const PADDING_Y = 65;
  const NODE_WIDTH = 220;
  const NODE_HEIGHT = 68;

  // Reactively calculate ranks and column placements for the DAG structure
  let computedLayout = $derived.by(() => {
    if (!nodes || nodes.length === 0) return { positions: new Map<string, {x: number, y: number}>(), width: 600, height: 400 };

    const ranks = new Map<string, number>();
    const incoming = new Map<string, string[]>();
    const outgoing = new Map<string, string[]>();

    // Initialise adjacency maps
    nodes.forEach((n: Node) => {
      incoming.set(n.id, []);
      outgoing.set(n.id, []);
    });

    edges.forEach((e: Edge) => {
      if (incoming.has(e.to)) incoming.get(e.to)!.push(e.from);
      if (outgoing.has(e.from)) outgoing.get(e.from)!.push(e.to);
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

    const columns: string[][] = [];
    ranks.forEach((rank, id) => {
      if (!columns[rank]) columns[rank] = [];
      columns[rank].push(id);
    });

    const cleanColumns = columns.filter(col => col && col.length > 0);
    const positions = new Map<string, { x: number; y: number; rank: number }>();
    let maxColSize = 0;

    cleanColumns.forEach((colNodes, colIndex) => {
      if (colNodes.length > maxColSize) maxColSize = colNodes.length;
      
      colNodes.forEach((nodeId, rowIndex) => {
        const colCenterOffset = (maxColSize - colNodes.length) * (ROW_HEIGHT / 2);
        const x = PADDING_X + colIndex * COLUMN_WIDTH;
        const y = PADDING_Y + rowIndex * ROW_HEIGHT + colCenterOffset;
        
        positions.set(nodeId, { x, y, rank: colIndex });
      });
    });

    const totalWidth = PADDING_X * 2 + cleanColumns.length * COLUMN_WIDTH;
    const totalHeight = PADDING_Y * 2 + maxColSize * ROW_HEIGHT;

    return { positions, width: Math.max(totalWidth, 800), height: Math.max(totalHeight, 420) };
  });

  // Calculate high-fidelity Bezier curves connecting the center points of adjacent node ports
  function getBezierPath(fromId: string, toId: string): string {
    const fromPos = computedLayout.positions.get(fromId);
    const toPos = computedLayout.positions.get(toId);
    
    if (!fromPos || !toPos) return '';

    const x1 = fromPos.x + NODE_WIDTH;
    const y1 = fromPos.y + NODE_HEIGHT / 2;
    const x2 = toPos.x;
    const y2 = toPos.y + NODE_HEIGHT / 2;

    const controlX = x1 + (x2 - x1) / 2;

    return `M ${x1} ${y1} C ${controlX} ${y1}, ${controlX} ${y2}, ${x2} ${y2}`;
  }

  // Curated premium cyberpunk color configurations for stable node hashing (adapted for light theme blackboard)
  function getNodeRandomColor(nodeId: string, nodeType: string) {
    if (nodeType === 'entry') {
      return { 
        border: 'rgba(16, 185, 129, 0.4)', 
        bg: 'rgba(240, 253, 244, 0.95)', // Gorgeous soft light green tint
        text: 'rgb(6, 95, 70)', // Highly readable dark green text
        glowColor: 'rgba(16, 185, 129, 0.15)', 
        leftBorderColor: 'rgb(16, 185, 129)' 
      };
    }
    if (nodeType === 'exit') {
      return { 
        border: 'rgba(244, 63, 94, 0.4)', 
        bg: 'rgba(255, 241, 242, 0.95)', // Gorgeous soft light red tint
        text: 'rgb(159, 18, 57)', // Highly readable dark red text
        glowColor: 'rgba(244, 63, 94, 0.15)', 
        leftBorderColor: 'rgb(244, 63, 94)' 
      };
    }
    
    // Stable hash seed from node ID string
    let hash = 0;
    for (let i = 0; i < nodeId.length; i++) {
      hash = nodeId.charCodeAt(i) + ((hash << 5) - hash);
    }
    const hue = Math.abs(hash) % 360;
    
    return {
      border: `hsla(${hue}, 60%, 55%, 0.35)`,
      bg: `hsla(${hue}, 85%, 98%, 0.96)`, // Soft high-contrast pastel background
      text: `hsl(${hue}, 75%, 28%)`, // Dark, highly readable colored text
      glowColor: `hsla(${hue}, 60%, 55%, 0.15)`,
      leftBorderColor: `hsl(${hue}, 70%, 48%)`
    };
  }

  function handleSelectNode(node: Node) {
    selectedNodeId = node.id;
    onSelectNode(node);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<!-- Visual grid editor wrapper featuring the custom infinite cyber grid, zoom and pan -->
<div 
  bind:this={viewportEl}
  class="relative w-full h-full border border-slate-800/80 rounded-2xl overflow-hidden shadow-2xl select-none
    {isDragging ? 'cursor-grabbing' : 'cursor-grab'}"
  style="height: 100%; min-height: 480px; background: radial-gradient(circle at center, #ffffff 0%, #f4f6fa 100%);"
  onmousedown={handleMouseDown}
  ontouchstart={handleTouchStart}
>
  {#if !nodes || nodes.length === 0}
    <!-- Empty State -->
    <div class="absolute inset-0 flex flex-col items-center justify-center text-slate-500 z-10 select-none pointer-events-none">
      <div class="w-14 h-14 rounded-2xl bg-slate-950/80 border border-slate-900 flex items-center justify-center text-slate-700 mb-4 shadow-inner">
        <svg width="24" height="24" class="stroke-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" d="M19.5 12c0-1.232-.046-2.453-.138-3.662a4.006 4.006 0 00-3.7-3.7 48.656 48.656 0 00-7.324 0 4.006 4.006 0 00-3.7 3.7C4.547 9.547 4.5 10.768 4.5 12s.047 2.453.138 3.662a4.006 4.006 0 003.7 3.7 48.656 48.656 0 007.324 0 4.006 4.006 0 003.7-3.7c.092-1.209.138-2.43.138-3.662z" />
        </svg>
      </div>
      <span class="text-xs font-extrabold uppercase tracking-widest text-slate-400">Empty Graph Blueprint</span>
      <span class="text-[10px] text-slate-500 mt-1">Activate a workflow definition schema to render the DAG canvas.</span>
    </div>
  {:else}
    <!-- Viewport-centered Masked Grid Container -->
    <div 
      class="absolute inset-0 pointer-events-none overflow-hidden rounded-2xl z-0"
      style="mask-image: radial-gradient(circle at center, black 40%, transparent 92%); -webkit-mask-image: radial-gradient(circle at center, black 40%, transparent 92%);"
    >
      <div 
        class="absolute -inset-[5000px] cyber-grid-canvas {isDragging ? '' : 'transition-transform duration-150 ease-out'}"
        style="transform: translate({panX}px, {panY}px) scale({scale});"
      ></div>
    </div>

    <!-- Panned and Zoomed Content Layer -->
    <div 
      class="absolute top-0 left-0 origin-top-left w-full h-full {isDragging ? '' : 'transition-transform duration-150 ease-out'}"
      style="transform: translate({panX}px, {panY}px) scale({scale});"
    >

      <!-- Dynamic Rank Box Container -->
      <div class="relative z-10" style="width: {computedLayout.width}px; height: {computedLayout.height}px;">
        
        <!-- SVG Connectors Background Layer -->
        <svg class="absolute top-0 left-0 w-full h-full pointer-events-none z-0">
          <defs>
            <linearGradient id="edge-flow-gradient" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#4f46e5" stop-opacity="0.5" />
              <stop offset="100%" stop-color="#8b5cf6" stop-opacity="0.5" />
            </linearGradient>
            <linearGradient id="edge-active-flow" x1="0%" y1="0%" x2="100%" y2="0%">
              <stop offset="0%" stop-color="#10b981" stop-opacity="0.9" />
              <stop offset="50%" stop-color="#c084fc" stop-opacity="0.9" />
              <stop offset="100%" stop-color="#38bdf8" stop-opacity="0.9" />
            </linearGradient>
            <filter id="laser-glow" x="-20%" y="-20%" width="140%" height="140%">
              <feGaussianBlur stdDeviation="3.5" result="blur" />
              <feComposite in="SourceGraphic" in2="blur" operator="over" />
            </filter>
          </defs>

          <!-- Draw all edges -->
          {#each edges as edge}
            {@const isActive = activeNodeId === edge.to || (activeNodeId && edge.from === activeNodeId)}
            <g>
              <!-- Edge Shadow path (for glowing hover) -->
              <path 
                d={getBezierPath(edge.from, edge.to)} 
                fill="none" 
                stroke={isActive ? 'url(#edge-active-flow)' : 'url(#edge-flow-gradient)'} 
                stroke-width={isActive ? '3.5' : '2'}
                class="transition-all duration-300"
                opacity={isActive ? '1' : '0.8'}
                filter={isActive ? 'url(#laser-glow)' : ''}
              />
              {#if isActive}
                <!-- Animated data packet particle flow along curve -->
                <path 
                  d={getBezierPath(edge.from, edge.to)} 
                  fill="none" 
                  stroke="#38bdf8" 
                  stroke-width="2.5"
                  stroke-dasharray="8 12"
                  class="animate-packet-flow"
                />
              {/if}
              
              <!-- Conditional routing text badges -->
              {#if edge.condition}
                {@const fromPos = computedLayout.positions.get(edge.from)}
                {@const toPos = computedLayout.positions.get(edge.to)}
                {#if fromPos && toPos}
                  {@const midX = fromPos.x + NODE_WIDTH + (toPos.x - (fromPos.x + NODE_WIDTH)) / 2}
                  {@const midY = (fromPos.y + toPos.y) / 2 + NODE_HEIGHT / 2 - 8}
                  <g class="cursor-help select-none">
                    <rect 
                      x={midX - 52} 
                      y={midY - 11} 
                      width="104" 
                      height="20" 
                      rx="8" 
                      class="fill-white/95 stroke-slate-200 shadow-sm" 
                      stroke-width="1"
                    />
                    <text 
                      x={midX} 
                      y={midY + 2} 
                      text-anchor="middle" 
                      class="fill-indigo-600 font-mono text-[8px] font-bold uppercase tracking-wider"
                    >
                      {edge.condition.length > 15 ? 'if: condition' : `if: ${edge.condition}`}
                    </text>
                    <title>{edge.condition}</title>
                  </g>
                {/if}
              {/if}
            </g>
          {/each}
        </svg>

        <!-- HTML Interactive Nodes Layer -->
        {#each nodes as node}
          {@const pos = computedLayout.positions.get(node.id)}
          {#if pos}
            {@const colorMeta = getNodeRandomColor(node.id, node.type)}
            {@const isSelected = selectedNodeId === node.id}
            {@const isActive = activeNodeId === node.id}
            
            {#if node.type === 'entry'}
              <!-- Special Entry Node Design -->
              <button 
                type="button"
                onclick={() => handleSelectNode(node)}
                class="absolute z-10 flex items-center gap-3.5 p-3 rounded-2xl border transition-all duration-300 backdrop-blur-xl select-none text-left shadow-lg shadow-black/10 glass-slab cursor-pointer
                  {isSelected ? 'scale-[1.04] ring-4 ring-emerald-500/15' : isActive ? 'scale-[1.04] ring-4 ring-emerald-500/25' : 'hover:scale-[1.02]'}
                  hover:brightness-105 hover:-translate-y-0.5 hover:shadow-emerald-500/5 active:scale-[0.96] active:translate-y-0 active:brightness-95 active:shadow-inner"
                style="left: {pos.x}px; top: {pos.y}px; width: {NODE_WIDTH}px; height: {NODE_HEIGHT}px; border-color: {isSelected ? '#10b981' : colorMeta.border}; background-color: {colorMeta.bg}; border-left: 3.5px solid #10b981; box-shadow: {isSelected ? '0 0 20px -2px rgba(16, 185, 129, 0.25)' : '0 4px 15px -5px rgba(0,0,0,0.08)'};"
              >
                <!-- Special Start Icon -->
                <div class="w-8 h-8 rounded-xl bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center shrink-0 shadow-inner">
                  <svg width="15" height="15" fill="currentColor" viewBox="0 0 24 24">
                    <path d="M8 5v14l11-7z" class="fill-emerald-600" />
                  </svg>
                </div>
                <div class="min-w-0 flex flex-col justify-center">
                  <span class="text-[8px] font-extrabold uppercase tracking-widest leading-none mb-1 font-sans" style="color: {colorMeta.leftBorderColor};">Start</span>
                  <span class="text-xs font-extrabold truncate leading-none font-display" style="color: {colorMeta.text};">{node.display_name || node.id}</span>
                </div>
              </button>

            {:else if node.type === 'exit'}
              <!-- Special Exit Node Design -->
              <button 
                type="button"
                onclick={() => handleSelectNode(node)}
                class="absolute z-10 flex items-center gap-3.5 p-3 rounded-2xl border transition-all duration-300 backdrop-blur-xl select-none text-left shadow-lg shadow-black/10 glass-slab cursor-pointer
                  {isSelected ? 'scale-[1.04] ring-4 ring-rose-500/15' : isActive ? 'scale-[1.04] ring-4 ring-rose-500/25' : 'hover:scale-[1.02]'}
                  hover:brightness-105 hover:-translate-y-0.5 active:scale-[0.96] active:translate-y-0 active:brightness-95 active:shadow-inner"
                style="left: {pos.x}px; top: {pos.y}px; width: {NODE_WIDTH}px; height: {NODE_HEIGHT}px; border-color: {isSelected ? '#f43f5e' : colorMeta.border}; background-color: {colorMeta.bg}; border-left: 3.5px solid #f43f5e; box-shadow: {isSelected ? '0 0 20px -2px rgba(244, 63, 94, 0.25)' : '0 4px 15px -5px rgba(0,0,0,0.08)'};"
              >
                <!-- Special End Icon -->
                <div class="w-8 h-8 rounded-xl bg-rose-500/10 border border-rose-500/20 flex items-center justify-center shrink-0 shadow-inner">
                  <svg width="14" height="14" fill="none" stroke="currentColor" stroke-width="3" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" class="stroke-rose-600" />
                  </svg>
                </div>
                <div class="min-w-0 flex flex-col justify-center">
                  <span class="text-[8px] font-extrabold uppercase tracking-widest leading-none mb-1 font-sans" style="color: {colorMeta.leftBorderColor};">Exit</span>
                  <span class="text-xs font-extrabold truncate leading-none font-display" style="color: {colorMeta.text};">{node.display_name || node.id}</span>
                </div>
              </button>

            {:else}
              <!-- Regular Colorful Node Design (showing only name or ID) -->
              <button 
                type="button"
                onclick={() => handleSelectNode(node)}
                class="absolute z-10 flex items-center p-3 rounded-xl border transition-all duration-300 backdrop-blur-xl select-none text-left shadow-lg shadow-black/10 glass-slab cursor-pointer
                  {isSelected 
                    ? 'scale-[1.04] z-20'
                    : isActive 
                      ? 'scale-[1.04] z-20 ring-4 ring-emerald-500/15'
                      : 'hover:scale-[1.02]'}
                  hover:brightness-105 hover:-translate-y-0.5 active:scale-[0.96] active:translate-y-0 active:brightness-95 active:shadow-inner"
                style="left: {pos.x}px; top: {pos.y}px; width: {NODE_WIDTH}px; height: {NODE_HEIGHT}px; border-color: {isSelected ? colorMeta.leftBorderColor : colorMeta.border}; background-color: {colorMeta.bg}; border-left: 3.5px solid {colorMeta.leftBorderColor}; box-shadow: {isSelected ? `0 0 22px -2px ${colorMeta.glowColor}` : '0 4px 15px -5px rgba(0,0,0,0.08)'};"
              >
                <div class="min-w-0 flex-1 flex flex-col justify-center text-left">
                  <span class="text-xs font-bold truncate transition-colors font-display tracking-wide" style="color: {colorMeta.text};">
                    {node.display_name || node.id}
                  </span>
                  <span class="text-[8px] font-mono opacity-80 uppercase tracking-wider block mt-1 select-none font-sans" style="color: {colorMeta.leftBorderColor};">
                    {node.type === 'agent_node' ? 'agent' : node.type === 'cli_node' ? 'cli' : node.type}
                  </span>
                </div>
              </button>
            {/if}
          {/if}
        {/each}

      </div>
    </div>

    <!-- Floating HUD Canvas Controls -->
    <div class="absolute bottom-4 right-4 flex items-center gap-2 z-30 select-none">
      <div class="glass-slab px-3 py-1.5 rounded-xl text-[9px] font-mono font-bold text-slate-400 border border-slate-800/80 shadow-md">
        {Math.round(scale * 100)}%
      </div>
      
      <div class="glass-slab p-1 rounded-xl flex items-center gap-1 border border-slate-800/80 shadow-md">
        <!-- Zoom Out -->
        <button 
          type="button"
          onclick={zoomOut}
          title="Zoom Out"
          class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-900/60 transition-all duration-150 cursor-pointer flex items-center justify-center tap-target hover:-translate-y-0.5 active:scale-90 active:bg-indigo-500/10 active:text-indigo-400"
        >
          <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M20 12H4" />
          </svg>
        </button>

        <!-- Reset Zoom to Fit -->
        <button 
          type="button"
          onclick={fitToViewport}
          title="Fit to Screen"
          class="px-2.5 py-1.5 rounded-lg text-[9px] font-mono font-bold text-slate-400 hover:text-white hover:bg-slate-900/60 transition-all duration-150 cursor-pointer flex items-center justify-center tap-target hover:-translate-y-0.5 active:scale-90 active:bg-indigo-500/10 active:text-indigo-400"
        >
          Fit
        </button>

        <!-- Zoom In -->
        <button 
          type="button"
          onclick={zoomIn}
          title="Zoom In"
          class="p-1.5 rounded-lg text-slate-400 hover:text-white hover:bg-slate-900/60 transition-all duration-150 cursor-pointer flex items-center justify-center tap-target hover:-translate-y-0.5 active:scale-90 active:bg-indigo-500/10 active:text-indigo-400"
        >
          <svg width="14" height="14" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .cyber-grid-canvas {
    background-image: 
      radial-gradient(rgba(99, 102, 241, 0.14) 1.2px, transparent 1.2px),
      linear-gradient(rgba(99, 102, 241, 0.06) 1px, transparent 1px),
      linear-gradient(90deg, rgba(99, 102, 241, 0.06) 1px, transparent 1px);
    background-size: 24px 24px, 48px 48px, 48px 48px;
    background-position: center;
  }
</style>
