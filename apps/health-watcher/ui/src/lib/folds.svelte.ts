/**
 * Which pages of a page list are folded, so a long document scrolls less.
 * Pages start open.
 */
export class Folds {
  #folded = $state<Record<string, boolean>>({});

  isOpen(id: string): boolean {
    return !this.#folded[id];
  }

  toggle(id: string): void {
    this.#folded[id] = !this.#folded[id];
  }

  open(id: string): void {
    this.#folded[id] = false;
  }

  /** True when every page in ids is folded. */
  allFolded(ids: string[]): boolean {
    return ids.length > 0 && ids.every((id) => this.#folded[id]);
  }

  /** Folds every page in ids, or opens them all when they are all folded. */
  toggleAll(ids: string[]): void {
    const shouldFold = !this.allFolded(ids);
    for (const id of ids) this.#folded[id] = shouldFold;
  }
}
