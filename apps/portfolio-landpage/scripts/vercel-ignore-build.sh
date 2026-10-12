#!/bin/sh
# Vercel "Ignored Build Step" for portfolio-landpage: exit 0 skips the deployment, exit 1 builds it.
# Preview deployments run only for feat/portfolio-* branches; main builds when Nx marks the app as affected.
# Vercel runs this from the project root directory (apps/portfolio-landpage).
set -eu

branch="${VERCEL_GIT_COMMIT_REF:-}"

case "$branch" in
  main) ;;
  feat/portfolio-*)
    echo "Building $branch"
    exit 1
    ;;
  *)
    echo "Skipping $branch: only main and feat/portfolio-* deploy portfolio-landpage"
    exit 0
    ;;
esac

cd ../..
PNPM_BIN=$(mktemp -d)
corepack enable --install-directory "$PNPM_BIN" pnpm
PATH="$PNPM_BIN:$PATH" COREPACK_ENABLE_DOWNLOAD_PROMPT=0 exec pnpm dlx nx-ignore@latest portfolio-landpage
