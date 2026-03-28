# Monorepo Architecture Guide

This document outlines the architectural patterns used in this Nx monorepo and explains the organization of each application and shared library.

## Overview

This monorepo combines **Clean Architecture**, **Feature-Sliced Design (FSD)**, and **Vertical Slice Architecture (VSA)** principles:

- **Next.js Apps**: Use Feature-Sliced Design (FSD)
- **Shared Libs**: Follow Clean Architecture principles with workspace-level boundaries

---

## Architecture Patterns

### 1. Feature-Sliced Design (FSD) - Next.js Applications

All Next.js applications follow FSD structure with 6 layers (from high to low priority):

```
apps/{app-name}/
├── app/                 # Next.js routing shell (thin adapter only) (JS/TS only)
│   └── [route]/        # Delegates to src/pages/
├── src/                 # JS/TS Source code or Python application source
└── requirements.txt     # (Python only) Dependencies
    ├── pages/          # Route-level page compositions
    ├── features/       # User interactions & feature-specific logic
    ├── widgets/        # Composite UI blocks (reusable across pages)
    ├── entities/       # App-specific domain models
    └── shared/         # App utilities, hooks, constants
```

#### FSD Import Rule (Golden Rule)

```
app/ → pages/ → features/ → widgets/ → entities/ → shared/
↑ can import from anything to the right (lower priority)
↑ CANNOT import upward (prevents circular deps)
```

#### FSD Examples

**cloudinary-photos-app:**

```
src/
├── pages/albums/          # Album list & detail page compositions
├── pages/gallery/         # Gallery view page composition
├── pages/detail/          # Detail view page composition
├── pages/favorites/       # Favorites page composition
├── features/              # Search, Upload interactions
├── widgets/               # AlbumCard, GalleryGrid, FavoritesList, SearchForm, UploadButton
├── entities/              # Image, Folder, SearchResult types
└── shared/                # Utilities, hooks, constants
```

**portfolio-landpage:**

```
src/
├── pages/home/            # Home page composition
├── features/              # Any interactive features
├── widgets/               # Reusable UI blocks (Header, Hero, Portfolio)
├── entities/              # Portfolio item types
└── shared/                # Utilities, styling constants
```

**vnstock-server (Python):**

```
apps/vnstock-server/
├── src/                   # FastAPI application logic
│   └── main.py            # Entrypoint wrapping vnstock library
├── tests/                 # Pytest suite
└── requirements.txt       # Python dependencies
```

**wealth-management:**

```
src/
├── pages/                 # Page compositions for each route
├── features/              # Domain features (accounts, loans, budget, transactions)
│   ├── accounts/
│   ├── loans/
│   ├── budget/
│   ├── transactions/
│   ├── investments/
│   ├── goals/
│   ├── chat/
│   └── settings/
├── components/            # (Legacy) Components - should migrate to widgets
├── widgets/               # Composite UI blocks
├── entities/              # Domain models (Account, Transaction, etc.)
└── shared/                # Utilities, hooks, config, constants
```

---

## Shared Libraries (libs/)

Shared code ONLY goes to `libs/` when:

1. ✅ Used by 2+ applications, OR
2. ✅ Cross-domain (auth, logging, shared UI), OR
3. ✅ Stable with documented public API

Shared code STAYS app-local (`src/shared/`) when:

- ❌ Used by only one app
- ❌ Frequently changing with app evolution
- ❌ App-specific utilities or hooks

### Current Shared Libraries Structure

```
libs/
├── cloudinary-photos-app/          # App-scoped libs (will consolidate)
│   ├── types/                      # Image, Folder, SearchResult types
│   ├── utils/                      # Cloudinary utilities
│   ├── components/                 # UI components library
│   │   ├── ui/                     # Primitive UI (Button, Card, Input)
│   │   ├── layout/                 # Layout components (AuthLayout, MainLayout)
│   │   └── icons/                  # Icon library
│   └── global-store/               # Global state (if needed)
│
├── portfolio-landpage/             # App-scoped libs
│   └── components/                 # Portfolio UI components
│
├── wealth-management/              # Shared wealth features
│   └── src/                        # Shared hooks, types, services
│
└── [FUTURE] shared/                # Workspace-wide shared (when needed)
    ├── ui/                         # Design system (if multiple apps)
    ├── domain/                     # Cross-app entities
    └── infrastructure/             # HTTP clients, logging, analytics
```

