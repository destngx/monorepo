type ToastSeverity = 'error' | 'warning' | 'info';

interface ToastOptions {
  message: string;
  severity: ToastSeverity;
}

type ToastListener = (options: ToastOptions) => void;

class ToastEmitter {
  private listeners: Set<ToastListener> = new Set();

  subscribe(listener: ToastListener) {
    this.listeners.add(listener);
    return () => this.listeners.delete(listener);
  }

  notify(options: ToastOptions) {
    this.listeners.forEach((listener) => listener(options));
  }

  error(message: string) {
    this.notify({ message, severity: 'error' });
  }

  warning(message: string) {
    this.notify({ message, severity: 'warning' });
  }

  info(message: string) {
    this.notify({ message, severity: 'info' });
  }
}

export const toast = new ToastEmitter();
