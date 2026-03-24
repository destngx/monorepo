Monorepo & Clean Architecture with Nx

Combining Nx monorepos with Clean Architecture gives teams a scalable, maintainable codebase where domain boundaries are enforced structurally — not just by convention. Nx acts as the tooling backbone that makes Clean Architecture's layered separation tangible at the file-system and build level.
What Is an Nx Monorepo?

A monorepo is a single Git repository that holds multiple applications and libraries along with their shared tooling. Nx extends this with a rich project graph, computation caching, affected-build detection, and code generators — making it a full platform for large-scale, multi-team development.

Key monorepo benefits Nx provides:

    Atomic changes — update a shared API and all downstream consumers in one commit

    Affected builds — only rebuild/test projects impacted by a given change (nx affected -t build)

    Developer mobility — consistent tooling and scaffolding across all projects

    Boundary enforcement — module tags and lint rules prevent illegal cross-layer imports

Mapping Clean Architecture to Nx

Clean Architecture defines concentric layers (Domain → Application → Infrastructure → Presentation). In Nx, each layer becomes a library project with explicit tags and enforced dependencies.
Clean Architecture Layer Nx Library Type Depends On
Domain (entities, value objects) domain lib Nothing
Application (use cases, interfaces) application lib domain only
Infrastructure (DB, APIs, adapters) infrastructure lib application, domain
Presentation / UI feature or ui lib application, domain
App Shell apps/\* All layers

This structure is directly supported in Nx by tagging libraries and enforcing rules via @nx/enforce-module-boundaries in ESLint.
Project Structure Example

A typical Nx + Clean Architecture workspace looks like this:

text
my-workspace/
├── apps/
│ ├── web/ # React/Angular app shell
│ └── api/ # NestJS API shell
├── libs/
│ ├── domain/ # Entities, value objects, interfaces
│ ├── application/ # Use cases, service ports
│ ├── infrastructure/# DB repos, HTTP clients, adapters
│ └── ui/ # Shared UI components
├── nx.json
└── project.json

The apps/ directory houses only thin shells (~20% unique code), while the real logic lives in libs/. This keeps apps "appropriately sized, mutually independent, and both flexible and scalable".​
Enforcing Layer Boundaries

Nx's module-boundaries rule prevents architectural violations at lint time:

json
// .eslintrc.json
{
"rules": {
"@nx/enforce-module-boundaries": ["error", {
"depConstraints": [
{ "sourceTag": "scope:ui", "onlyDependOnLibsWithTags": ["scope:application", "scope:domain"] },
{ "sourceTag": "scope:application", "onlyDependOnLibsWithTags": ["scope:domain"] },
{ "sourceTag": "scope:domain", "onlyDependOnLibsWithTags": [] }
]
}]
}
}

This ensures clean deployment boundaries and eliminates hidden coupling at CI time.​
Domain-Driven Design Integration

Nx pairs well with Domain-Driven Design (DDD), where each bounded context maps to a folder group (e.g., libs/ordering/, libs/catalog/) containing its own domain, application, and infrastructure layers. Independent deployment cycles are supported — your admin interface can evolve at a different cadence than your customer-facing storefront.
Practical Real-World References

    clean-architecture-monorepo (React/TS/TDD)
    — showcases Clean Architecture in Nx with TDD applied to a chat app​

    Angular + NestJS monorepo (Bennet303)
    — full-stack Nx monorepo with Angular/Ionic frontend, NestJS backend, Storybook, Cypress, and Swagger​

    Angular Architects blog — details implementing strategic DDD bounded contexts with Angular and Nx​

When to Choose Nx Over Turborepo
Factor Nx Turborepo
Project graph & affected builds ✅ Native ⚠️ Partial
Boundary/policy enforcement ✅ Built-in ❌ Bring your own
Code generators & scaffolding ✅ Rich plugins ❌ Minimal
Learning curve Higher Lower
Best for Product platforms, multi-team Small teams, simple pipelines

​

Nx is the stronger choice when your monorepo is a product platform with multiple teams and you need standardized scaffolding and guardrails — exactly the context where Clean Architecture shines.​
Prepared using Claude Sonnet 4.6 Thinking
nx monorepo, clean architecture, FSD and vertical slide architecture

