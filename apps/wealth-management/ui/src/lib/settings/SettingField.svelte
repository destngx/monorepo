<script lang="ts">
  import type { EngineSetting } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import type { EngineSettingsForm } from './engine.svelte';
  import { choiceOptions } from './sections';

  // One setting as a label/control row. The env-style key is in the label's
  // tooltip; it is what engine errors and the docs name.
  let { setting, form }: { setting: EngineSetting; form: EngineSettingsForm } = $props();

  const id = $derived(`setting-${setting.key}`);
  const draft = $derived(form.drafts[setting.key]);
  // Drafts stay strings: bind:value on a number input would store a number.

  // What an empty value means: the inherited setting's value, the default, or nothing.
  const emptyLabel = $derived.by(() => {
    if (setting.inherits) {
      const inherited = form.effectiveValue(setting.inherits);
      return inherited ? `Default (${inherited})` : 'Default';
    }
    if (setting.default) return `Default (${setting.default})`;
    return setting.kind === 'choice' ? 'None' : 'Not set';
  });
  const placeholder = $derived(setting.secret ? (setting.isSet ? 'Stored; type to replace' : 'Not set') : emptyLabel);
  // Model settings become a select once the endpoint listed its models.
  const options = $derived(
    setting.kind === 'choice'
      ? (setting.options ?? [])
      : setting.kind === 'model' && form.models?.length
        ? form.models
        : null,
  );
</script>

{#if draft}
  <div class="grid grid-cols-1 gap-x-6 gap-y-1.5 py-3 sm:grid-cols-[12rem_minmax(0,1fr)] sm:items-center">
    <div class="min-w-0">
      <label class="text-sm font-medium" for={id} title={setting.key}>{setting.label}</label>
      {#if setting.description}<p class="muted text-xs">{setting.description}</p>{/if}
    </div>
    <div class="flex min-w-0 items-center gap-2">
      {#if options}
        <select {id} class="field" bind:value={form.drafts[setting.key].value}>
          <option value="">{emptyLabel}</option>
          {#each choiceOptions(options, draft.value) as option (option)}
            <option value={option}>{option}</option>
          {/each}
        </select>
      {:else}
        <input
          {id}
          class="field {setting.kind === 'integer' ? 'max-w-32' : 'font-mono'}"
          type={setting.secret ? 'password' : setting.kind === 'integer' ? 'number' : 'text'}
          min={setting.kind === 'integer' ? 1 : undefined}
          inputmode={setting.kind === 'url' ? 'url' : undefined}
          autocomplete="off"
          spellcheck="false"
          disabled={draft.clear}
          {placeholder}
          value={draft.value}
          oninput={(event) => (form.drafts[setting.key].value = event.currentTarget.value)}
        />
      {/if}
      {#if setting.secret && setting.isSet}
        <Badge status={draft.clear ? 'warning' : 'good'}>{draft.clear ? 'will be cleared' : 'stored'}</Badge>
        <button
          type="button"
          class="btn-ghost px-2"
          onclick={() => (form.drafts[setting.key] = { value: '', clear: !draft.clear })}
          >{draft.clear ? 'Keep' : 'Clear'}</button
        >
      {/if}
    </div>
  </div>
{/if}
