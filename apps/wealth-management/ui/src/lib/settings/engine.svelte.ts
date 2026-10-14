import { invalidateAll } from '$app/navigation';
import { api } from '$lib/api/endpoints';
import type { EngineSettings } from '$lib/api/types';
import { toasts } from '$lib/state/toasts.svelte';
import { groupSettings, initialDrafts, pendingChanges, waitForRestart, type Drafts } from './form';

const message = (failure: unknown) => (failure instanceof Error ? failure.message : String(failure));

/**
 * The engine settings editor shared by every Settings tab, so one save covers
 * changes made in several tabs. Saved changes apply after a restart. Secrets
 * are write-only: the engine only says whether they are set.
 */
export class EngineSettingsForm {
  view = $state<EngineSettings | null>(null);
  drafts = $state<Drafts>({});
  loadError = $state('');
  saveError = $state('');
  saving = $state(false);
  restarting = $state(false);
  /** The AI endpoint's models; null until loaded. */
  models = $state<string[] | null>(null);
  modelsError = $state('');

  readonly groups = $derived(this.view ? groupSettings(this.view) : []);
  readonly changes = $derived(this.view ? pendingChanges(this.view.settings, this.drafts) : {});
  readonly changeCount = $derived(Object.keys(this.changes).length);

  private _show(next: EngineSettings) {
    this.view = next;
    this.drafts = initialDrafts(next.settings);
    this.saveError = '';
  }

  load = async () => {
    this.loadError = '';
    try {
      this._show(await api.settings());
    } catch (failure) {
      this.loadError = message(failure);
    }
  };

  loadModels = async () => {
    this.modelsError = '';
    try {
      this.models = (await api.aiModels()).models;
    } catch (failure) {
      this.models = [];
      this.modelsError = message(failure);
    }
  };

  /** Settings of the given groups, in the engine's order. */
  groupsIn(ids: readonly string[]) {
    return this.groups.filter(({ group }) => ids.includes(group.id));
  }

  /** The value a setting runs with as edited: the draft, else its default. */
  effectiveValue(key: string): string {
    const setting = this.view?.settings.find((candidate) => candidate.key === key);
    if (!setting || setting.secret) return '';
    return this.drafts[key]?.value.trim() || setting.default || '';
  }

  discard = () => {
    if (this.view) this._show(this.view);
  };

  save = async () => {
    if (this.changeCount === 0) return;
    this.saving = true;
    this.saveError = '';
    try {
      this._show(await api.updateSettings(this.changes));
      toasts.push('success', 'Settings saved. Restart the engine to apply them.');
    } catch (failure) {
      this.saveError = message(failure);
    } finally {
      this.saving = false;
    }
  };

  restart = async () => {
    this.restarting = true;
    try {
      await api.restartEngine();
      this._show(await waitForRestart(() => api.settings()));
      await invalidateAll();
      toasts.push('success', 'Engine restarted with the saved settings.');
    } catch (failure) {
      toasts.error(failure, 'Restart');
    } finally {
      this.restarting = false;
    }
  };
}
