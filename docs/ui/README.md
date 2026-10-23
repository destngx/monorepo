# UI style guide

The shared look and the rules for every web UI in this repo. It comes from three apps:

| App                                                 | Role in this guide                                                                         | Tokens and utilities                              |
| --------------------------------------------------- | ------------------------------------------------------------------------------------------ | ------------------------------------------------- |
| health-watcher (HW), `apps/health-watcher/ui`       | **Reference for new tool apps.** Newest token set, named by role.                          | `apps/health-watcher/ui/src/routes/layout.css`    |
| wealth-management (WM), `apps/wealth-management/ui` | Reference for loading states, skeletons and charts.                                        | `apps/wealth-management/ui/src/routes/layout.css` |
| portfolio-landpage, `apps/portfolio-landpage`       | Source of the family look (Aurora, halftone, ink and paper). Public site, so it is louder. | `apps/portfolio-landpage/src/app.css`             |

A new UI app reads this page first, then builds on [`libs/ui-kit`](../../libs/ui-kit/README.md), the shared theme and components, as section 12 lists. App docs (for example the WM "UI & Styling" section in `docs/wealth-management/AGENTS.md`) add rules for their own app. They do not override this page.

## 1. Stack

- SvelteKit 2 with Svelte 5 runes (runes mode is forced for app code), Tailwind CSS 4 through `@tailwindcss/vite`, TypeScript. Versions come from the root `package.json`; a UI has no `package.json` of its own.
- An app with a Go engine builds a client-only SPA: `ssr = false`, `adapter-static` with `fallback: 'index.html'` and `paths.base: '/ui'`. The engine serves it at `/ui`, and the dev server proxies `/api` to the engine.
- No third-party component library (no bits-ui, no shadcn) and no icon package. Shared components live in `libs/ui-kit`; an app's own components are small Svelte files in `src/lib/components`. Icons are inline 24×24 stroke paths in `Icon.svelte` (Lucide style, `stroke-width` 1.8).
- Fonts are bundled with `@fontsource` or `@fontsource-variable` and imported in the root layout. Do not name a font in `--font-*` that the app does not import; WM does this with Inter, and machines without Inter fall back to the system font.

## 2. The look

Neo-brutalism with quiet luxury: flat warm paper, near-black ink, hard offset shadows with no blur, an editorial serif for headings, one muted accent. Every app shares these rules:

- The page is flat paper (`bg-canvas`). Raised things sit on an opaque `bg-surface`, so text never sits on the background.
- Behind everything is `Aurora`, mounted once in the root layout: four blurred colour blobs that drift, under a halftone dot screen. The blobs stand still under reduced motion.
- Shadows are hard offsets of the shade colour (`2px`, `4px`, `6px`), never blurred.
- Corners are square or nearly square. The theme sets Tailwind's `rounded-lg`, `-xl` and `-2xl` to `0` (HW) or tightens the whole scale (WM).
- Headings (`h1` to `h3`, card and modal titles) use the display serif. Body text is sans. Numbers use tabular figures (`tabular`).
- Labels and eyebrows are small, uppercase and tracked (`0.08em` to `0.16em`).
- The primary button and the current tab are solid ink, inverted in dark mode. It is the highest contrast on the page.
- One accent hue marks links, the current page and quiet emphasis. Each app picks its own: WM bronze, HW olive, portfolio pink.
- Red (`danger`) is for errors and destructive actions only.
- Do not add gradients, glass, blur or soft shadows beyond Aurora.

How loud an app is depends on how strong its edges and shadows are:

| App       | Quiet surface                      | Featured surface (buttons, modals, upload zone) | Display font             |
| --------- | ---------------------------------- | ----------------------------------------------- | ------------------------ |
| portfolio | 3px ink border, 6px shadow         | the same, lifts and tilts on hover              | Dela Gothic One          |
| WM        | 1.5px edge, 4px shadow, 4px radius | the same                                        | Iowan Old Style (system) |
| HW        | 1px stone line, no shadow          | 2px ink edge, 4px shadow, square                | Playfair Display         |

