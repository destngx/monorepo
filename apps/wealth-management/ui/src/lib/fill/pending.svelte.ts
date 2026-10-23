import type { FillChatForm, FillFormKey } from '$lib/api/types';
import { untrack } from 'svelte';

// A chat fill waiting for its form (WM-FILL-7): Open the form stashes it
// and goes to the form's page; the page opens the form and its Describe it
// box applies the fill, which the user then checks and saves.

/** A stashed fill is dropped after this long, so it never surprises later. */
const PENDING_MS = 60_000;

class PendingFills {
  #pending = $state<{ fill: FillChatForm; at: number } | null>(null);

  set(fill: FillChatForm): void {
    this.#pending = { fill, at: Date.now() };
  }

  /** Whether a fill waits for the form. */
  waiting(form: FillFormKey): boolean {
    const pending = this.#pending;
    return pending !== null && pending.fill.form === form && Date.now() - pending.at < PENDING_MS;
  }

  /** The fill for the form, once. */
  take(form: FillFormKey): FillChatForm | null {
    if (!this.waiting(form)) return null;
    const { fill } = this.#pending!;
    this.#pending = null;
    return fill;
  }
}

export const pendingFills = new PendingFills();

/**
 * Opens a page's form when a chat fill waits for it. Call during component
 * setup; `open` runs untracked.
 */
export function openOnPendingFill(form: FillFormKey, open: () => void): void {
  $effect(() => {
    if (pendingFills.waiting(form)) untrack(open);
  });
}
