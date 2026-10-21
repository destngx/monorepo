<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '$lib/api/endpoints';
  import AuthCard from '$lib/components/AuthCard.svelte';
  import Field from '$lib/components/Field.svelte';
  import { roleLabels } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  const token = $derived(page.params.token ?? '');
  let displayName = $state('');
  let email = $state('');
  let password = $state('');

  // HW-TEN-4: a signed-in user joins at once; anyone else creates an account.
  const accept = action(
    async () => {
      const tenant = await auth.acceptInvite(token);
      // The layout reloads the memberships; then the new tenant can be picked.
      await invalidateAll();
      session.selectTenant(tenant.id);
      await goto(`${base}/`);
    },
    { errorContext: 'Could not join' },
  );
  const signUp = action(
    async () => {
      session.set(await auth.signUp(token, { displayName, email, password }));
      await goto(`${base}/`);
    },
    { errorContext: 'Could not create the account' },
  );
</script>

<AuthCard
  title="Join {data.invite.tenantName}"
  subtitle="You are invited as {roleLabels[data.invite.role].toLowerCase()}."
>
  {#if session.me}
    <p class="text-sm">Signed in as {session.me.user.email}.</p>
    <button class="btn-primary mt-4 w-full" type="button" disabled={accept.busy} onclick={() => void accept.run()}>
      {accept.busy ? 'Joining…' : `Join ${data.invite.tenantName}`}
    </button>
  {:else}
    <form
      class="space-y-4"
      onsubmit={(e) => {
        e.preventDefault();
        void signUp.run();
      }}
    >
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
      <button class="btn-primary w-full" type="submit" disabled={signUp.busy}>
        {signUp.busy ? 'Creating…' : 'Create account and join'}
      </button>
      <p class="muted text-sm">
        Already have an account?
        <a class="link" href="{base}/login?next={encodeURIComponent(`/invite/${token}`)}">Log in first</a>, then open
        this link again.
      </p>
    </form>
  {/if}
</AuthCard>
