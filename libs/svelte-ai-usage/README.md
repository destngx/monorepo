# svelte-ai-usage

The menu bar gauge of the AI quota that ai-gateway reports, shared by wealth-management and health-watcher. The outer ring shows the weekly window and the inner ring the 5-hour window. A tap opens the details.

## Using it in an app

1. Add the alias in `svelte.config.js`:
   ```js
   kit: { alias: { '@libs/svelte-ai-usage': '../../../libs/svelte-ai-usage/src/index.ts' } }
   ```
2. Let Tailwind scan the library from the app's CSS entry:
   ```css
   @source '../../../../../libs/svelte-ai-usage/src';
   ```
3. Define the `panel`, `muted`, `link`, `tabular` and `icon-btn` utilities. The gauge uses them for its look.
4. Add `"implicitDependencies": ["svelte-ai-usage"]` to the app's `project.json`, so Nx reruns the app's UI targets when this library changes.
5. Render it with the app's quota state and theme:
   ```svelte
   <AIUsageGauge
     source={aiUsage}
     leftColor={leftColorScale({ lightness: 0.7, chroma: 0.17, emptyHue: 25, fullHue: 150 })}
     formatReset={formatDateTime}
     errorClass="text-rose-600 dark:text-rose-400"
     settingsHref="{base}/settings"
   />
   ```

`source` is any object with `usage`, `error`, `loadedAt` and `load()`, plus an optional `enabled`. The gauge loads it on mount and every 5 minutes while the tab is visible.

## Targets

`pnpm exec nx run svelte-ai-usage:test`, `:check` and `:lint`.
