<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import { AI_USE_CASES, type AIStatus, type AIUseCase } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import { aiModels } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { effectiveModel, type ModelSource } from './models';

  // AI provider status and per-use-case model overrides (WM-AI-4, WM-AI-15).
  const useCaseLabels: Record<AIUseCase, string> = {
    insight: 'Reviews and insights',
    suggest: 'Category suggestion',
    vision: 'Smart capture',
    chat: 'Chat',
  };

  const sourceLabels: Record<ModelSource, string> = {
    browser: 'this browser',
    engine: 'engine',
    fallback: 'engine fallback',
    none: 'not set',
  };

  let status = $state<AIStatus | null>(null);
  let error = $state('');

  async function load() {
    error = '';
    try {
      status = await api.aiStatus();
    } catch (failure) {
      status = null;
      error = failure instanceof Error ? failure.message : String(failure);
    }
  }

  $effect(() => {
    void load();
  });
</script>

<Card title="AI" subtitle="OpenAI-compatible provider configured on the engine">
  {#snippet actions()}
    <button type="button" class="btn-ghost text-xs" onclick={load}>Refresh</button>
  {/snippet}
  {#if status}
    <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
      <dt class="muted">Status</dt>
      <dd>
        <Badge tone={status.configured ? 'green' : 'neutral'}
          >{status.configured ? 'configured' : 'not configured'}</Badge
        >
      </dd>
      {#if status.provider}
        <dt class="muted">Provider</dt>
        <dd class="font-mono text-xs">{status.provider}</dd>
      {/if}
      {#if status.endpoint}
        <dt class="muted">Endpoint</dt>
        <dd class="font-mono text-xs break-all">{status.endpoint}</dd>
      {/if}
      {#if status.fallbackModel}
        <dt class="muted">Fallback model</dt>
        <dd class="font-mono text-xs">{status.fallbackModel}</dd>
      {/if}
      <dt class="muted">Prompts</dt>
      <dd>
        {#if status.promptsError}
          <span class="text-rose-600 dark:text-rose-400">{status.promptsError}</span>
        {:else}
          {status.prompts} loaded{#if status.missingPrompts.length}, <span class="text-amber-700 dark:text-amber-300"
              >{status.missingPrompts.length} missing</span
            >{/if}
        {/if}
      </dd>
    </dl>
    <table class="mt-3 w-full text-sm">
      <caption class="label text-left">Model in use per use case</caption>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each AI_USE_CASES as useCase (useCase)}
          {@const current = effectiveModel(useCase, status, aiModels[useCase].value)}
          <tr>
            <td class="py-1.5 pr-3">{useCaseLabels[useCase]}</td>
            <td class="py-1.5 pr-3 font-mono text-xs break-all">{current.model || '-'}</td>
            <td class="muted py-1.5 text-right text-xs">{sourceLabels[current.source]}</td>
          </tr>
        {/each}
      </tbody>
    </table>
    {#if status.missingPrompts.length}
      <details class="mt-2 text-xs">
        <summary class="cursor-pointer text-amber-700 dark:text-amber-300"
          >Missing prompt keys (add them to the Prompts sheet)</summary
        >
        <ul class="mt-1 font-mono">
          {#each status.missingPrompts as key (key)}<li>{key}</li>{/each}
        </ul>
      </details>
    {/if}
  {:else if error}
    <p class="text-sm text-rose-600 dark:text-rose-400">AI status unavailable: {error}</p>
  {:else}
    <p class="muted text-sm">Loading…</p>
  {/if}

  <fieldset class="mt-4 space-y-2">
    <legend class="label">Model per use case</legend>
    {#each AI_USE_CASES as useCase (useCase)}
      <div class="grid grid-cols-[10rem_1fr] items-center gap-2 text-sm">
        <label for="ai-model-{useCase}">{useCaseLabels[useCase]}</label>
        <input
          id="ai-model-{useCase}"
          class="field font-mono text-xs"
          autocomplete="off"
          spellcheck="false"
          placeholder={status?.models[useCase] || 'engine default'}
          value={aiModels[useCase].value}
          onchange={(event) => {
            aiModels[useCase].value = event.currentTarget.value.trim();
            toasts.info(`${useCaseLabels[useCase]}: ${aiModels[useCase].value || 'engine default'} saved.`);
          }}
        />
      </div>
    {/each}
    <p class="muted text-xs">Overrides are stored in this browser. Leave empty to use the engine's model.</p>
  </fieldset>
  <p class="muted mt-3 text-xs">
    The engine has one adapter, for the OpenAI chat completions API, by design. Any OpenAI-compatible provider works
    (OpenAI, OpenRouter, Ollama, LM Studio, vLLM and others): set <code>AI_BASE_URL</code> and <code>AI_API_KEY</code>
    on the engine, and its model names in <code>AI_MODEL_*</code> or here.
  </p>
</Card>
