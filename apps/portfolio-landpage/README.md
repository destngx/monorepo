# portfolio-landpage

Personal site of Quang Dinh Nguyen Pham, styled as a manga volume. SvelteKit 2 (Svelte 5), Tailwind v4, mdsvex, and a
lazy-loaded Threlte scene for the hero plane. Every route is prerendered to static HTML.

## Commands

Run from the repo root:

```sh
pnpm exec nx serve portfolio-landpage     # dev server on http://localhost:8083
pnpm exec nx test portfolio-landpage      # vitest
pnpm exec nx check portfolio-landpage     # svelte-check
pnpm exec nx lint portfolio-landpage      # eslint
pnpm exec nx build portfolio-landpage     # static output in apps/portfolio-landpage/build
pnpm exec nx preview portfolio-landpage   # build, then serve the output
```

## Layout

```
src/
  content/projects/*.md   case studies (file name = URL slug)
  content/posts/*.md      blog posts (file name = URL slug)
  lib/content/            frontmatter schemas (Zod) and loaders
  lib/data/               profile (roles, milestones, skills, hobbies) and site config
  lib/three/              hero plane scene and toon shading helpers
  lib/ui/                 shared components
  routes/                 pages, rss.xml and sitemap.xml
static/                   images, og.png, models/plane.glb
```

## Writing content

Add a Markdown file to `src/content/posts/` or `src/content/projects/`. The frontmatter reference, drafts and
conventions are in [docs/portfolio-landpage/writing-content.md](../../docs/portfolio-landpage/writing-content.md).

## Design notes

- Colours, borders and shadows are CSS variables in `src/app.css`. Dark mode switches them via `data-theme` on
  `<html>`, defaulting to the system preference and persisting the choice in `localStorage`.
- Motion respects `prefers-reduced-motion`: the intro splash, reveals and the 3D plane are all skipped.
- The plane is a progressive enhancement. It loads only after mount, only with WebGL2, and the page is complete
  without it.

## Deploy

`vercel.json` holds the build settings, so the Vercel project only needs its **Root Directory** set to
`apps/portfolio-landpage`. Vercel installs the pnpm workspace, runs `vite build`, and serves `build/` with clean URLs
(`/projects/ai-gateway` rather than `.html`). Unknown paths get `404.html`, which renders the app's own 404 page.

### Changing the domain

The production origin lives in one place, `site.url` in `src/lib/data/site.ts`. It feeds canonical URLs, Open Graph
tags, RSS and the sitemap. To move to a new domain:

1. In Vercel, open the project, then **Settings > Domains**, and add the new domain (for example
   `destngx.vercel.app`; `*.vercel.app` names are first come, first served).
2. Edit the old domain there and set it to redirect (308) to the new one, so existing links and search results follow.
3. Update `site.url`, commit, and let it deploy.
