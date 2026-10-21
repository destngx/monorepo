<script lang="ts">
  import { goto } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '$lib/api/endpoints';
  import AuthCard from '$lib/components/AuthCard.svelte';
  import Field from '$lib/components/Field.svelte';
  import { safeNext } from '$lib/paths';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';

  let email = $state('');
  let password = $state('');

  const submit = action(
    async () => {
      session.set(await auth.login(email, password));
      password = '';
      await goto(`${base}${safeNext(page.url.searchParams.get('next'))}`, { invalidateAll: true });
    },
    { errorContext: 'Could not log in' },
  );
</script>

<AuthCard title="Log in" subtitle="Health Watcher">
  <form
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void submit.run();
    }}
  >
    <Field id="email" label="Email">
      <input id="email" class="field" type="email" bind:value={email} autocomplete="email" required />
    </Field>
    <Field id="password" label="Password">
      <input
        id="password"
        class="field"
        type="password"
        bind:value={password}
        autocomplete="current-password"
        required
      />
    </Field>
    <button class="btn-primary w-full" type="submit" disabled={submit.busy} aria-busy={submit.busy || undefined}>
      {submit.busy ? 'Logging in…' : 'Log in'}
    </button>
    <p class="muted text-xs">Forgot your password? Ask whoever runs the app on the Mac to reset it.</p>
  </form>
</AuthCard>
