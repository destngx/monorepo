<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { GoogleAuthPurpose, GoogleAuthStart } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { signInHere } from './googleAuth';

  interface Props {
    purpose: GoogleAuthPurpose;
    label: string;
    disabled?: boolean;
    /** Why the button is disabled. */
    disabledReason?: string;
    /** Reloads the settings; a sign-in may have saved a token meanwhile. */
    onchange: () => Promise<void>;
  }

  let { purpose, label, disabled = false, disabledReason, onchange }: Props = $props();

  // The sign-in runs in this tab: Google returns to the engine's callback,
  // which saves the refresh token and sends the browser back to Settings
  // (see googleAuthNotice). When Google returns somewhere this browser may
  // not reach, the sign-in opens in a new tab and the address it ends on is
  // pasted here instead.
  let started = $state<GoogleAuthStart | null>(null);
  let starting = $state(false);
  let pasted = $state('');
  let completing = $state(false);
  let pasteError = $state('');

  async function connect() {
    starting = true;
    try {
      const start = await api.startGoogleAuth(purpose);
      // Leaving for Google; the button stays busy until the page unloads.
      if (signInHere(start, window.location)) return;
      started = start;
      pasted = '';
      pasteError = '';
    } catch (failure) {
      toasts.error(failure, label);
    }
    starting = false;
  }

  async function complete() {
    if (!pasted.trim() || completing) return;
    completing = true;
    pasteError = '';
    try {
      const result = await api.completeGoogleAuth(pasted);
      toasts.push('success', `Saved ${result.settingKey}. Restart the engine to use it.`);
      await finish();
    } catch (failure) {
      pasteError = failure instanceof Error ? failure.message : String(failure);
    } finally {
      completing = false;
    }
  }

  async function finish() {
    if (!started) return;
    started = null;
    await onchange();
  }

  function refresh() {
    if (started) void onchange();
  }
</script>

<svelte:window onfocus={refresh} />

<Button
  busy={starting}
  busyText="Opening Google…"
  {disabled}
  title={disabled ? disabledReason : undefined}
  onclick={() => void connect()}>{label}</Button
>

<Modal open={started !== null} title={label} onclose={finish}>
  {#if started}
    <div class="space-y-3 text-sm">
      <p>
        <a class="link" href={started.authUrl} target="_blank" rel="noopener noreferrer">Open Google sign-in</a> in a
        new tab and allow access. Google then returns to
        <code class="font-mono break-all">{started.redirectUrl}</code>, which the OAuth client must allow and this
        browser may not reach.
      </p>
      <p class="muted text-xs">
        If that tab shows Settings, the account is connected: press Done. The sign-in expires after 10 minutes.
      </p>
      <!-- Not a form: this renders inside the settings form. -->
      <div class="space-y-2">
        <label class="label" for="google-redirect-{purpose}"
          >If the tab ended on a page that cannot be reached, paste its address</label
        >
        <div class="flex gap-2">
          <input
            id="google-redirect-{purpose}"
            class="field font-mono"
            type="url"
            autocomplete="off"
            spellcheck="false"
            placeholder={`${started.redirectUrl}?state=...&code=...`}
            aria-invalid={pasteError ? true : undefined}
            aria-describedby={pasteError ? `google-redirect-${purpose}-error` : undefined}
            bind:value={pasted}
            onkeydown={(event) => {
              if (event.key !== 'Enter') return;
              event.preventDefault();
              void complete();
            }}
          />
          <Button busy={completing} busyText="Saving…" disabled={!pasted.trim()} onclick={() => void complete()}
            >Complete</Button
          >
        </div>
        {#if pasteError}<p
            id="google-redirect-{purpose}-error"
            class="text-sm text-rose-600 dark:text-rose-400"
            role="alert"
          >
            {pasteError}
          </p>{/if}
      </div>
    </div>
  {/if}
  {#snippet footer()}
    <Button variant="primary" onclick={finish}>Done</Button>
  {/snippet}
</Modal>