### Publishing & Boundaries

Each library in `libs/` must have:

1. **Public API**: `index.ts` or `src/index.ts` exporting only intended API
2. **Documentation**: README explaining purpose and usage
3. **Version**: Update when breaking changes occur
4. **Tags**: Nx project tags for boundary enforcement (e.g., `scope:ui`, `scope:domain`)

---

## App-Specific Vs. Shared: Decision Matrix

| Code Type           | Single App                      | Multi-App                | Decision                                |
| ------------------- | ------------------------------- | ------------------------ | --------------------------------------- |
| **UI Components**   | `src/widgets/` or `src/shared/` | `libs/{app}/components/` | lib if 2+ apps; otherwise app-local     |
| **Domain Entities** | `src/entities/`                 | `libs/domain-{feature}/` | lib if shared logic; otherwise separate |
| **Utilities**       | `src/shared/utils/`             | `libs/shared/utils/`     | lib only if truly cross-app             |
| **Constants**       | `src/shared/constants/`         | `libs/shared/config/`    | lib only if stable                      |
| **Hooks**           | `src/shared/hooks/`             | `libs/shared/hooks/`     | lib only if reused                      |
| **API Clients**     | `src/shared/api/`               | `libs/infrastructure/`   | lib for external API integrations       |

**Rule of thumb**: Every line of code in `libs/` should answer "Why am I here?" with a clear, documented reason.

---

## Next.js App Structure (Detailed)

### app/ Directory (Routing Shell Only)

```tsx
// apps/{app}/app/albums/page.tsx
// ✅ CORRECT: Thin wrapper delegating to src/pages/
import { AlbumsPage } from '@/pages/albums';
export default AlbumsPage;

// ❌ WRONG: Business logic in routing
import { useQuery } from 'react-query';
export default function AlbumsPage() {
  const { data } = useQuery(/* ... */);
  // ← This belongs in src/pages/, not here
}
```

### src/pages/ Directory (Page Compositions)

```tsx
// apps/{app}/src/pages/albums/index.tsx
// Composes widgets, features, and fetches data for this route
import { AlbumsPage as Component } from './content';
export { Component as AlbumsPage };

// apps/{app}/src/pages/albums/content.tsx
// OR inline in index.tsx
import { AlbumCard } from '@/widgets/album-card';
import { getAlbums } from '@/features/albums';

export async function AlbumsPage() {
  const albums = await getAlbums();
  return (
    <div>
      {albums.map((album) => (
        <AlbumCard key={album.id} album={album} />
      ))}
    </div>
  );
}
```

### src/features/ Directory (Feature Logic)

```tsx
// apps/{app}/src/features/search/search-by-tag.ts
// Business logic for search interaction
export async function searchByTag(tag: string) {
  const results = await cloudinary.v2.search.expression(`tag:${tag}`).execute();
  return results;
}

// apps/{app}/src/features/upload/upload-handler.tsx
// Client-side upload interaction
('use client');
export function UploadHandler() {
  // Event handlers, state for upload flow
}
```

### src/widgets/ Directory (Composite UI)

```tsx
// apps/{app}/src/widgets/album-card/index.ts
export { AlbumCard } from './ui';

// apps/{app}/src/widgets/album-card/ui.tsx
// Pure presentational, receives all props
import { Album } from '@/entities';
export interface AlbumCardProps {
  album: Album;
  onView?: (id: string) => void;
}
export function AlbumCard({ album, onView }: AlbumCardProps) {
  return ( /* JSX */ );
}
```

### src/entities/ Directory (Domain Models)

```ts
// apps/{app}/src/entities/index.ts
export type Image = {
  public_id: string;
  secure_url: string;
  width: number;
  height: number;
  tags: string[];
};

export type Album = {
  id: string;
  name: string;
  images: Image[];
};
```

