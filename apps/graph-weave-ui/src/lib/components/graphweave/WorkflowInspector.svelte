<script lang="ts">
  import { onMount } from 'svelte';
  import { graphWeaveApi } from '$lib/graphweave/api';

  interface Node {
    id: string;
    type: string;
    display_name?: string;
    description?: string;
    node_id?: string;
    node_hash?: string;
    tenant_id?: string;
    config?: Record<string, any>;
    guardrails?: Record<string, any>;
    retry_config?: Record<string, any>;
  }

  // Props
  let { 
    node = null, 
    tenantId = 'default',
    onSave = (updatedNode: Node) => {},
    onDelete = (nodeId: string) => {},
    onAddNode = (newNode: Node) => {}
  } = $props<{
    node: Node | null;
    tenantId?: string;
    onSave: (updatedNode: Node) => void;
    onDelete: (nodeId: string) => void;
    onAddNode: (newNode: Node) => void;
  }>();

  // Selected tab in the inspector
  let activeTab = $state<'form' | 'json'>('form');

  // Mobile sheet expand status
  let isMobileExpanded = $state(true);

  // Form states mapped from the node
  let id = $state('');
  let type = $state('agent_node');
  let displayName = $state('');
  let description = $state('');
  let nodeHash = $state('');
  
  // Specific node configs
  let systemPrompt = $state('');
  let userPromptTemplate = $state('');
  let outputKey = $state('');
  let conditionExpression = $state('');
  let maxIterations = $state(10);
  let cliCommand = $state('');
  
  // Add new node form states
  let showAddForm = $state(false);
  let newId = $state('');
  let newType = $state('agent_node');

  // JSON copy state
  let isCopied = $state(false);

  // Reusable standalone catalog registry states
  let isLoadingNode = $state(false);
  let catalogFetchError = $state<string | null>(null);
  let catalogLinkage = $state<'syncing' | 'linked' | 'local' | 'error'>('local');
  let lastLoadedNodeId = '';

  async function loadCatalogData(hash: string) {
    if (!hash || hash === '') {
      catalogLinkage = 'local';
      return;
    }
    
    // Parse tenant_id and node_id from the hash
    let clean = hash;
    if (clean.startsWith('nodes:')) {
      clean = clean.substring(6);
    }
    const parts = clean.split(':');
    let tId = tenantId || 'default';
    let nId = '';
    
    if (parts.length >= 3) {
      tId = parts[0];
      nId = parts.slice(1).join(':');
    } else if (parts.length === 2) {
      nId = clean;
    } else {
      nId = `${clean}:v1.0.0`;
    }
    
    isLoadingNode = true;
    catalogFetchError = null;
    catalogLinkage = 'syncing';
    
    try {
      const catalogNode = await graphWeaveApi.getNode(nId, tId) as any;
      if (catalogNode) {
        // Map catalog details to state
        if (catalogNode.description) {
          description = catalogNode.description;
        }
        if (catalogNode.node_name || catalogNode.name) {
          displayName = catalogNode.node_name || catalogNode.name;
        }
        
        // Extract config either from top level or config object
        const cfg = catalogNode.config || {};
        const sysPr = catalogNode.system_prompt || cfg.system_prompt;
        const usrPr = catalogNode.user_prompt_template || cfg.user_prompt_template;
        
        if (type === 'agent_node') {
          if (sysPr !== undefined) systemPrompt = sysPr;
          if (usrPr !== undefined) userPromptTemplate = usrPr;
          if (cfg.output_key !== undefined) outputKey = cfg.output_key;
        } else if (type === 'orchestrator') {
          if (sysPr !== undefined) systemPrompt = sysPr;
          if (cfg.max_iterations !== undefined) maxIterations = Number(cfg.max_iterations);
          if (cfg.output_key !== undefined) outputKey = cfg.output_key;
        } else if (type === 'branch') {
          if (cfg.condition_expression !== undefined) conditionExpression = cfg.condition_expression;
        } else if (type === 'cli_node') {
          if (cfg.command !== undefined) cliCommand = cfg.command;
          if (cfg.output_key !== undefined) outputKey = cfg.output_key;
        }
        
        catalogLinkage = 'linked';
      } else {
        catalogLinkage = 'local';
      }
    } catch (err: any) {
      console.warn('Failed to fetch node from catalog registry:', err);
      catalogFetchError = err.message || String(err);
      catalogLinkage = 'error';
    } finally {
      isLoadingNode = false;
    }
  }

  // Watch for changes to the selected node and populate form states
  $effect(() => {
    if (node) {
      if (node.id !== lastLoadedNodeId) {
        lastLoadedNodeId = node.id;
        showAddForm = false;
        isMobileExpanded = true; // Auto expand on mobile when a node is selected
        id = node.id;
        type = node.type;
        displayName = node.display_name || '';
        description = node.description || '';
        let rawHash = '';
        if (node.node_hash) {
          rawHash = node.node_hash;
        } else if (node.node_id) {
          if (node.node_id.startsWith('nodes:')) {
            rawHash = node.node_id;
          } else {
            const tenant = node.tenant_id || tenantId || 'default';
            if (node.node_id.includes(':')) {
              const parts = node.node_id.split(':');
              if (parts.length === 3) {
                rawHash = `nodes:${node.node_id}`;
              } else if (parts.length === 2) {
                rawHash = `nodes:${tenant}:${node.node_id}`;
              } else {
                rawHash = `nodes:${tenant}:${node.node_id}`;
              }
            } else {
              rawHash = `nodes:${tenant}:${node.node_id}:v1.0.0`;
            }
          }
        } else {
          const tenant = node.tenant_id || tenantId || 'default';
          if (node.id === 'classify_and_extract' || node.id === 'classify_and_extract_metadata' || node.id === 'classify_metadata') {
            rawHash = 'nodes:pkm:extract_sources_and_atomic_ideas:v1.0.0';
          } else {
            rawHash = (node.type !== 'entry' && node.type !== 'exit' ? `nodes:${tenant}:${node.id}:v1.0.0` : '');
          }
        }
        nodeHash = rawHash;
        
        if (node.type === 'agent_node') {
          systemPrompt = node.config?.system_prompt || '';
          userPromptTemplate = node.config?.user_prompt_template || '';
          outputKey = node.config?.output_key || '';
        } else if (node.type === 'orchestrator') {
          systemPrompt = node.config?.system_prompt || '';
          maxIterations = node.config?.max_iterations || 10;
          outputKey = node.config?.output_key || '';
        } else if (node.type === 'branch') {
          conditionExpression = node.config?.condition_expression || '';
        } else if (node.type === 'cli_node') {
          cliCommand = node.config?.command || '';
          outputKey = node.config?.output_key || '';
        }

        // Trigger dynamic fetch from standalone registry catalog in Redis
        if (node.type !== 'entry' && node.type !== 'exit' && rawHash) {
          loadCatalogData(rawHash);
        } else {
          catalogLinkage = 'local';
        }
      }
    } else {
      lastLoadedNodeId = '';
      id = '';
      type = 'agent_node';
      displayName = '';
      description = '';
      nodeHash = '';
      systemPrompt = '';
      userPromptTemplate = '';
      outputKey = '';
      conditionExpression = '';
      cliCommand = '';
      catalogLinkage = 'local';
    }
  });

  async function handleSave() {
    if (!node) return;

    const updatedConfig: Record<string, any> = { ...(node.config || {}) };
    
    if (type === 'agent_node') {
      updatedConfig.system_prompt = systemPrompt;
      updatedConfig.user_prompt_template = userPromptTemplate;
      updatedConfig.output_key = outputKey;
    } else if (type === 'orchestrator') {
      updatedConfig.system_prompt = systemPrompt;
      updatedConfig.max_iterations = Number(maxIterations);
      updatedConfig.output_key = outputKey;
    } else if (type === 'branch') {
      updatedConfig.condition_expression = conditionExpression;
    } else if (type === 'cli_node') {
      updatedConfig.command = cliCommand;
      updatedConfig.output_key = outputKey;
    }

    let tenant = node.tenant_id || tenantId || 'default';
    let cleanNodeId = nodeHash;
    if (cleanNodeId.startsWith('nodes:')) {
      cleanNodeId = cleanNodeId.substring(6);
    }
    
    if (cleanNodeId.startsWith(tenant + ':')) {
      cleanNodeId = cleanNodeId.substring(tenant.length + 1);
    } else if (cleanNodeId.includes(':')) {
      const parts = cleanNodeId.split(':');
      if (parts.length === 3) {
        tenant = parts[0];
        cleanNodeId = parts.slice(1).join(':');
      }
    }
    
    const finalNodeHash = `nodes:${tenant}:${cleanNodeId}`;

    // Separation of Concerns: Persist prompts to the standalone Redis catalog node, not in the workflow JSON
    if (catalogLinkage === 'linked') {
      try {
        await graphWeaveApi.updateNode(cleanNodeId, tenant, {
          node_name: cleanNodeId.split(':')[0],
          description: description,
          tags: node.tags || [],
          capabilities: node.capabilities || [],
          config: {
            system_prompt: systemPrompt,
            user_prompt_template: userPromptTemplate,
            output_key: outputKey,
            max_iterations: type === 'orchestrator' ? Number(maxIterations) : undefined,
            command: type === 'cli_node' ? cliCommand : undefined,
            condition_expression: type === 'branch' ? conditionExpression : undefined,
          }
        });
        
        // Strip heavy prompts from the local workflow definition config to keep it structural
        delete updatedConfig.system_prompt;
        delete updatedConfig.user_prompt_template;
      } catch (err) {
        console.warn('Failed to update standalone catalog node in Redis registry:', err);
      }
    }

    const updatedNode: Node = {
      ...node,
      display_name: displayName,
      description: description,
      node_hash: finalNodeHash,
      node_id: cleanNodeId,
      tenant_id: tenant,
      config: updatedConfig
    };

    onSave(updatedNode);
  }

  function handleAddNewNode() {
    if (!newId || !newId.trim()) return;

    const newNode: Node = {
      id: newId.trim().toLowerCase().replace(/\s+/g, '-'),
      type: newType,
      display_name: newId,
      description: `Custom ${newType} node`,
      config: newType === 'agent_node' ? {
        system_prompt: 'You are an autonomous agent node. Load appropriate skills and execute task.',
        user_prompt_template: 'Analyze the following task: {topic}',
        output_key: `${newId.trim().toLowerCase().replace(/\s+/g, '_')}_output`
      } : newType === 'orchestrator' ? {
        system_prompt: 'You are a ReAct orchestration loop agent. Gather data and output results.',
        max_iterations: 10,
        allowed_skills: ['search', 'load_skill']
      } : {}
    };

    onAddNode(newNode);
    newId = '';
    showAddForm = false;
  }

  function handleCopyJson() {
    if (!node) return;
    navigator.clipboard.writeText(JSON.stringify(node, null, 2));
    isCopied = true;
    setTimeout(() => isCopied = false, 2000);
  }

  // Derive variable placeholders inside prompt template
  let detectedVariables = $derived.by(() => {
    if (!userPromptTemplate) return [];
    const regex = /\{([^}]+)\}/g;
    const matches = [];
    let match;
    while ((match = regex.exec(userPromptTemplate)) !== null) {
      matches.push(match[1]);
    }
    return Array.from(new Set(matches));
  });
