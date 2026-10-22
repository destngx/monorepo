<script lang="ts">
  import type { Finding, RecordStatus } from '$lib/api/types';
  import { findingKindLabels } from '$lib/format';
  import Button from './Button.svelte';

  // Findings as a list, with the same actions as ObservationTable.
  interface Props {
    findings: Finding[];
    editable: boolean;
    pageNumber: (pageId: string | undefined) => number | undefined;
    onedit: (f: Finding) => void;
    onstatus?: (f: Finding, status: RecordStatus) => Promise<unknown>;
    ondelete?: (f: Finding) => Promise<unknown>;
    onhistory?: (f: Finding) => Promise<unknown>;
  }

  let { findings, editable, pageNumber, onedit, onstatus, ondelete, onhistory }: Props = $props();
</script>

<ul class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
  {#each findings as f (f.id)}
    {@const page = pageNumber(f.pageId)}
    <li class="flex items-start gap-3 py-2 {f.status === 'rejected' ? 'opacity-50' : ''}">
      <div class="min-w-0 flex-1">
        <p class="muted text-xs">
          {findingKindLabels[f.kind]}{f.icd10 ? ` · ${f.icd10}` : ''}{page ? ` · page ${page}` : ''}
          {#if f.status === 'draft' && f.confidence < 70}
            · <span class="text-warn-700 dark:text-warn-300">check this ({f.confidence}%)</span>
          {/if}
        </p>
        <p class="text-sm whitespace-pre-wrap {f.status === 'rejected' ? 'line-through' : ''}">{f.text}</p>
        {#if f.kind === 'prescription' && (f.drug || f.dose || f.frequency || f.duration)}
          <p class="muted text-xs">{[f.drug, f.dose, f.frequency, f.duration].filter(Boolean).join(' · ')}</p>
        {/if}
      </div>
      <div class="shrink-0 whitespace-nowrap">
        {#if editable && f.status !== 'rejected'}
          <Button size="icon" icon="pencil" label="Edit" onclick={() => onedit(f)} />
        {/if}
        {#if editable && onstatus && f.status === 'draft'}
          <Button
            size="icon"
            icon="x"
            label="Reject"
            errorContext="Could not reject"
            onclick={() => onstatus(f, 'rejected')}
          />
        {:else if editable && onstatus && f.status === 'rejected'}
          <Button size="sm" errorContext="Could not restore" onclick={() => onstatus(f, 'draft')}>Restore</Button>
        {/if}
        {#if f.status === 'approved' && onhistory}
          <Button
            size="icon"
            icon="clock"
            label="History"
            errorContext="Could not load history"
            onclick={() => onhistory(f)}
          />
        {/if}
        {#if editable && ondelete && f.status === 'approved'}
          <Button size="icon" icon="trash" label="Delete" errorContext="Could not delete" onclick={() => ondelete(f)} />
        {/if}
      </div>
    </li>
  {/each}
</ul>
