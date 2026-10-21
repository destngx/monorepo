<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import AuthCard from '$lib/components/AuthCard.svelte';
  import Field from '$lib/components/Field.svelte';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import { auth } from '$lib/api/endpoints';

  let { data } = $props();
  let displayName = $state('');
  let email = $state('');
  let password = $state('');
  let tenantName = $state('');
  let setupCode = $state('');

  // HW-AUTH-4: the first user and family. The code proves access to the Mac.
  const submit = action(
    async () => {
      session.set(await auth.setup({ displayName, email, password, tenantName, setupCode }));
      await goto(`${base}/`);
    },
    { errorContext: 'Setup failed' },
  );
</script>

<AuthCard title="Welcome to Health Watcher" subtitle="Create the first account and your family.">
  {#if !data.status.needsSetup}
    <p class="text-sm">Setup is already done.</p>
    <a class="btn-primary mt-4" href="{base}/login">Log in</a>
  {:else}
    <form
      class="space-y-4"
      onsubmit={(e) => {
        e.preventDefault();
        void submit.run();
      }}
    >
      <Field
        id="code"
        label="Setup code"
        hint="On the Mac, in the setup-code file in the data folder. The engine log shows the path."
      >
        <input
          id="code"
          class="field font-mono uppercase"
          bind:value={setupCode}
          autocomplete="off"
          placeholder="XXXX-XXXX-XXXX"
          required
        />
      </Field>
      <Field id="name" label="Your name">
        <input id="name" class="field" bind:value={displayName} autocomplete="name" required />
      </Field>
      <Field id="email" label="Email">
        <input id="email" class="field" type="email" bind:value={email} autocomplete="email" required />
      </Field>
      <Field id="password" label="Password" hint="At least 10 characters.">
        <input
          id="password"
          class="field"
          type="password"
          bind:value={password}
          autocomplete="new-password"
          minlength="10"
          required
        />
      </Field>
      <Field id="tenant" label="Family or group name" hint="For example: Nguyen family.">
        <input id="tenant" class="field" bind:value={tenantName} required />
      </Field>
      <button class="btn-primary w-full" type="submit" disabled={submit.busy} aria-busy={submit.busy || undefined}>
        {submit.busy ? 'Creating…' : 'Create account'}
      </button>
    </form>
  {/if}
</AuthCard>
