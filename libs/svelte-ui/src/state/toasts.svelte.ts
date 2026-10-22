export type ToastKind = 'error' | 'warning' | 'info' | 'success';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

/**
 * Turns a failed action into the toast to show, or null when the app shows it
 * another way (wealth-management raises its Google Sheets re-auth alert).
 * `text` is the message with the context prefix already added.
 */
export type ErrorToast = (error: unknown, text: string) => { kind: ToastKind; message: string } | null;

export const TOAST_MS = 5000;

const plainError: ErrorToast = (_error, text) => ({ kind: 'error', message: text });

class Toasts {
  items = $state<Toast[]>([]);
  #nextId = 1;
  #errorToast: ErrorToast = plainError;

  push(kind: ToastKind, message: string): void {
    const id = this.#nextId++;
    this.items = [...this.items, { id, kind, message }];
    setTimeout(() => this.dismiss(id), TOAST_MS);
  }

  dismiss(id: number): void {
    this.items = this.items.filter((toast) => toast.id !== id);
  }

  /** Reports a failed action, prefixed with what was being done: "Could not save: disk full". */
  error(error: unknown, context?: string): void {
    const message = error instanceof Error ? error.message : String(error);
    const toast = this.#errorToast(error, context ? `${context}: ${message}` : message);
    if (toast) this.push(toast.kind, toast.message);
  }

  warning(message: string): void {
    this.push('warning', message);
  }

  info(message: string): void {
    this.push('info', message);
  }

  /** Confirms an action that finished: "Saved.", "Synced with Google Sheets." */
  success(message: string): void {
    this.push('success', message);
  }

  /** Lets the app decide how its own errors show; call it once at start-up. Null restores the default. */
  setErrorToast(errorToast: ErrorToast | null): void {
    this.#errorToast = errorToast ?? plainError;
  }
}

export const toasts = new Toasts();
