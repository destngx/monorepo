<script lang="ts">
  // A small busy indicator; decorative, so the busy control carries the text and aria-busy.
  let { size = 16, class: className = '' }: { size?: number; class?: string } = $props();
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  aria-hidden="true"
  class="shrink-0 animate-spin motion-reduce:animate-[spin_2s_linear_infinite] {className}"
  data-spinner
>
  <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="2.5" opacity="0.25" />
  <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" />
</svg>
