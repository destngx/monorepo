<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { EmptyState, Icon, PageTitle } from '@libs/ui-kit';
  import { formatDate } from '$lib/format';
  import { hitHref, hitTitle } from '$lib/search';

  // Search over page text, findings and visits, ignoring case and accents (HW-FIND-2).
  let { data } = $props();
  let q = $state('');

  $effect(() => {
    q = data.q;
  });

  const kindLabels = { page: 'Page text', finding: 'Finding', visit: 'Visit' };
</script>

<PageTitle
  title="Search"
  subtitle="Words from any paper, finding or visit. Accents are optional: 'kham suc khoe' finds 'khám sức khỏe'."
/>

<form
  class="mb-5 flex gap-2"
  role="search"
  onsubmit={(e) => {
    e.preventDefault();
    void goto(`${base}/search?q=${encodeURIComponent(q.trim())}`, { keepFocus: true });
  }}
>
  <label class="sr-only" for="search-q">Search</label>
  <input
    id="search-q"
    class="field"
    type="search"
    bind:value={q}
    maxlength="200"
    placeholder="e.g. glucose, viêm họng"
  />
  <button class="btn-primary" type="submit"><Icon name="search" size={16} />Search</button>
</form>

{#if data.result}
  <p class="muted mb-3 text-sm">
    {data.result.hits.length === 0
      ? 'Nothing found.'
      : `${data.result.hits.length}${data.result.hits.length === 50 ? '+' : ''} results`}
  </p>
  <ol class="panel divide-y divide-ink-900/5 dark:divide-ink-50/5">
    {#each data.result.hits as hit, i (i)}
      <li>
        <a href="{base}{hitHref(hit)}" class="row-link block px-4 py-3">
          <span class="flex flex-wrap items-baseline gap-x-2">
            <span class="font-medium">{hitTitle(hit)}</span>
            <span class="muted text-xs">
              {kindLabels[hit.kind]}{hit.pageNumber ? `, page ${hit.pageNumber}` : ''}
              {hit.personName ? ` · ${hit.personName}` : ''}{hit.date
                ? ` · ${formatDate(hit.date)}`
                : ''}{hit.providerName ? ` · ${hit.providerName}` : ''}
            </span>
          </span>
          <span class="mt-1 block text-sm break-words">
            {#each hit.snippet ?? [] as part, j (j)}{#if part.match}<mark
                  class="rounded bg-warn-300/60 px-0.5 text-inherit dark:bg-warn-400/30">{part.text}</mark
                >{:else}{part.text}{/if}{/each}
          </span>
        </a>
      </li>
    {:else}
      <li><EmptyState title="No matches" detail="Try fewer or shorter words." /></li>
    {/each}
  </ol>
{/if}