</script>

<!-- Floating Glass Slab Deck Container -->
<div 
  class="w-full flex flex-col glass-slab overflow-hidden transition-all duration-300 z-20 h-full relative
    {node ? 'fixed bottom-0 inset-x-0 max-h-[85vh] lg:relative lg:inset-auto lg:h-full lg:max-h-none' : 'relative h-full'} 
    {!isMobileExpanded && node ? 'h-14 lg:h-full' : ''}"
>
  
  {#if node}
    <!-- Drag Handle / Title bar for Mobile bottom sheet -->
    <div 
      role="button"
      tabindex="0"
      onclick={() => isMobileExpanded = !isMobileExpanded}
      onkeydown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); isMobileExpanded = !isMobileExpanded; } }}
      class="lg:hidden w-full flex flex-col items-center py-2.5 bg-slate-950 border-b border-slate-900 cursor-pointer select-none"
    >
      <div class="w-12 h-1 bg-slate-800 rounded-full mb-1"></div>
      <div class="flex items-center gap-2 px-4 w-full justify-between">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full bg-indigo-500 animate-pulse"></span>
          <span class="text-xs font-bold text-slate-200">Inspector: {displayName || id}</span>
        </div>
        <span class="text-[10px] text-indigo-400 font-bold">{isMobileExpanded ? 'Minimize' : 'Expand'}</span>
      </div>
    </div>

    <!-- Active Node Header (Desktop/Mobile Expanded) -->
    <div class="flex items-center justify-between px-5 py-4 border-b border-slate-900/60 bg-slate-950/20 shrink-0 {!isMobileExpanded ? 'hidden lg:flex' : 'flex'} select-none">
      <div class="flex items-center gap-2.5 min-w-0">
        <h3 class="text-sm font-extrabold font-display text-slate-200 truncate max-w-[130px]" title={displayName || id}>
          {displayName || id}
        </h3>
        <span class="px-2 py-0.5 rounded-lg text-[8px] font-extrabold uppercase tracking-widest bg-indigo-500/10 text-indigo-400 border border-indigo-500/20 shrink-0">
          {type}
        </span>
      </div>

      <!-- Segmented Tab Selectors inside Inspector -->
      <div class="segmented-control">
        <button 
          onclick={() => activeTab = 'form'}
          class="px-3 py-1.5 text-[9px] font-extrabold uppercase tracking-widest rounded-lg transition-all cursor-pointer tap-target flex items-center justify-center
            {activeTab === 'form' ? 'bg-slate-800/80 text-white shadow-sm' : 'text-slate-500 hover:text-slate-350'}"
        >
          Form
        </button>
        <button 
          onclick={() => activeTab = 'json'}
          class="px-3 py-1.5 text-[9px] font-extrabold uppercase tracking-widest rounded-lg transition-all cursor-pointer tap-target flex items-center justify-center
            {activeTab === 'json' ? 'bg-slate-800/80 text-white shadow-sm' : 'text-slate-500 hover:text-slate-350'}"
        >
          Spec
        </button>
      </div>
    </div>

    <!-- Inspector Body -->
    <div class="flex-1 overflow-y-auto p-5 space-y-4.5 custom-scrollbar {!isMobileExpanded ? 'hidden lg:block' : 'block'} text-left">
      {#if activeTab === 'form'}
        <!-- Input fields -->
        <div class="space-y-1.5">
          <label for="disp-name" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Display Name</label>
          <input 
            type="text" 
            id="disp-name"
            bind:value={displayName}
            class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
          />
        </div>

        <div class="space-y-1.5">
          <label for="desc" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Purpose / Description</label>
          <textarea 
            id="desc"
            bind:value={description}
            rows="2"
            class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors resize-none leading-relaxed glass-input"
          ></textarea>
        </div>

        <!-- Catalog Sync Linkage -->
        <div class="space-y-1.5">
          <div class="flex justify-between items-center select-none">
            <label for="node-hash" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500">Catalog Hash Signature</label>
            {#if nodeHash}
              <button 
                type="button"
                onclick={() => {
                  navigator.clipboard.writeText(nodeHash);
                }}
                class="text-[9px] font-bold text-indigo-400 hover:text-indigo-300 transition-colors flex items-center gap-1 cursor-pointer"
              >
                <svg width="11" height="11" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2.5">
                  <path stroke-linecap="round" stroke-linejoin="round" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2" />
                </svg>
                Copy
              </button>
            {/if}
          </div>
          <input 
            type="text" 
            id="node-hash"
            bind:value={nodeHash}
            placeholder="e.g. nodes:system:data-fetcher:v1.0.0"
            class="w-full rounded-xl px-4 py-2.5 text-xs text-indigo-300 font-mono focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
          />
          
          <!-- Redis Registry catalog link badge -->
          {#if type !== 'entry' && type !== 'exit'}
            <div class="flex flex-col gap-1.5 mt-2">
              <div class="flex items-center justify-between px-3 py-2 rounded-xl bg-slate-950/60 border border-slate-900 min-h-[38px] select-none">
                <div class="flex items-center gap-2">
                  {#if catalogLinkage === 'syncing'}
                    <span class="flex h-2 w-2 relative shrink-0">
                      <span class="animate-ping absolute inline-flex h-full w-full rounded-full bg-indigo-400 opacity-75"></span>
                      <span class="relative inline-flex rounded-full h-2 w-2 bg-indigo-500"></span>
                    </span>
                    <span class="text-[9px] font-bold text-indigo-400">Syncing with Registry...</span>
                  {:else if catalogLinkage === 'linked'}
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 shrink-0 shadow shadow-emerald-500/50 animate-heart-pulse"></span>
                    <span class="text-[9px] font-extrabold uppercase tracking-wider text-emerald-400">Linked to Catalog</span>
                  {:else if catalogLinkage === 'error'}
                    <span class="w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0"></span>
                    <span class="text-[9px] font-extrabold uppercase tracking-wider text-amber-400" title={catalogFetchError}>Local Override Active</span>
                  {:else}
                    <span class="w-1.5 h-1.5 rounded-full bg-slate-700 shrink-0"></span>
                    <span class="text-[9px] font-extrabold uppercase tracking-wider text-slate-500">Local Blueprint</span>
                  {/if}
                </div>
                
                {#if nodeHash}
                  <button 
                    type="button" 
                    onclick={() => loadCatalogData(nodeHash)}
                    class="text-[9px] font-extrabold uppercase tracking-widest text-indigo-400 hover:text-indigo-300 cursor-pointer px-2.5 py-1.5 border border-indigo-500/10 hover:border-indigo-500/30 rounded-lg bg-indigo-500/5 transition-all flex items-center justify-center min-h-[32px]"
                    disabled={isLoadingNode}
                  >
                    Fetch Catalog
                  </button>
                {/if}
              </div>
            </div>
          {/if}
        </div>

        <hr class="border-slate-900/60" />

        <!-- Node type specific properties -->
        {#if type === 'agent_node' || type === 'orchestrator'}
          <!-- System Prompt editor -->
          <div class="space-y-1.5">
            <div class="flex justify-between items-center select-none">
              <label for="sys-prompt" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500">System Instruction Prompt</label>
              <span class="text-[8px] bg-violet-500/10 text-violet-400 border border-violet-500/25 px-2 py-0.5 rounded-full font-bold uppercase tracking-wider">Agent Role</span>
            </div>
            <textarea 
              id="sys-prompt"
              bind:value={systemPrompt}
              rows="6"
              class="w-full rounded-xl px-4 py-3 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors leading-relaxed custom-scrollbar glass-input"
              placeholder="Define rules, skills playbook references, boundaries..."
            ></textarea>
          </div>

          <!-- User Prompt Template (For agent_node) -->
          {#if type === 'agent_node'}
            <div class="space-y-1.5">
              <label for="user-prompt" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">User Prompt Template</label>
              <textarea 
                id="user-prompt"
                bind:value={userPromptTemplate}
                rows="4"
                class="w-full rounded-xl px-4 py-3 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors leading-relaxed custom-scrollbar glass-input"
                placeholder={"Provide template context: Analyze {topic}..."}
              ></textarea>

              <!-- Variable Pills Helper -->
              {#if detectedVariables.length > 0}
                <div class="pt-1 flex flex-wrap items-center gap-2 select-none">
                  <span class="text-[8px] font-extrabold text-slate-500 uppercase tracking-widest">Braces variables:</span>
                  {#each detectedVariables as v}
                    <span class="text-[9px] font-bold font-mono bg-indigo-500/10 text-indigo-400 border border-indigo-500/25 px-2.5 py-0.5 rounded-full shadow-sm">
                      {v}
                    </span>
                  {/each}
                </div>
              {/if}
            </div>
          {/if}

          <!-- Max Iterations (For orchestrator react loop) -->
          {#if type === 'orchestrator'}
            <div class="space-y-1.5">
              <label for="max-iter" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Max ReAct Loop Iterations</label>
              <input 
                type="number" 
                id="max-iter"
                bind:value={maxIterations}
                min="1"
                max="30"
                class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
              />
            </div>
          {/if}

          <!-- Output State Key -->
          <div class="space-y-1.5">
            <label for="out-key" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Output State Storage Key</label>
            <input 
              type="text" 
              id="out-key"
              bind:value={outputKey}
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
              placeholder="e.g. analysis_output"
            />
          </div>
        {:else if type === 'branch'}
          <!-- Condition Expression -->
          <div class="space-y-1.5">
            <label for="cond-expr" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Branch Routing Expression (JSONPath)</label>
            <input 
              type="text" 
              id="cond-expr"
              bind:value={conditionExpression}
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
              placeholder="e.g. $.research_output.confidence"
            />
          </div>
        {:else if type === 'cli_node'}
          <!-- Bash Command -->
          <div class="space-y-1.5">
            <label for="cli-cmd" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">CLI Bash Command</label>
            <input 
              type="text" 
              id="cli-cmd"
              bind:value={cliCommand}
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
              placeholder={"e.g. swift run mac-ocr --file {image_path}"}
            />
          </div>

          <!-- Output State Key -->
          <div class="space-y-1.5">
            <label for="out-key-cli" class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500 block">Output State Storage Key</label>
            <input 
              type="text" 
              id="out-key-cli"
              bind:value={outputKey}
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 font-mono focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
              placeholder="e.g. cli_output"
            />
          </div>
        {:else}
          <div class="p-4 bg-slate-900/40 border border-slate-900 rounded-xl text-xs text-slate-500 italic leading-relaxed select-none">
            Structural checkpoint. No prompt parameters required.
          </div>
        {/if}

      {:else if activeTab === 'json'}
        <!-- Futuristic Spec Viewer -->
        <div class="h-full flex flex-col space-y-3">
          <div class="flex items-center justify-between select-none">
            <span class="text-[9px] font-extrabold uppercase tracking-widest text-slate-500">JSON Blueprint Spec</span>
            <button 
              onclick={handleCopyJson}
              class="px-3 py-1.5 text-[9px] font-extrabold text-indigo-400 hover:text-indigo-300 border border-indigo-500/20 bg-indigo-500/5 hover:bg-indigo-500/10 rounded-lg cursor-pointer transition-colors"
            >
              {isCopied ? 'Copied!' : 'Copy Schema'}
            </button>
          </div>
          <pre class="flex-1 w-full bg-[#020307]/90 border border-slate-900 rounded-xl p-4 font-mono text-[10px] text-indigo-300 overflow-auto whitespace-pre-wrap select-text leading-relaxed max-h-[380px] custom-scrollbar shadow-inner">
{JSON.stringify(node, null, 2)}
          </pre>
        </div>
      {/if}
    </div>

    <!-- Inspector Actions -->
    <div class="p-5 border-t border-slate-900 bg-slate-950/40 flex items-center justify-between gap-3 shrink-0 {!isMobileExpanded ? 'hidden lg:flex' : 'flex'} select-none">
      <button 
        type="button"
        onclick={() => onDelete(id)}
        class="flex-1 sm:flex-initial px-4 py-2.5 text-xs font-bold text-rose-455 border border-rose-500/20 bg-rose-500/5 hover:bg-rose-500/10 hover:border-rose-500/35 rounded-xl transition-colors cursor-pointer min-h-[44px]"
      >
        Delete Node
      </button>

      <button 
        type="button"
        onclick={handleSave}
        class="flex-1 sm:flex-initial px-5 py-2.5 text-xs font-bold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl shadow shadow-indigo-600/15 transition-all cursor-pointer min-h-[44px]"
      >
        Save Changes
      </button>
    </div>

  {:else}
    <!-- No selected node placeholder: add node forms -->
    <div class="flex-1 flex flex-col items-center justify-center p-6 text-center text-slate-500">
      {#if !showAddForm}
        <div class="w-14 h-14 rounded-2xl bg-slate-900/60 border border-slate-900/80 flex items-center justify-center text-slate-600 mb-4 shadow-inner">
          <svg width="24" height="24" class="stroke-1.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" d="M9.813 15.904L9 21m0 0l-.813-5.096L9 21zm0 0h1m2.5-11.5a1.5 1.5 0 11-3 0 1.5 1.5 0 013 0zM21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <span class="text-xs font-extrabold uppercase tracking-widest text-slate-400 block select-none">Node Inspector</span>
        <p class="text-xs text-slate-600 mt-2 max-w-[210px] leading-relaxed">
          Select any active node card on the visualizer canvas to configure its instruction sets and prompt templates.
        </p>
        
        <button 
          onclick={() => showAddForm = true}
          class="mt-6 px-4 py-2.5 text-xs font-bold text-slate-200 border border-slate-800 bg-slate-900/40 hover:bg-slate-900 rounded-xl hover:border-slate-700 transition-all cursor-pointer flex items-center gap-2 min-h-[44px] shadow-sm select-none"
        >
          <svg width="16" height="16" class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
            <path stroke-linecap="round" stroke-linejoin="round" d="M12 4v16m8-8H4" />
          </svg>
          Add Graph Node
        </button>
      {:else}
        <!-- Form to add new node -->
        <div class="w-full text-left space-y-4.5 max-h-[80vh] overflow-y-auto custom-scrollbar">
          <div class="flex justify-between items-center pb-2 border-b border-slate-900/60 select-none">
            <h4 class="text-xs font-extrabold uppercase tracking-wider text-slate-400">Add Canvas Node</h4>
            <button 
              onclick={() => showAddForm = false} 
              class="p-2 text-slate-500 hover:text-slate-350 cursor-pointer tap-target flex items-center justify-center rounded-lg" 
              aria-label="Cancel" 
              title="Cancel"
            >
              <svg width="20" height="20" class="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2">
                <path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          <div class="space-y-1.5">
            <label for="new-node-id" class="text-[9px] font-extrabold uppercase tracking-wider text-slate-500 block select-none">Node Identifier (unique ID)</label>
            <input 
              type="text" 
              id="new-node-id"
              bind:value={newId}
              placeholder="e.g. prompt_analyst"
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors min-h-[44px] glass-input"
            />
          </div>

          <div class="space-y-1.5">
            <label for="new-node-type" class="text-[9px] font-extrabold uppercase tracking-wider text-slate-500 block select-none">Execution Type</label>
            <select 
              id="new-node-type"
              bind:value={newType}
              class="w-full rounded-xl px-4 py-2.5 text-xs text-slate-200 focus:outline-none focus:border-indigo-500 transition-colors cursor-pointer min-h-[44px] glass-input"
            >
              <option value="agent_node">agent_node (prompt autonomy)</option>
              <option value="orchestrator">orchestrator (ReAct loop)</option>
              <option value="branch">branch (routing gate)</option>
              <option value="cli_node">cli_node (bash execution)</option>
              <option value="exit">exit (final node)</option>
            </select>
          </div>

          <button 
            onclick={handleAddNewNode}
            disabled={!newId.trim()}
            class="w-full py-2.5 text-xs font-bold text-white bg-indigo-600 hover:bg-indigo-500 rounded-xl shadow shadow-indigo-600/10 transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed min-h-[44px] flex items-center justify-center"
          >
            Create Node
          </button>
        </div>
      {/if}
    </div>
  {/if}
</div>
