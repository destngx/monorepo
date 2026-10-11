# Writing content for portfolio-landpage

Projects and blog posts are Markdown files in `apps/portfolio-landpage/src/content/`. There is no CMS: add a file,
commit it, and the next deploy publishes it.

## How posts work

- Each post is a Markdown file in `src/content/posts/`. The file name becomes the URL slug, so
  `hello-world.md` is served at `/blog/hello-world`.
- Frontmatter is validated with Zod (`src/lib/content/schema.ts`) at build time, so a typo fails the build instead of
  shipping a broken page.
- Set `draft: true` to keep a post out of production builds, the RSS feed and the sitemap. Drafts still show in
  `nx serve`, so you can preview them.
- Posts are listed newest first, by `date`.

The loader in `src/lib/server/content.ts` is the only place that decides what gets published:

```ts
export const getPosts = (): Post[] =>
  publishedPosts(parseCollection(postModules, postMetaSchema), { includeDrafts: dev });
```

### Post frontmatter

```yaml
title: Rebuilding this site from scratch
description: At most 240 characters. Used for the list, RSS and social cards.
date: 2026-10-10
updated: 2026-10-12 # optional
tags: [svelte, design]
draft: true # optional, defaults to false
```

## How projects work

Case studies live in `src/content/projects/`, one file per project, and are served at `/projects/<file-name>`.
`featured: true` puts a project on the home page; `order` sorts it (lower comes first).

### Project frontmatter

```yaml
title: AI Gateway
summary: One or two sentences, at most 240 characters.
year: 2026
status: active # active | shipped | experiment | archived
stack: [Go, net/http]
repo: https://github.com/... # optional
live: https://... # optional
featured: true # optional, defaults to false
order: 2 # optional, defaults to 99
sfx: シュッ # optional sound effect stamped on the card, at most 6 characters
```

Case studies use the same section headings so they read like one series: The problem, What it does, How it's built,
Interesting bits, What's next.

## Markdown features

- Fenced code blocks are highlighted with Shiki at build time. Always set the language (` ```ts `, ` ```go `).
- `##` headings render as manga panel titles; keep `#` for the page title, which comes from frontmatter.
- Images go in `apps/portfolio-landpage/static/images/` and are referenced as `/images/<name>`.
