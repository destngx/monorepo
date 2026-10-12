// See https://svelte.dev/docs/kit/types#app.d.ts
declare global {
  namespace App {
    interface Error {
      message: string;
      status?: number;
      // Set when the failure is a Google Sheets credential problem (WM-UI-4).
      authProblem?: boolean;
    }
    // Loaded by the root layout for every page.
    interface PageData {
      health: import('$lib/api/types').EngineHealth | null;
    }
  }
}

export {};
