import { ApiError } from '$lib/api/client';

export type ToastKind = 'error' | 'warning' | 'info' | 'success';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

const TOAST_MS = 5000;

class Toasts {
  items = $state<Toast[]>([]);
  // Set when Google Sheets credentials fail, to show the re-auth alert (WM-UI-4).
  sheetsAuthProblem = $state<string | null>(null);
  #nextId = 1;

  push(kind: ToastKind, message: string): void {
    const id = this.#nextId++;
    this.items = [...this.items, { id, kind, message }];
    setTimeout(() => this.dismiss(id), TOAST_MS);
  }

  dismiss(id: number): void {
    this.items = this.items.filter((toast) => toast.id !== id);
  }

  /** Reports a failed action; credential failures raise the re-auth alert. */
  error(error: unknown, context?: string): void {
    const message = error instanceof Error ? error.message : String(error);
    if (error instanceof ApiError && error.authProblem) {
      this.sheetsAuthProblem = message;
      return;
    }
    this.push('error', context ? `${context}: ${message}` : message);
  }
}

export const toasts = new Toasts();
