# svelte-ui

The shared theme, components and loading helpers of the web UIs, built to the rules in [docs/ui/README.md](../../docs/ui/README.md). It started from health-watcher's look, with wealth-management's loading stack and modal behaviour. No app uses it yet; health-watcher and wealth-management still have their own copies.

## What is in it

| Part        | Contents                                                                                                                                                                          |
| ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `theme.css` | Tokens (`ink`, `accent`, `good`, `warn`, `danger`, `info`, surfaces, shadows, fonts), the base layer, and every shared utility (`panel`, `btn-*`, `field`, `tag`, `hit-area`, …). |
| Components  | `Aurora`, `Badge`, `Button`, `Card`, `CopyField`, `EmptyState`, `Field`, `FoldToggle`, `Icon`, `MenuButton`, `Modal`, `PageTitle`, `Spinner`, `ThemeToggle`, `Toasts`.            |
| Loading     | `defer`, `createLoader`, `classifyFailure`, `settle`, `settled`, `track`; `Deferred`, `LoadError`, `NavProgress`, `Loading` and the `Skeleton*` shapes.                           |
| State       | `toasts`, `action()`, `createTheme(key)`, `applyTheme`, `themeScript(key)`.                                                                                                       |
| Helpers     | `STATUS` and `usageTone` (`status.ts`); `formatDate`, `formatDateTime`, `todayISO`, `formatSize`, `MISSING` (`format.ts`).                                                        |

## Using it in an app

The paths below are for a UI at `apps/<app>/ui`.

1. Add the alias in `svelte.config.js`:
   ```js
   kit: { alias: { '@libs/svelte-ui': '../../../libs/svelte-ui/src/index.ts' } }
   ```
2. Import the theme in `src/routes/layout.css`, right after Tailwind, and give the app its own accent. The theme's `@source` already lets Tailwind scan this library.
   ```css
   @import 'tailwindcss';
   @import '../../../../../libs/svelte-ui/src/theme.css';

   @theme {
     --color-accent-50: oklch(0.97 0.012 250);
     /* … every step to 950 */
   }
   ```
   Change the Aurora mix the same way, by setting `--aurora-1` to `--aurora-4` under `:root` and `.dark` in the app's base layer.
3. Import the fonts in the root `+layout.svelte`: `@fontsource-variable/inter` and `@fontsource-variable/playfair-display`.
4. Make the theme setting once, in `src/lib/state/theme.svelte.ts`:
   ```ts
   import { createTheme } from '@libs/svelte-ui';
   export const theme = createTheme('<app>.theme');
   ```
   Call `theme.watchSystem()` in the root layout's `onMount` and return its result. Put the output of `themeScript('<app>.theme')` in a `<script>` in the `<head>` of `app.html`, so the page never flashes light.
5. Mount `<Aurora />`, `<Toasts />` and `<NavProgress active={!!navigating.to} />` once in the root layout.
6. Make the app's loader in `src/lib/api/load.ts` when its API errors need more than the default rules, and use its `defer` in every load function:
   ```ts
   import { createLoader } from '@libs/svelte-ui';
   export const { defer, classify } = createLoader({ classify: myClassify, onFailure: raiseAlert });
   ```
   Pass the same `classify` to `<Deferred>` if a page hands it raw promises. The default classifier reads `status` and `code` from any `Error`, which matches the engines' `ApiError`.
7. If the app shows some errors in its own way (wealth-management's Google Sheets re-auth alert), call `toasts.setErrorToast(...)` once at start-up.
8. Add `"implicitDependencies": ["svelte-ui"]` to the app's `project.json`, so Nx reruns the app's UI targets when this library changes.

`Deferred`'s default Retry calls `invalidateAll` from `$app/navigation`, so the library works only inside a SvelteKit app. Its tests use a stub (`src/test/app-navigation.ts`).

## Targets

`pnpm exec nx run svelte-ui:test`, `:check` and `:lint`. `theme.test.ts` compiles `theme.css` with Tailwind and checks that every utility exists and that an app's accent overrides the default.
