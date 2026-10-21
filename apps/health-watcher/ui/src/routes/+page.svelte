<script lang="ts">
  import { base } from '$app/paths';
  import Icon from '$lib/components/Icon.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import { ageOn, formatDate, todayISO } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  const today = todayISO();
</script>

<PageTitle
  title={session.membership?.tenant.name ?? 'Home'}
  subtitle="Health records for everyone in your family or group."
>
  {#snippet actions()}
    {#if session.canEdit}
      <a class="btn-primary" href="{base}/inbox"><Icon name="upload" size={16} />Add records</a>
    {/if}
  {/snippet}
</PageTitle>

{#if data.inboxCount > 0}
  <a href="{base}/inbox" class="glass interactive mb-5 flex items-center gap-3 p-4">
    <span class="grid size-10 place-items-center rounded-xl bg-amber-500/15 text-amber-700 dark:text-amber-300"
      ><Icon name="inbox" /></span
    >
    <span class="flex-1">
      <span class="block font-medium"
        >{data.inboxCount} {data.inboxCount === 1 ? 'item' : 'items'} waiting in the inbox</span
      >
      <span class="muted text-sm">File them under a visit so they show up in the timeline.</span>
    </span>
    <Icon name="chevronRight" />
  </a>
{/if}

<div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
  {#each data.people as p (p.id)}
    {@const age = ageOn(p.birthDate, today)}
    <a href="{base}/people/{p.id}" class="glass interactive flex items-center gap-4 p-4">
      <span
        class="grid size-12 shrink-0 place-items-center rounded-2xl bg-teal-600/10 text-lg font-semibold text-teal-700 dark:text-teal-300"
      >
        {p.displayName.slice(0, 1).toUpperCase()}
      </span>
      <span class="min-w-0">
        <span class="block truncate font-medium">{p.displayName}</span>
        <span class="muted block text-sm">
          {age !== null ? `${age} years · ` : ''}{p.visitCount}
          {p.visitCount === 1 ? 'visit' : 'visits'}
        </span>
        <span class="muted block text-xs"
          >{p.lastVisitOn ? `Last visit ${formatDate(p.lastVisitOn)}` : 'No visits yet'}</span
        >
      </span>
    </a>
  {/each}
  {#if session.canEdit}
    <a
      href="{base}/people"
      class="glass interactive flex items-center justify-center gap-2 p-4 text-sm text-teal-700 dark:text-teal-300"
    >
      <Icon name="plus" size={16} />Add a person
    </a>
  {/if}
</div>
