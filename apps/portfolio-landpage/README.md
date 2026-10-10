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
  routes/                 pages, rss.xml, sitemap.xml and robots.txt
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

All build settings live in `vercel.json`, which Vercel reads from the project's Root Directory:

| Step          | Command                                                                                                                                                           |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Ignored build | `cd ../.. && npx -y nx-ignore@latest portfolio-landpage` (skips deploys when Nx says the app is unaffected)                                                       |
| Install       | `cd ../.. && pnpm install --frozen-lockfile --ignore-scripts` (the whole workspace, from the repo root; skips lifecycle scripts such as the husky `prepare` hook) |
| Build         | `pnpm --ignore-scripts exec vite build`                                                                                                                           |
| Output        | `build/`, served with clean URLs (`/projects/ai-gateway`, not `.html`) and `404.html` for unknown paths                                                           |

Vercel picks the pnpm version from `pnpm-lock.yaml`, so the root `package.json` has no `packageManager` pin. Vercel project settings:

- **Build and Deployment > Root Directory**: `apps/portfolio-landpage`, with "Include files outside the root
  directory" enabled (the workspace lockfile and `node_modules` live at the repo root).
- **Build and Deployment > Framework Settings**: preset **Other**, every override switched off, so `vercel.json`
  is the single source of truth. Same for **Ignored Build Step**: behaviour **Automatic**.
- **Build and Deployment > Node.js Version**: 24.x (Vite 8 needs Node 20.19+ or 22.12+).
- **Git > Git LFS**: enabled. Images and `og.png` are LFS-tracked by `.gitattributes`.

### Changing the domain

The production origin lives in one place, `site.url` in `src/lib/data/site.ts`. It feeds canonical URLs, Open Graph
tags, RSS, the sitemap and robots.txt. To move to a new domain:

1. In Vercel, open the project, then **Settings > Domains**, and add the new domain (for example
   `<name>.vercel.app`; `*.vercel.app` names are first come, first served).
2. Edit the old domain there and set it to redirect (308) to the new one, so existing links and search results follow.
3. Update `site.url`, commit, and let it deploy.
