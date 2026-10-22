/**
 * Which pages of a page list are folded, so a long document scrolls less.
 * Pages start folded unless startOpen says otherwise, for every page or for
 * the ones it returns true for.
 */
export class Folds {
  #folded = $state<Record<string, boolean>>({});
  readonly #startOpen: (id: string) => boolean;

  constructor(options: { startOpen?: boolean | ((id: string) => boolean) } = {}) {
    const start = options.startOpen ?? false;
    this.#startOpen = typeof start === 'function' ? start : () => start;
  }

  #isFolded(id: string): boolean {
    return this.#folded[id] ?? !this.#startOpen(id);
  }

  isOpen(id: string): boolean {
    return !this.#isFolded(id);
  }

  toggle(id: string): void {
    this.#folded[id] = !this.#isFolded(id);
  }

  open(id: string): void {
    this.#folded[id] = false;
  }

  /** True when every page in ids is folded. */
  allFolded(ids: string[]): boolean {
    return ids.length > 0 && ids.every((id) => this.#isFolded(id));
  }

  /** Folds every page in ids, or opens them all when they are all folded. */
  toggleAll(ids: string[]): void {
    const shouldFold = !this.allFolded(ids);
    for (const id of ids) this.#folded[id] = shouldFold;
  }
}