These four architectural approaches are complementary, not competing — in a well-designed Nx monorepo, you can use all four simultaneously: Nx provides tooling and enforcement, Clean Architecture defines dependency direction, FSD structures frontend apps, and Vertical Slice Architecture (VSA) organizes backend/feature delivery. Here's a complete breakdown of how they relate and fit together.
At-a-Glance Comparison
Dimension Clean Architecture Feature-Sliced Design (FSD) Vertical Slice Architecture
Primary concern Dependency direction & isolation Frontend repo structure Feature delivery isolation
Axis of organization Technical layers (concentric) Feature + layer (horizontal slices within vertical) Full-stack vertical slices per feature
Typical scope Full-stack / backend Frontend apps only Backend / full-stack features
Key rule Dependencies always point inward Higher layers import only from lower layers Each slice is self-contained
Risk if misused Over-engineering; rigid layer ceremony Inconsistent public APIs at scale Code duplication across slices

Clean Architecture: The Dependency Rule

Clean Architecture organizes code into concentric rings — Domain → Application → Infrastructure → Presentation — where outer layers depend on inner ones, never the reverse. In an Nx monorepo, each ring becomes a dedicated library project with @nx/enforce-module-boundaries lint rules ensuring the dependency rule is structurally enforced, not just a convention.

    Domain lib → zero dependencies, pure business entities and interfaces

    Application lib → depends on Domain only (use cases, ports)

    Infrastructure lib → depends on Application + Domain (DB repos, HTTP clients)

    Presentation lib → depends on Application + Domain (UI, controllers)

Feature-Sliced Design (FSD): Frontend Layering

FSD is a frontend-specific methodology that answers "where does this code live?" in a standardized way. It introduces two axes: layers (vertical priority) and slices (domain partitions within each layer).

FSD's seven-layer hierarchy, from high to low:​

text
app/ ← global providers, routing, init
pages/ ← route-level page composition
widgets/ ← large composite UI blocks (Navbar, Sidebar)
features/ ← user-facing actions, use cases & UI bindings
entities/ ← domain models, core rules, domain UI
shared/ ← utilities, UI kit, API clients, config

The golden import rule: a module can only import from layers strictly below it in the hierarchy — preventing circular deps. FSD maps cleanly onto Clean Architecture: entities/ → Domain, features/ → Use Cases, shared/ → Interface Adapters/Frameworks.
Vertical Slice Architecture (VSA): Feature Delivery

VSA organizes code by business capability rather than technical layer — a "UserRegistration" slice contains its own handler, validator, DB query, and response mapper in one folder. Each slice is independently deployable and testable, making it ideal for backend APIs and microservice-style boundaries.

The best hybrid pattern, per practitioners:

    Use Domain + Infrastructure layers from Clean Architecture + VSA for Application + Presentation layers. The Domain model remains central and shared across all slices, while each slice implements its own use case logic independently.

text
src/
├── domain/ ← Central (Clean Arch: entities, interfaces)
├── infrastructure/ ← Central (Clean Arch: DB, external services)
└── features/
├── user-registration/ ← Vertical Slice (handler + logic + query)
├── product-search/ ← Vertical Slice
└── checkout/ ← Vertical Slice

Combining All Four in Nx Monorepo

In a real Nx workspace, the four patterns apply at different scopes:

text
my-workspace/
├── apps/
│ ├── web/src/ ← FSD inside frontend app
│ │ ├── app/
│ │ ├── pages/
│ │ ├── widgets/
│ │ ├── features/
│ │ ├── entities/
│ │ └── shared/
│ └── api/ ← VSA inside backend app
│ └── features/
│ ├── user-registration/
│ └── product-catalog/
├── libs/
│ ├── domain/ ← Clean Arch: shared Domain layer
│ ├── infrastructure/ ← Clean Arch: shared Infrastructure layer
│ └── ui/ ← Shared FSD "shared/" components
└── nx.json ← Enforces module boundaries

