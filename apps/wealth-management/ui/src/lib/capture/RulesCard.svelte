<script lang="ts">
  import type { CaptureRule } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import { ruleKey } from './review';

  // Learned rules (WM-CAP-11): offered after rows are written.
  interface Props {
    rules: CaptureRule[];
    /** Keys of rules the user unticked. */
    skipped: string[];
    saving: boolean;
    saveCount: number;
    onsave: () => void;
  }

  let { rules, skipped = $bindable(), saving, saveCount, onsave }: Props = $props();
</script>

<Card title="Remember for next time" subtitle="Saved to the CaptureRules tab and used by later captures." class="mt-4">
  <ul class="space-y-1 text-sm">
    {#each rules as rule (ruleKey(rule))}
      <li>
        <label class="inline-flex items-center gap-2">
          <input
            type="checkbox"
            checked={!skipped.includes(ruleKey(rule))}
            onchange={(event) =>
              (skipped = event.currentTarget.checked
                ? skipped.filter((key) => key !== ruleKey(rule))
                : [...skipped, ruleKey(rule)])}
          />
          {#if rule.kind === 'alias'}
            “{rule.pattern}” is account <strong>{rule.value}</strong>
          {:else}
            Payee “{rule.pattern}” is category <strong>{rule.value}</strong>
          {/if}
        </label>
      </li>
    {/each}
  </ul>
  <div class="mt-3 flex justify-end">
    <button type="button" class="btn-ghost" disabled={saving || !saveCount} onclick={onsave}>
      {saving ? 'Saving…' : `Save ${saveCount} rule${saveCount === 1 ? '' : 's'}`}
    </button>
  </div>
</Card>