A new tool app follows HW: quiet surfaces stay light, and the ink edge and shadow are kept for featured things and for a card under the pointer. Keep the portfolio's level of noise for public pages only.

## 3. Tokens

The shared tokens live in `@theme` in `libs/ui-kit/src/theme.css`. An app imports it from `src/routes/layout.css` and overrides only its accent. Name colours by role, never by hue, so a palette change touches one file.

| Token                                 | Role                                                                                       |
| ------------------------------------- | ------------------------------------------------------------------------------------------ |
| `ink-50` to `ink-950`                 | Warm neutrals. 50 is paper, 100 ivory, 200 stone, 900 ink.                                 |
| `accent-50` to `accent-950`           | The app's one accent: links, what is selected, quiet emphasis.                             |
| `good-*`                              | A good status, nothing else.                                                               |
| `danger-*`                            | Errors and destructive actions, nothing else.                                              |
| `warn-*`                              | Warnings and things that wait for a person.                                                |
| `info-*`                              | Work in progress and neutral notices.                                                      |
| `canvas`, `surface`, `rule`, `edge`   | Page, raised surface, thin line, ink edge. They point at CSS variables that `.dark` swaps. |
| `shadow-brut-sm`, `-brut`, `-brut-lg` | The 2px, 4px and 6px hard shadows in `--shade`.                                            |
| `--aurora-1` to `--aurora-4`, `--dot` | Aurora blobs and the halftone dot, set for light and dark.                                 |
| `--font-sans`, `--font-display`       | Body sans and heading serif.                                                               |

Rules:

- Use the role scales in components. Do not write Tailwind hue names (`rose-800`, `emerald-600`) or hex values in markup. WM's raw `stone-*`, `rose-*` and `sky-*` classes are older drift, not the pattern.
- Pick a new accent hue for a new app, at low chroma, and build all eleven steps. Check that `accent-700` text passes 4.5:1 on `surface` in light mode and `accent-300` in dark mode.
- In dark mode the edge becomes a soft stone and the shadow a deep brown, so blocks stay quiet. Swap `--canvas`, `--surface`, `--rule`, `--edge`, `--shade`, the Aurora colours and `--dot` under `.dark`, nothing else.
- Aurora colours are the accent, a neutral and a trace of danger mixed with `transparent` (HW), or the portfolio's teal, pink, blue and lime (WM). Keep them faint: 10 to 40 percent.

## 4. Base styles and utilities

`theme.css` in `libs/ui-kit` has the base layer (HW's, plus WM's scroller rule). It gives every page:

- pointer cursors on buttons, summaries, selects and checkbox labels; `not-allowed` on disabled; `progress` on `aria-busy`
- a 150ms ease-out transition on links and buttons, and a 1px press on buttons
- underlined bare links (`a:not([class])`) in the accent
- table body rows that highlight under the pointer
- a focus ring on `:focus-visible`
- reduced motion: transitions drop to 0.01ms and the press stops

It also makes every `overflow-x-auto` a containing block (`position: relative`), so a table's sr-only text cannot widen the page on phones.

Base rules use `:where()` so any utility on a component wins.

`theme.css` defines these utilities with `@utility`. An app that keeps its own look in its own `layout.css` (WM) defines them with the same names and meaning, and maps ui-kit's role names (`ink`, `danger`, `warn`, `info`, `good`, `shadow-brut*`) onto its palette. The AI usage gauge in `libs/ui-kit` uses `panel`, `muted`, `link`, `tabular` and `icon-btn`, so an app with its own `layout.css` needs at least those.

