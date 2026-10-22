<script lang="ts">
  import { base } from '$app/paths';
  import type { SheetsAuthKind } from '$lib/api/client';

  // What to do about a Google Sheets credential problem (WM-UI-4), shared by
  // the alert, the error page and the dependency banner.
  let { kind, class: className = '' }: { kind: SheetsAuthKind; class?: string } = $props();

  const code = 'rounded bg-stone-900/10 px-1.5 py-0.5 dark:bg-white/10';
</script>

{#if kind === 'missing'}
  <ol class="list-decimal space-y-1 pl-5 text-left text-sm {className}">
    <li>
      In <a class="underline" href="{base}/settings">Settings</a> under Google Sheets, set
      <code class={code}>GOOGLE_SHEETS_ID</code>, <code class={code}>GOOGLE_CLIENT_ID</code> and
      <code class={code}>GOOGLE_CLIENT_SECRET</code>.
    </li>
    <li>
      Save them, then press <strong>Connect Google account</strong> to save
      <code class={code}>GOOGLE_REFRESH_TOKEN</code>.
    </li>
    <li>
      Restart the engine from Settings; check with <code class={code}>pnpx nx run wealth-management:verify-auth</code>.
    </li>
  </ol>
{:else}
  <p class="text-sm {className}">
    Re-authorise with <strong>Connect Google account</strong> in
    <a class="underline" href="{base}/settings">Settings</a>
    under Google Sheets, then restart the engine. OAuth apps in Testing mode issue refresh tokens that expire after 7 days.
  </p>
{/if}