Nx enforcement ties it all together via module boundary tags: scope:domain, scope:infra, scope:feature, scope:ui — so the dependency rules of both Clean Architecture and FSD are validated at lint time during CI.​
When to Use What

    All greenfield enterprise projects: Start with Nx monorepo + Clean Architecture at the lib level; apply FSD inside each frontend app, VSA inside each backend service

    FSD alone (without Clean Arch): Sufficient for frontend-only apps with moderate business logic​

    VSA alone: Works well for CRUD-heavy backends with low domain complexity​

    Clean Architecture alone: Best when domain rules are complex and long-lived, e.g., financial or healthcare systems​

    Hybrid (Clean + VSA): The most pragmatic backend pattern for mid-to-large teams — domain/infra are shared, application layer is sliced

Prepared using Claude Sonnet 4.6 Thinking
why not move all shared and entities to the libs?

This is actually a nuanced and common architectural debate. The short answer: you should move them — but only selectively, based on clear criteria. Moving everything indiscriminately to libs/ creates its own set of serious problems.
The Real Distinction: Workspace-Scoped vs. App-Scoped

Nx's libs/ folder has a specific semantic: it holds code that is cacheable, publishable, and shared across multiple projects. FSD's shared/ and entities/ inside an app are a different concept — they represent code that is scoped to that app's domain, even if not used by other apps. Conflating the two scopes is the root cause of monorepo mess.​
Dimension FSD shared/ + entities/ inside app Nx libs/
Scope This app's domain only Cross-app, workspace-wide
Stability expectation Can evolve freely with the app Requires stable public API
Ownership App team Shared/platform team
Nx affected impact Only affects this app Affects all consumers
Right to exist Always Only when shared by 2+ apps

Problem 1: The "Dumping Ground" Anti-Pattern

Moving all shared/ and entities/ into libs/ without FSD structure creates an unscoped global shared folder — the single most common cause of unmaintainable monorepos. When everything is in libs/shared/, no one can answer:​

    Is this entity truly cross-app, or just app A's internal model?

    Is this utility safe to modify without breaking app B?

    Who owns this code?

The FSD monorepo guide explicitly warns: "You avoid a 'global shared folder' that becomes a dumping ground. Instead, shared is scoped and structured."​
Problem 2: Premature Extraction = Hidden Coupling

Extracting entities/ to libs/ before a second consumer exists is premature abstraction. If app-web and app-admin have slightly different models of User, forcing them to share a single libs/domain/user entity couples their evolution. Each change to that shared entity now triggers nx affected rebuilds across all consumers, slowing CI and increasing coordination overhead.​
Problem 3: FSD Layer Rules Still Apply Inside Libs

Even if you move entities and shared to libs/, those libs must still respect FSD's import hierarchy internally. You'd need to replicate the layer structure inside each lib:​

text
libs/
└── domain-sales/src/
├── entities/ # FSD entities layer — scoped to sales domain
├── shared/ # FSD shared layer — scoped to sales domain
└── index.ts # Explicit public API

This is the "scoped shared" pattern — FSD structure inside a named domain lib, not one flat global libs/shared/. It preserves FSD's benefits while enabling cross-app reuse.​
The Decision Rule: When to Move to Libs