### src/shared/ Directory (Utilities)

```
src/shared/
├── config/        # App configuration
├── constants/     # Constants
├── utils/         # Utility functions
├── hooks/         # Custom React hooks
└── types.ts       # App-wide types
```

---

---

## Nx Module Boundary Rules (ESLint)

ESLint enforces architectural boundaries via `@nx/enforce-module-boundaries`:

```json
{
  "rules": {
    "@nx/enforce-module-boundaries": [
      "error",
      {
        "allow": ["@nx/next", "@nx/nest"],
        "depConstraints": [
          {
            "sourceTag": "scope:app",
            "onlyDependOnLibsWithTags": ["*"]
          },
          {
            "sourceTag": "scope:ui",
            "onlyDependOnLibsWithTags": ["scope:ui", "scope:shared"]
          },
          {
            "sourceTag": "scope:shared",
            "onlyDependOnLibsWithTags": []
          }
        ]
      }
    ]
  }
}
```

---

## Import Conventions

### Next.js Apps (src/ imports)

Use path aliases for clean imports:

```typescript
// ✅ Clean imports
import { AlbumCard } from '@/widgets/album-card';
import { getAlbums } from '@/features/albums';
import { Album } from '@/entities';
import { useDebounce } from '@/shared/hooks';

// ❌ Messy imports
import { AlbumCard } from '../../../widgets/album-card';
import { useDebounce } from '@mono/wealth-management/hooks';
```

tsconfig.json path aliases:

```json
{
  "compilerOptions": {
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  }
}
```

### Shared Libraries (Nx imports)

```typescript
// ✅ From app to lib
import { Button } from '@mono/cloudinary-photos-app/components/ui';

// ❌ Importing internals
import { Button } from '@mono/cloudinary-photos-app/components/ui/button';

// Public API via index.ts
// libs/cloudinary-photos-app/components/ui/index.ts
export { Button } from './button';
export { Card } from './card';
```

---

## Migration Path (Going Forward)

When code earns its place in `libs/`:

1. **Identify reuse**: Confirm 2+ apps need it
2. **Extract to lib**: Create `libs/{scope}/{feature}/`
3. **Document API**: Add `libs/{scope}/{feature}/README.md`
4. **Update imports**: Change all consumer imports
5. **Tag for boundaries**: Add `tags: ["scope:domain", "scope:ui"]` to `project.json`

Example: Moving shared UI components

```bash
# Before: Components scattered in each app
libs/
├── cloudinary-photos-app/components/ui/
├── portfolio-landpage/components/ui/
└── wealth-management/components/ui/

# After: Unified design system
libs/
├── ui/
│   ├── primitives/    # Button, Card, Input, etc.
│   ├── composites/    # Header, Footer, Sidebar
│   └── README.md
├── cloudinary-photos-app/components/  # App-specific only
├── portfolio-landpage/components/     # App-specific only
└── wealth-management/components/      # App-specific only
```

---

## Checklist for Clean Architecture

- [ ] Each app has correct FSD or Clean Arch structure
- [ ] ESLint enforces module boundaries (`bun nx run-many -t lint`)
- [ ] All builds pass (`bun nx run-many -t build`)
- [ ] Shared libs have public APIs (index.ts exports)
- [ ] No circular dependencies between layers
- [ ] Each layer has clear responsibility
- [ ] Type-safe imports using path aliases
- [ ] Documentation updated for significant changes

---

## References

- [Feature-Sliced Design (FSD)](https://feature-sliced.design/)
- [Clean Architecture by Robert C. Martin](https://blog.cleancoder.com/uncle-bob/2012/08/13/the-clean-architecture.html)
- [Vertical Slice Architecture](https://www.jimmybogard.com/vertical-slice-architecture/)
- [Nx Module Boundaries](https://nx.dev/features/module-boundary-rules)
- [Next.js App Router](https://nextjs.org/docs/app)
- [NestJS Documentation](https://docs.nestjs.com/)
