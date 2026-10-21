export type ToastKind = 'error' | 'warning' | 'info' | 'success';

export interface Toast {
  id: number;
  kind: ToastKind;
  message: string;
}

const TOAST_MS = 5000;

class Toasts {
  items = $state<Toast[]>([]);
  #nextId = 1;

  push(kind: ToastKind, message: string): void {
    const id = this.#nextId++;
    this.items = [...this.items, { id, kind, message }];
    setTimeout(() => this.dismiss(id), TOAST_MS);
  }

  dismiss(id: number): void {
    this.items = this.items.filter((toast) => toast.id !== id);
  }

  /** Reports a failed action, prefixed with what was being done. */
  error(error: unknown, context?: string): void {
    const message = error instanceof Error ? error.message : String(error);
    this.push('error', context ? `${context}: ${message}` : message);
  }

  info(message: string): void {
    this.push('info', message);
  }

  success(message: string): void {
    this.push('success', message);
  }
}

export const toasts = new Toasts();
