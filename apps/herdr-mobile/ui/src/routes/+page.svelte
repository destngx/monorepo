<script lang="ts">
  import { base } from '$app/paths';
  import { Badge, EmptyState, Icon, STATUS, ThemeToggle } from '@libs/ui-kit';
  import { agentName, groupAgents, statusOf, summary } from '$lib/agents';
  import ConnectionDot from '$lib/components/ConnectionDot.svelte';
  import { live } from '$lib/state/live.svelte';
  import { theme } from '$lib/state/theme.svelte';

  const groups = $derived(groupAgents(live.agents));
  const waiting = $derived(live.agents.filter((a) => a.status === 'blocked').length);
</script>

<svelte:head><title>{waiting > 0 ? `(${waiting}) ` : ''}Agents</title></svelte:head>

<div class="mx-auto max-w-xl px-3 pb-[max(1.5rem,env(safe-area-inset-bottom))]">
  <header class="sticky top-0 z-30 -mx-3 bg-canvas/90 px-3 pt-[max(0.75rem,env(safe-area-inset-top))] pb-2">
    <div class="flex items-center gap-2">
      <h1 class="text-3xl leading-tight">Agents</h1>
      <ConnectionDot connection={live.connection} />
      <div class="ml-auto"><ThemeToggle {theme} /></div>
    </div>
  </header>

  <main class="mt-2">
    {#if !live.loaded}
      <div role="status" aria-busy="true" class="space-y-3">
        <span class="sr-only">Loading agents…</span>
        {#each [0, 1, 2] as i (i)}<div class="skeleton h-20"></div>{/each}
      </div>
    {:else}
      <p
        class="panel mb-5 border-l-4 px-4 py-3 text-lg font-medium {waiting > 0
          ? STATUS.warning.edge
          : 'border-l-accent-500'}"
      >
        {summary(live.agents)}
      </p>

      {#if live.error}
        <div role="alert" class="panel mb-5 border-l-4 px-4 py-3 {STATUS.critical.edge}">
          <p class="font-medium">Could not load agents</p>
          <p class="muted mt-0.5 text-sm">{live.error}</p>
        </div>
      {:else if live.agents.length === 0}
        <div class="panel"><EmptyState title="No agents yet" detail="Start one in herdr on the laptop." /></div>
      {/if}

      {#each groups as group (group.status)}
        <section class="mb-6" aria-labelledby="group-{group.status}">
          <h2 id="group-{group.status}" class="eyebrow mb-2 font-sans">
            {statusOf(group.status).label} · {group.agents.length}
          </h2>
          <ul class="space-y-2">
            {#each group.agents as agent (agent.paneId)}
              {@const status = statusOf(agent.status)}
              <li>
                <a
                  href="{base}/agents/{encodeURIComponent(agent.paneId)}"
                  class="panel interactive flex min-h-16 items-center gap-3 px-4 py-3 {agent.status === 'blocked'
                    ? `border-l-4 ${STATUS.warning.edge}`
                    : ''}"
                >
                  <div class="min-w-0 flex-1">
                    <p class="line-clamp-2 font-medium break-words">{agentName(agent)}</p>
                    <p class="muted mt-1 flex min-w-0 items-center gap-2 text-sm">
                      <Badge status={status.tone}>{status.label}</Badge>
                      <span class="truncate">{agent.workspace || agent.workspaceId} · {agent.kind || 'agent'}</span>
                    </p>
                  </div>
                  <Icon name="chevronRight" size={18} class="muted shrink-0" />
                </a>
              </li>
            {/each}
          </ul>
        </section>
      {/each}
    {/if}
  </main>
</div>