| Utility        | Meaning                                                                                                                        |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------ |
| `panel`        | A quiet raised surface.                                                                                                        |
| `panel-strong` | A featured surface: ink edge and hard shadow. Modals, login card, toasts, upload zone.                                         |
| `btn`          | Base button. Small hard shadow, lifts 1px on hover, presses flat into its shadow.                                              |
| `btn-primary`  | Solid ink. One per view.                                                                                                       |
| `btn-ghost`    | Quiet; the edge shows under the pointer.                                                                                       |
| `btn-danger`   | Solid danger, for destructive actions.                                                                                         |
| `field`        | Inputs, selects and textareas. Hover darkens the border; focus adds the edge and small shadow; `aria-invalid` turns it danger. |
| `label`        | Form label: small, uppercase, tracked.                                                                                         |
| `heading`      | Section heading in the display serif.                                                                                          |
| `eyebrow`      | Uppercase metadata above a heading, in the accent.                                                                             |
| `tag`          | Small square status label with a thin edge in `currentColor`.                                                                  |
| `muted`        | Secondary text.                                                                                                                |
| `link`         | A text link: accent underline that thickens on hover.                                                                          |
| `interactive`  | A card that is clickable as a whole: edge and shadow on hover, presses flat.                                                   |
| `row-link`     | A table row or list item that opens something.                                                                                 |
| `skeleton`     | A loading placeholder shape; pulses only with motion allowed.                                                                  |
| `tabular`      | Tabular figures.                                                                                                               |
| `icon-btn`     | A small icon-only button (`size-7`).                                                                                           |
| `hit-area`     | A larger invisible tap target on touch screens for controls drawn smaller than a fingertip.                                    |

Use Tailwind classes for everything else. Write plain CSS only for what Tailwind cannot express (keyframes, `::backdrop`, Aurora).

## 5. Theme

- Dark mode is a `.dark` class on `<html>` plus `style.colorScheme`, declared with `@custom-variant dark (&:where(.dark, .dark *));`.
- The setting is `system`, `light` or `dark` in `localStorage` under `<app>.theme` (`wm.theme`, `hw.theme`). `system` is the default and follows `prefers-color-scheme`. Listen to the media query so a later system change applies at once, as WM's root layout does; HW does not yet.
- An inline script in `app.html` applies the theme before first paint, so the page never flashes.
- The portfolio uses `data-theme` and stores only `light` or `dark`. It is a static public site; do not copy that into a tool app.

## 6. Layout

- The shell is a sticky `panel` header (`sticky top-3 z-30`) over a centred `main` (`max-w-6xl`, or `max-w-7xl` with a sidebar), padded `px-3 sm:px-6`.
- Navigation: up to five or so destinations go in the header on desktop and in a bottom tab bar on phones (HW). More than that gets a collapsible sidebar on desktop, a sheet on phones and area tabs (WM).
- The phone tab bar is `fixed inset-x-2 bottom-2` with `padding-bottom: max(0.375rem, env(safe-area-inset-bottom))`, hidden from `lg`. Give `main` enough bottom padding (`pb-24` to `pb-28`, `lg:pb-8`) to clear it.
- Every page fits a 390px screen with no sideways scrolling:
  - A grid with breakpoint columns starts at `grid-cols-1`.
  - A flex child that holds text gets `min-w-0`.
  - A wide table scrolls inside its card with `overflow-x-auto`, or stacks its rows into cards below `sm` (WM `TransactionTable`).
- A page starts with a page title component (`PageHeader` in WM, `PageTitle` in HW): the title, an optional subtitle and an `actions` snippet. It also sets `<title>Page · App</title>`.
- Cards are `Card` (`panel p-5`) with an optional title, subtitle and actions. `foldable` adds a chevron before the title that folds the body (bind `open`; it hides the actions while folded), for setup or tools a page rarely needs. Put spacing on the wrapper, not inside the card.
- Pages that answer a question lead with one headline answer (WM `PageHeadline`), then the detail.

## 7. Components and behaviour

These components are in `libs/ui-kit`; import them from `@libs/ui-kit`. HW and WM both use them. WM keeps its own `Badge`, `PageHeader`, `ThemeToggle` and `status.ts`, which use its own hues and settings store.

