<script lang="ts">
  import { toasts } from '../state/toasts.svelte';
  import Icon from './Icon.svelte';

  // A read-only value with a copy button, for links and tokens shown once.
  let { value, label }: { value: string; label: string } = $props();

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      toasts.success('Copied');
    } catch {
      toasts.warning('Copy is blocked here; select the text and copy it by hand.');
    }
  }
</script>

<div class="flex gap-2">
  <input
    class="field font-mono text-xs"
    readonly
    {value}
    aria-label={label}
    onfocus={(e) => e.currentTarget.select()}
  />
  <button class="btn-ghost shrink-0" type="button" onclick={copy}><Icon name="copy" size={16} />Copy</button>
</div>
