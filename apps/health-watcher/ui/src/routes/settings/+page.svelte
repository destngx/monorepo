<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { auth } from '$lib/api/endpoints';
  import type { CreatedInvite, CreatedUploadToken, Member, Role } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import CopyField from '$lib/components/CopyField.svelte';
  import Field from '$lib/components/Field.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import { formatDate, roleHints, roleLabels, options } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import { theme, type Theme } from '$lib/state/theme.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  let { data } = $props();

  const origin = $derived(page.url.origin);
  const me = $derived(session.me?.user);

  // Group
  let tenantName = $state('');
  // Follows the active tenant, so Rename never sends one group's name to another.
  $effect(() => {
    tenantName = session.membership?.tenant.name ?? '';
  });
  const rename = action(
    async () => {
      await session.api.rename(tenantName);
      await invalidateAll();
    },
    { errorContext: 'Could not rename', success: 'Renamed' },
  );
  let newTenant = $state('');
  const createTenant = action(
    async () => {
      const t = await auth.createTenant(newTenant);
      newTenant = '';
      await invalidateAll();
      session.selectTenant(t.id);
      await invalidateAll();
    },
    { errorContext: 'Could not create', success: 'Created and switched to it' },
  );

  // Members
  async function setRole(m: Member, select: HTMLSelectElement) {
    try {
      await session.api.setRole(m.userId, select.value as Role);
      await invalidateAll();
    } catch (error) {
      select.value = m.role;
      toasts.error(error, 'Could not change the role');
    }
  }
  async function removeMember(m: Member) {
    const self = m.userId === me?.id;
    if (
      !confirm(self ? 'Leave this group? You lose access to its records.' : `Remove ${m.displayName} from this group?`)
    )
      return;
    await session.api.removeMember(m.userId);
    if (self) {
      await goto(`${base}/`);
    }
    await invalidateAll();
  }

  // Invites (HW-TEN-4)
  let inviteRole = $state<Role>('editor');
  let createdInvite = $state<CreatedInvite | null>(null);
  const invite = action(
    async () => {
      createdInvite = await session.api.createInvite(inviteRole);
      await invalidateAll();
    },
    { errorContext: 'Could not create the invite' },
  );
  async function revokeInvite(id: string) {
    await session.api.revokeInvite(id);
    if (createdInvite?.invite.id === id) createdInvite = null;
    await invalidateAll();
  }

  // Upload tokens (HW-AUTH-3)
  let tokenName = $state('iPhone Shortcut');
  let createdToken = $state<CreatedUploadToken | null>(null);
  const createToken = action(
    async () => {
      createdToken = await session.api.createUploadToken(tokenName);
      await invalidateAll();
    },
    { errorContext: 'Could not create the token' },
  );
  async function revokeToken(id: string) {
    if (!confirm('Revoke this token? A Shortcut using it stops working.')) return;
    await session.api.revokeUploadToken(id);
    if (createdToken?.uploadToken.id === id) createdToken = null;
    await invalidateAll();
  }

  // Password
  let current = $state('');
  let next = $state('');
  const changePassword = action(
    async () => {
      session.set(await auth.changePassword(current, next));
      current = '';
      next = '';
    },
    { errorContext: 'Could not change the password', success: 'Password changed; other devices are signed out' },
  );

  const themes: { value: Theme; label: string }[] = [
    { value: 'system', label: 'System' },
    { value: 'light', label: 'Light' },
    { value: 'dark', label: 'Dark' },
  ];

  function submit(run: () => Promise<boolean>) {
    return (e: SubmitEvent) => {
      e.preventDefault();
      void run();
    };
  }
</script>

<PageTitle title="Settings" subtitle="Signed in as {me?.email}" />

