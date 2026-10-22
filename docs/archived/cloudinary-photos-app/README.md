# Cloudinary Photos App (archived)

Archived on 2026-10-22. The app and its libs were removed from `main`.

|                          |                                                                                                                                            |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Last commit with the app | `165a2ac140a2bedea2cbb27e20993e2a4f305432` (2026-10-13)                                                                                    |
| App at that commit       | [apps/cloudinary-photos-app](https://github.com/destngx/monorepo/tree/165a2ac140a2bedea2cbb27e20993e2a4f305432/apps/cloudinary-photos-app) |
| Libs at that commit      | [libs/cloudinary-photos-app](https://github.com/destngx/monorepo/tree/165a2ac140a2bedea2cbb27e20993e2a4f305432/libs/cloudinary-photos-app) |

## What it was

A Next.js photo manager built on Cloudinary. It uploaded images with `CldUploadButton`, listed the gallery with `cloudinary.v2.search`, marked favorites with a `favorite` tag, and used Cloudinary folders as albums. The detail page showed EXIF data and had a "Generative Fill" button that used Cloudinary's `fillBackground` transform. EXIF tags were stored in MongoDB.

## Why it was archived

- It could not build or run. Its dependencies (`cloudinary`, `next-cloudinary`, `exifreader`, `mongodb`) were removed from the root `package.json` in the Sprint 1 cleanup, `.nxignore` kept it out of the Nx graph, and ESLint and lint-staged skipped it.
- Nothing deployed it and no other project imported its libs.
- The goal behind it is reaching all photos and videos from any device. A hosted image service fits that poorly: the Cloudinary free tier is small, video bandwidth costs money, and the app would need ongoing work. A self-hosted photo server does it out of the box.

## What replaced it

[Immich](../../immich/README.md), self-hosted on an always-on PC, with automatic backup from phones and tablets.

## Restore

To look at the code locally:

```sh
git show 165a2ac1:apps/cloudinary-photos-app/src/app/page.tsx
git worktree add ../monorepo-worktrees/cloudinary-photos-app-archive-view 165a2ac1
```

To bring it back onto a branch:

```sh
git checkout 165a2ac1 -- apps/cloudinary-photos-app libs/cloudinary-photos-app
```

It also needs its dependencies added back to the root `package.json`, Nx build and serve targets, a Cloudinary account, and a MongoDB instance.
