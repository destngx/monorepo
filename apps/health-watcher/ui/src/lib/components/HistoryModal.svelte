<script lang="ts">
  import type { AuditEntry } from '$lib/api/types';
  import { formatDate } from '$lib/format';
  import { oldRecord } from '$lib/reading';
  import { Modal } from '@libs/ui-kit';

  // The earlier versions of an approved record (HW-REC-6).
  let { entries, onclose }: { entries: AuditEntry[] | null; onclose: () => void } = $props();

  const shown: Record<string, string> = {
    analyte: 'Test',
    value: 'Result',
    unit: 'Unit',
    refText: 'Reference',
    flag: 'Flag',
    sampleDate: 'Sample date',
    metricCode: 'Catalog entry',
    kind: 'Kind',
    section: 'Section',
    text: 'Text',
    icd10: 'ICD-10',
    drug: 'Medicine',
    dose: 'Dose',
    frequency: 'How often',
    duration: 'For how long',
  };

  function fields(json: string): [string, string][] {
    const rec = oldRecord(json);
    return Object.entries(shown)
      .filter(([k]) => rec[k] !== undefined && rec[k] !== '')
      .map(([k, label]) => [label, String(rec[k])]);
  }
</script>

<Modal open={entries !== null} title="History" {onclose}>
  {#if entries?.length}
    <ol class="space-y-4">
      {#each entries as e (e.id)}
        <li>
          <p class="text-sm font-medium">
            {e.action === 'delete' ? 'Deleted' : 'Changed'} on {formatDate(e.at)}
            {new Date(e.at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
          </p>
          <p class="muted mb-1 text-xs">Before the change:</p>
          <dl class="grid grid-cols-[8rem_1fr] gap-x-3 gap-y-0.5 text-sm">
            {#each fields(e.old) as [k, v] (k)}
              <dt class="muted">{k}</dt>
              <dd class="break-words">{v}</dd>
            {/each}
          </dl>
        </li>
      {/each}
    </ol>
  {:else}
    <p class="muted text-sm">No earlier versions. Changes to approved records show up here.</p>
  {/if}
</Modal>