<div class="grid gap-5 lg:grid-cols-2">
  <Card title="Family or group" subtitle="Everything stored belongs to one group.">
    {#if session.canManage}
      <form class="flex gap-2" onsubmit={submit(rename.run)}>
        <input class="field" bind:value={tenantName} aria-label="Group name" maxlength="80" required />
        <button class="btn-ghost shrink-0" type="submit" disabled={rename.busy}>Rename</button>
      </form>
    {:else}
      <p class="text-sm">
        {session.membership?.tenant.name} · you are {session.role ? roleLabels[session.role].toLowerCase() : ''}
      </p>
    {/if}
    <form class="mt-4 flex gap-2" onsubmit={submit(createTenant.run)}>
      <input
        class="field"
        bind:value={newTenant}
        placeholder="New group, e.g. Parents"
        aria-label="New group name"
        maxlength="80"
        required
      />
      <button class="btn-ghost shrink-0" type="submit" disabled={createTenant.busy}>Create</button>
    </form>
    <p class="muted mt-2 text-xs">You own any group you create. Switch groups from the header.</p>
  </Card>

  <Card title="Members">
    <ul class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
      {#each data.members as m (m.userId)}
        <li class="flex items-center gap-3 py-2">
          <div class="min-w-0 flex-1">
            <p class="truncate text-sm font-medium">{m.displayName}{m.userId === me?.id ? ' (you)' : ''}</p>
            <p class="muted truncate text-xs">{m.email}</p>
          </div>
          {#if session.canManage}
            <select
              class="field w-auto py-1 text-xs"
              value={m.role}
              aria-label="Role of {m.displayName}"
              onchange={(e) => void setRole(m, e.currentTarget)}
            >
              {#each options(roleLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
            </select>
          {:else}
            <span class="muted text-xs">{roleLabels[m.role]}</span>
          {/if}
          {#if session.canManage || m.userId === me?.id}
            <Button size="sm" errorContext="Could not remove" onclick={() => removeMember(m)}
              >{m.userId === me?.id ? 'Leave' : 'Remove'}</Button
            >
          {/if}
        </li>
      {/each}
    </ul>
  </Card>

  {#if session.canManage}
    <Card title="Invite someone" subtitle="A one-time link, valid for 7 days. Send it in a private chat.">
      <form class="flex flex-wrap gap-2" onsubmit={submit(invite.run)}>
        <select class="field w-auto" bind:value={inviteRole} aria-label="Role">
          {#each options(roleLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
        </select>
        <button class="btn-primary" type="submit" disabled={invite.busy}>Create link</button>
      </form>
      <p class="muted mt-2 text-xs">{roleHints[inviteRole]}</p>
      {#if createdInvite}
        <div class="mt-4">
          <CopyField label="Invite link" value="{origin}{base}/invite/{createdInvite.token}" />
          <p class="muted mt-1 text-xs">This link is shown once.</p>
        </div>
      {/if}
      {#if data.invites.length}
        <h3 class="label mt-5">Pending</h3>
        <ul class="divide-y divide-ink-900/5 text-sm dark:divide-ink-50/5">
          {#each data.invites as inv (inv.id)}
            <li class="flex items-center justify-between py-2">
              <span>{roleLabels[inv.role]} · expires {formatDate(inv.expiresAt)}</span>
              <Button size="sm" errorContext="Could not revoke" onclick={() => revokeInvite(inv.id)}>Revoke</Button>
            </li>
          {/each}
        </ul>
      {/if}
    </Card>
  {/if}

  {#if session.canEdit}
    <Card title="iPhone Shortcut" subtitle="Send scans and PDFs from the share sheet straight to the inbox.">
      <form class="flex gap-2" onsubmit={submit(createToken.run)}>
        <input class="field" bind:value={tokenName} aria-label="Token name" maxlength="60" required />
        <button class="btn-primary shrink-0" type="submit" disabled={createToken.busy}>Create token</button>
      </form>
      {#if createdToken}
        <div class="mt-4 space-y-3">
          <Field id="shortcut-url" label="URL for Get Contents of URL (POST, form, field file)">
            <CopyField label="Upload URL" value="{origin}/api/inbox" />
          </Field>
          <Field
            id="shortcut-token"
            label="Authorization header value"
            hint="Shown once. It can only upload to this group's inbox."
          >
            <CopyField label="Header value" value="Bearer {createdToken.token}" />
          </Field>
        </div>
      {/if}
      {#if data.tokens.length}
        <h3 class="label mt-5">Your tokens in this group</h3>
        <ul class="divide-y divide-ink-900/5 text-sm dark:divide-ink-50/5">
          {#each data.tokens as tok (tok.id)}
            <li class="flex items-center justify-between gap-2 py-2">
              <span class="min-w-0">
                <span class="block truncate">{tok.name}</span>
                <span class="muted text-xs"
                  >{tok.lastUsedAt ? `Last used ${formatDate(tok.lastUsedAt)}` : 'Never used'}</span
                >
              </span>
              <Button size="sm" errorContext="Could not revoke" onclick={() => revokeToken(tok.id)}>Revoke</Button>
            </li>
          {/each}
        </ul>
      {/if}
      <p class="muted mt-4 text-xs">Setup steps: docs/health-watcher/_setup/iphone-shortcut.md</p>
    </Card>
  {/if}

  <Card title="Password">
    <form class="space-y-3" onsubmit={submit(changePassword.run)}>
      <Field id="pw-current" label="Current password">
        <input
          id="pw-current"
          class="field"
          type="password"
          bind:value={current}
          autocomplete="current-password"
          required
        />
      </Field>
      <Field id="pw-next" label="New password" hint="At least 10 characters. Other devices get signed out.">
        <input
          id="pw-next"
          class="field"
          type="password"
          bind:value={next}
          autocomplete="new-password"
          minlength="10"
          required
        />
      </Field>
      <button class="btn-primary" type="submit" disabled={changePassword.busy}>Change password</button>
    </form>
  </Card>

  {#if me?.admin}
    <Card title="Engine" subtitle="AI address and key, mac-ocr, background jobs, upload limits, host and port.">
      <a class="btn-ghost" href="{base}/settings/engine">Engine settings</a>
      <p class="muted mt-2 text-xs">
        They apply to every family on this Mac. You see this because you are an engine admin.
      </p>
    </Card>
  {/if}

  <Card title="Appearance">
    <div class="flex gap-2">
      {#each themes as t (t.value)}
        <button
          type="button"
          class="btn-ghost {theme.value === t.value ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300' : ''}"
          aria-pressed={theme.value === t.value}
          onclick={() => theme.set(t.value)}>{t.label}</button
        >
      {/each}
    </div>
  </Card>
</div>
