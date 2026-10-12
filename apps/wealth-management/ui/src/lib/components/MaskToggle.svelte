<script lang="ts">
  import { mask } from '$lib/state/mask.svelte';
  import Icon from './Icon.svelte';
</script>

<button
  type="button"
  class="btn-ghost px-2.5"
  aria-pressed={mask.revealed}
  aria-label={mask.revealed ? 'Hide amounts' : 'Show amounts'}
  title={mask.revealed ? 'Hide amounts' : 'Show amounts'}
  onclick={() => mask.toggle()}
>
  <Icon name={mask.revealed ? 'eye' : 'eyeOff'} />
</button>
