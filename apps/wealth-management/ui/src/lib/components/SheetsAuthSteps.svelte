<script lang="ts">
  import type { SheetsAuthKind } from '$lib/api/client';

  // What to do about a Google Sheets credential problem (WM-UI-4), shared by
  // the alert, the error page and the dependency banner.
  let { kind, class: className = '' }: { kind: SheetsAuthKind; class?: string } = $props();

  const code = 'rounded bg-slate-900/10 px-1.5 py-0.5 dark:bg-white/10';
</script>

{#if kind === 'missing'}
  <ol class="list-decimal space-y-1 pl-5 text-left text-sm {className}">
    <li>
      In <code class={code}>.env.local</code>, set <code class={code}>GOOGLE_SHEETS_ID</code>,
      <code class={code}>GOOGLE_CLIENT_ID</code> and <code class={code}>GOOGLE_CLIENT_SECRET</code>.
    </li>
    <li>
      Run <code class={code}>pnpm run auth:sheets</code> to write <code class={code}>GOOGLE_REFRESH_TOKEN</code>.
    </li>
    <li>
      Restart the engine; check with <code class={code}>pnpx nx run wealth-management:verify-auth</code>.
    </li>
  </ol>
{:else}
  <p class="text-sm {className}">
    Re-authorise by running <code class={code}>pnpm run auth:sheets</code> in the repository, then restart the engine. OAuth
    apps in Testing mode issue refresh tokens that expire after 7 days.
  </p>
{/if}