| Component         | Behaviour                                                                                                                                                                                                                                                                                                                                                                                                                |
| ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `Button`          | `variant` `primary`, `ghost`, `danger`; `size` `md`, `sm`, `icon`. An `onclick` that returns a promise makes it busy (spinner, `aria-busy`, repeat clicks ignored), then toasts `success` or the error with an `errorContext` prefix. Icon-only buttons need `label`, which is the aria-label and the tooltip.                                                                                                           |
| `action()`        | `state/action.svelte.ts`: the same busy and toast handling for actions outside a button.                                                                                                                                                                                                                                                                                                                                 |
| `Modal`           | A native `<dialog>` opened with `showModal()`, so focus is trapped and returns to the opener. `title`, `footer`, `wide`, `dismissible`, `busy`. Escape and backdrop clicks close it unless it is busy or not dismissible; a backdrop click counts only if the press also started on the backdrop. Focus goes to the first field or `[data-autofocus]`, except on touch screens, where the keyboard would cover the form. |
| `Toasts`          | `error`, `warning`, `info`, `success`, bottom right, above the phone tab bar. Errors are `role="alert"`, others `role="status"`, in an `aria-live="polite"` region.                                                                                                                                                                                                                                                      |
| `EmptyState`      | `title` and an optional `detail`, centred.                                                                                                                                                                                                                                                                                                                                                                               |
| `Icon`, `Spinner` | `Icon` is `aria-hidden`; a button carries the label.                                                                                                                                                                                                                                                                                                                                                                     |
| `Aurora`          | Mount it once; the app sets its colours with `--aurora-1` to `--aurora-4` and `--dot`.                                                                                                                                                                                                                                                                                                                                   |
| `MenuButton` (HW) | `aria-haspopup`, `aria-expanded`, `role="menu"`; arrow keys move, Escape and an outside click close.                                                                                                                                                                                                                                                                                                                     |
| `AIUsageGauge`    | The AI quota gauge: weekly window on the outer ring, 5-hour window on the inner one. HW and WM already import it from `@libs/ui-kit`, with `@source` for `libs/ui-kit/src/ai-usage` in their `layout.css`.                                                                                                                                                                                                               |

### Loading

Use WM's pattern, described in full in `docs/wealth-management/AGENTS.md` under "Loading and feedback". In short:

- A page never waits for the API. The load function returns `defer(() => api.x({ fetch }))`, which never rejects. The page renders its shell at once.
- Each slow section is a `<Deferred data={...} label="...">` with its own skeleton, then its data, or its own error with Retry. Failures are classified (`auth`, `not-configured`, `unavailable`, `unreachable`, `other`).
- A refresh keeps the shown value with a small Refreshing badge. Skeletons show only before the first value.
- Skeletons are `role="status"` and `aria-busy`, with "Loading {label}…" for screen readers.
- Page changes show `NavProgress`: nothing under 150ms, then a top bar.

All of it is in `libs/ui-kit`. An app whose API errors need more than the default rules makes its own `defer` with `createLoader({ classify, onFailure })`; the failure's `hint` is the advice `LoadError` shows.

### Status and charts

- A status is never shown by colour alone. Pair it with an icon and a label, and give the label to screen readers (`sr-only` "Good:").
- Use the one status scale, `STATUS` in `libs/ui-kit` (`good`, `warning`, `serious`, `critical`, `info`, `neutral`, each with icon, label and classes). It is built on the `good`, `warn`, `danger`, `info` and `ink` tokens.
- Charts are hand-written SVG. `libs/ui-kit` has the geometry (`extent`, `scale`, `ticks`, `toDay`, `dateTicks`, `linePoints`, `linePath`, `areaPath`, `arcPath`, `slices`, `radarPoints`), `leavesReadout` for hover readouts that stay open after a tap, and the small charts `Sparkline`, `Donut`, `Ring` and `Radar`. A chart tied to one app's data or formatting (an axis that masks amounts, a metric's reference bands) stays in that app and is built on the same geometry.
- Series colours come from `PALETTE` in `libs/ui-kit` in a fixed order, with `PRIMARY` for a chart's one series and `NEUTRAL` for context such as a cap. The palette passes the dataviz skill's validator on both apps' light and dark surfaces. An app's own colour sets (income and expense, flagged results) go in its `lib/charts/palette.ts`, each checked as a set on that app's surfaces with the same validator. Do not write a hex colour in a chart. Tell estimated or secondary series apart by dash and marker shape as well as colour.

