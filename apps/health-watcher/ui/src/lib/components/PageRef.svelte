<script lang="ts">
  // Where a value or finding was read: its page's name, a link to the page
  // when the page can be opened.
  let { page, href }: { page: { name: string } | undefined; href?: string } = $props();
</script>

{#if page}
  · {#if href}<a class="link" {href}>{page.name}</a>{:else}{page.name}{/if}
{/if}
