import type { NewsReport } from '$lib/api/types';
import { NEWS_STORAGE_KEY, parseStoredReports, removeReport, upsertReport } from './news';

function load(): NewsReport[] {
  try {
    return parseStoredReports(globalThis.sessionStorage?.getItem(NEWS_STORAGE_KEY) ?? null);
  } catch {
    return [];
  }
}

function save(reports: NewsReport[]): void {
  try {
    if (reports.length === 0) globalThis.sessionStorage?.removeItem(NEWS_STORAGE_KEY);
    else globalThis.sessionStorage?.setItem(NEWS_STORAGE_KEY, JSON.stringify(reports));
  } catch {
    // Storage can be unavailable (private mode); reports then last until reload.
  }
}

/** Scanned news reports of this tab, shared by the News Scanner and the Think Tank. */
class NewsStore {
  reports = $state.raw<NewsReport[]>(load());

  upsert(report: NewsReport): void {
    this.reports = upsertReport(this.reports, report);
    save(this.reports);
  }

  remove(topic: string): void {
    this.reports = removeReport(this.reports, topic);
    save(this.reports);
  }

  clear(): void {
    this.reports = [];
    save(this.reports);
  }
}

export const newsStore = new NewsStore();