## 8. Motion

- Interactions take 150ms ease-out. Hover lifts by 1px to 2px and grows the shadow; a press sinks into the shadow.
- Use `motion-safe:` and `motion-reduce:` variants. Under reduced motion, nothing lifts, pulses or drifts; spinners slow down.
- Tool apps stop there. View transitions, scroll reveal (`use:reveal`), the splash and the ink cursor belong to the portfolio.

## 9. Writing in the UI

- The UI is in English.
- Use sentence case everywhere: page titles, nav, buttons, card titles ("Engine settings", "New account", "Add a person"). WM's Title Case page names ("Cash Flow") are drift.
- Buttons are a verb and an object. Busy text is the verb's gerund with a real ellipsis: "Saving…", "Syncing…". Loading text is "Loading {thing}…".
- Error toasts are "Could not {action}" followed by the server's message: `toasts.error(error, 'Could not save the draft')`. A failed section reads "Could not load {label}".
- Success toasts are short full sentences with a period: "Prices refreshed.", "Synced with Google Sheets."
- An empty state says what is missing, then how to add it: "No accounts yet" / "Create one with New account."
- Subtitles are one sentence.
- Dates are `dd/MM/yyyy` (`dd/MM/yyyy HH:mm` with a time), taken from the calendar part of the string so the time zone cannot shift them. "Today" is in `Asia/Ho_Chi_Minh`. VND is `1.234.567 ₫` through `formatVND`. A missing value shows `-`. Formatters live in `lib/format.ts`, with tests.
- Never put real names, amounts or accounts in fixtures, tests or screenshots. Use obvious placeholders.

## 10. Accessibility

- Labelled nav landmarks (`aria-label="Main"`), `aria-current="page"` on the current link.
- A visible focus ring on every control (`focus-visible:outline-2 outline-offset-2`).
- Icon-only controls have an `aria-label`; decorative elements (Aurora, icons) are `aria-hidden`.
- Busy controls and sections set `aria-busy`. Loading regions are `role="status"`, errors `role="alert"`.
- Disclosures set `aria-expanded` and `aria-controls` (HW `FoldToggle`).
- Text contrast meets 4.5:1 in both themes.
- A control drawn smaller than a fingertip, such as an `icon-btn`, gets `hit-area`.

## 11. Tests and tooling

- Vitest 4 with jsdom and `@testing-library/svelte` (`svelteTesting()`), `include: ['src/**/*.test.ts']`. Test pure helpers (`format.ts`, `status.ts`) and components with behaviour (`Button`, `Modal`, `Deferred`, `MenuButton`), test-first.
- Nx targets: `test-ui` (`pnpm exec vitest run`), `check-ui` (`svelte-kit sync && svelte-check`), `lint-ui` (`pnpm exec eslint --concurrency auto .`), `build-ui`.
- Prettier from the root `.prettierrc` (single quotes, `printWidth` 120) with `prettier-plugin-svelte`. Tailwind classes are not sorted automatically.

## 12. Starting a new UI app

1. Create the SvelteKit app under `apps/<app>/ui` with the same `svelte.config.js` and `vite.config.mjs` shape as HW.
2. Wire up `libs/ui-kit` as its README describes: the alias, the `theme.css` import, the fonts, `createTheme('<app>.theme')` with the first-paint script, and `implicitDependencies`.
3. Pick a new accent hue, set all eleven `accent-*` steps in the app's `@theme`, and adjust the Aurora mix. Keep the token and utility names.
4. Mount `Aurora`, `Toasts` and `NavProgress` in the root layout, and use `defer` with `<Deferred>` for every slow section.
5. Build the shell from section 6 and check every page at 390px in both themes.
6. Put app-only formatters (money, labels) in the app's `lib/format.ts`, with tests, next to the shared ones from the library.
7. Add the app's own UI rules to its docs and link back here. If a component would help another app, add it to the library instead.
