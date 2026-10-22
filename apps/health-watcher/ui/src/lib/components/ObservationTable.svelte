<script lang="ts">
  import type { Observation, RecordStatus } from '$lib/api/types';
  import { flagLabel, formatDate } from '$lib/format';
  import { catalog } from '$lib/state/catalog.svelte';
  import Button from './Button.svelte';

  // Observations as a table. Drafts can be edited, rejected and restored;
  // approved ones edited, deleted and their history shown (HW-READ-4, HW-REC-6).
  interface Props {
    observations: Observation[];
    editable: boolean;
    pageNumber: (pageId: string | undefined) => number | undefined;
    onedit: (o: Observation) => void;
    onstatus?: (o: Observation, status: RecordStatus) => Promise<unknown>;
    ondelete?: (o: Observation) => Promise<unknown>;
    onhistory?: (o: Observation) => Promise<unknown>;
  }

  let { observations, editable, pageNumber, onedit, onstatus, ondelete, onhistory }: Props = $props();

  const flagTone: Record<string, string> = {
    H: 'text-danger-700 dark:text-danger-300',
    L: 'text-info-700 dark:text-info-300',
    '*': 'text-warn-700 dark:text-warn-300',
  };
</script>

<div class="overflow-x-auto">
  <table class="w-full text-sm">
    <thead class="muted text-left text-xs">
      <tr>
        <th class="py-2 pr-3 font-medium">Test</th>
        <th class="py-2 pr-3 font-medium">Result</th>
        <th class="hidden py-2 pr-3 font-medium sm:table-cell">Reference</th>
        <th class="hidden py-2 pr-3 font-medium md:table-cell">Sampled</th>
        <th class="py-2 font-medium"><span class="sr-only">Actions</span></th>
      </tr>
    </thead>
    <tbody class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
      {#each observations as o (o.id)}
        {@const metric = o.metricCode ? catalog.byCode.get(o.metricCode) : undefined}
        {@const page = pageNumber(o.pageId)}
        <tr class={o.status === 'rejected' ? 'opacity-50' : ''}>
          <td class="py-2 pr-3 align-top">
            <span class="font-medium {o.status === 'rejected' ? 'line-through' : ''}">{o.analyte}</span>
            <span class="muted block text-xs">
              {metric ? metric.name : 'Not in the catalog'}{page ? ` · page ${page}` : ''}
              {#if o.status === 'draft' && o.confidence < 70}
                · <span class="text-warn-700 dark:text-warn-300">check this ({o.confidence}%)</span>
              {/if}
            </span>
          </td>
          <td class="py-2 pr-3 align-top tabular">
            <span class="font-medium {flagTone[o.flag] ?? ''}">{o.value}</span>
            {o.unit}
            {#if o.flag}<span class="text-xs {flagTone[o.flag] ?? ''}" title={flagLabel(o.flag)}>{o.flag}</span>{/if}
          </td>
          <td class="muted hidden py-2 pr-3 align-top tabular sm:table-cell">{o.refText}</td>
          <td class="muted hidden py-2 pr-3 align-top md:table-cell">{formatDate(o.sampleDate)}</td>
          <td class="py-2 text-right align-top whitespace-nowrap">
            {#if editable && o.status !== 'rejected'}
              <Button size="icon" icon="pencil" label="Edit" onclick={() => onedit(o)} />
            {/if}
            {#if editable && onstatus && o.status === 'draft'}
              <Button
                size="icon"
                icon="x"
                label="Reject"
                errorContext="Could not reject"
                onclick={() => onstatus(o, 'rejected')}
              />
            {:else if editable && onstatus && o.status === 'rejected'}
              <Button size="sm" errorContext="Could not restore" onclick={() => onstatus(o, 'draft')}>Restore</Button>
            {/if}
            {#if o.status === 'approved' && onhistory}
              <Button
                size="icon"
                icon="clock"
                label="History"
                errorContext="Could not load history"
                onclick={() => onhistory(o)}
              />
            {/if}
            {#if editable && ondelete && o.status === 'approved'}
              <Button
                size="icon"
                icon="trash"
                label="Delete"
                errorContext="Could not delete"
                onclick={() => ondelete(o)}
              />
            {/if}
          </td>
        </tr>
      {/each}
    </tbody>
  </table>
</div>