Apply this checklist before extracting anything to libs/:

    ✅ Two or more apps consume it — single-consumer code stays inside the app

    ✅ It has a stable, documented public API (index.ts exports only what's supported)

    ✅ It has a clear owner (a team or domain group)

    ✅ It carries domain-agnostic or cross-domain meaning (e.g., Button, AuthToken, HttpClient)

    ❌ Do NOT move if the entity is specific to one app's view of a domain (e.g., web-app's CartItem model is not the same as mobile-app's)

Recommended Structure

text
my-workspace/
├── apps/
│ ├── web/src/
│ │ ├── entities/ ← app-specific domain models (stays here)
│ │ ├── shared/ ← app-specific utils/UI (stays here)
│ │ └── features/
│ └── admin/src/
│ ├── entities/ ← admin's own domain models
│ └── shared/
├── libs/
│ ├── ui/ ← Moved: design system used by web + admin
│ ├── domain-auth/ ← Moved: Auth entities + shared, used by all apps
│ │ ├── entities/ ← FSD structure preserved inside lib
│ │ ├── shared/
│ │ └── index.ts ← Public API only
│ └── infrastructure/ ← Moved: HTTP client, analytics, logging

The key insight: libs/ is not a replacement for FSD's shared/ and entities/ — it is the promotion destination for app-internal code that has earned cross-app status through demonstrated reuse.
Prepared using Claude Sonnet 4.6 Thinking
if the frontend is build from nextjs, I think only apps and features need to keep in application folder

Your instinct is mostly correct but needs one important refinement. In a Next.js App Router + Nx monorepo, the routing runtime (app/) and business interaction layer (features/) are the strongest candidates to stay inside the application — but the reasoning is more nuanced than just "keep those two".
Why Next.js Changes the FSD Equation

Next.js App Router introduces a hard constraint: the app/ directory is a file-system router — routes must physically exist there. This means Next.js app/ already absorbs both FSD's app layer (global providers, init) and FSD's pages layer (route-level composition). The recommended solution from the FSD team is to separate the Next.js routing shell from the FSD business layers:

text
apps/web/
├── app/ ← Next.js App Router (routing shell ONLY)
│ ├── (public)/
│ │ └── page.tsx ← just imports from src/pages/
│ └── \_providers/
│ └── Providers.tsx
└── src/ ← FSD business layers live here
├── pages/ ← FSD pages (route-level composition)
├── widgets/
├── features/
├── entities/
└── shared/

The app/ directory becomes a thin routing adapter that delegates to src/pages/ immediately — it should have zero business logic.
What Should Stay in the App vs. Move to Libs
FSD Layer Stay in apps/web/src/ Move to libs/ Reasoning
app/ (Next.js routing) ✅ Always ❌ Never Next.js file-system routing constraint ​
pages/ (FSD) ✅ Usually ⚠️ Only if page logic is reused Pages are route-bound compositions, highly app-specific ​
features/ ✅ Usually ⚠️ Only if feature is used in 2+ apps User interaction scenarios tied to specific app context ​
widgets/ ⚠️ App-specific widgets ✅ Reusable composite blocks (Header, Sidebar) Widgets without app-specific state can be shared ​
entities/ ⚠️ App-specific views ✅ Core domain models shared across apps Promote when 2+ apps need the same domain model ​
shared/ ⚠️ App-specific utils ✅ Design system, HTTP clients, config Business-agnostic code is almost always lib-ready ​
The Corrected Mental Model for Next.js in Nx

Your thinking is right in spirit — app/ routing and features/ are the most app-specific layers. But the more precise rule is:

    Keep everything in the app by default. Promote to libs/ only when a second consumer appears — not before.​

The practical result in a Next.js Nx monorepo ends up being very close to what you described:

text
my-workspace/
├── apps/
│ └── web/
│ ├── app/ ← Next.js routing shell (always stays)
│ └── src/
│ ├── pages/ ← stays (route-level, app-specific)
│ ├── features/ ← stays (user interactions, app-specific)
│ └── widgets/ ← app-specific widgets only
├── libs/
│ ├── ui/ ← Promoted: shared design system (widgets/shared)
│ │ ├── widgets/ ← Reusable composite blocks
│ │ └── shared/ ← Tokens, primitives, UI kit
│ ├── domain-auth/ ← Promoted: cross-app entities
│ │ └── entities/
│ └── infrastructure/ ← Promoted: HTTP clients, analytics, logging
│ └── shared/

One Critical Nuance: FSD Import Rules Still Apply Across the Boundary

Even after promotion to libs/, the FSD unidirectional import rule must hold. Code in apps/web/src/features/ can import from libs/domain-auth/entities/ (lower layer) — but libs/ must never import back from apps/. Each libs/ package must expose a clean index.ts public API and have zero knowledge of any consuming app. This is where Nx's enforce-module-boundaries lint rule acts as the enforcement mechanism.
Prepared using Claude Sonnet 4.6 Thinking
Follow-ups
How to configure Nx to build Next.js app with root app folder
Examples of FSD structure importing into Next.js app folder
Pros and cons of root app folder vs src/app in FSD Next.js
How to handle Next.js pages router with FSD in Nx monorepo
Best way to organize route groups in FSD with App Router
