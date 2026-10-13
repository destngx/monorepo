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

# nx-ignore compares HEAD with the last successful deployment (VERCEL_GIT_PREVIOUS_SHA), but when that commit
# is missing from Vercel's shallow clone it silently falls back to HEAD^, so a push of several commits is judged
# on its last commit alone. Fetch the base when needed and build whenever it cannot be compared reliably.
base="${VERCEL_GIT_PREVIOUS_SHA:-}"
if [ -z "$base" ]; then
  echo "Building: no previous deployment to compare against"
  exit 1
fi
if ! git cat-file -e "$base^{commit}" 2>/dev/null; then
  echo "Fetching previous deployment commit $base"
  if ! GIT_TERMINAL_PROMPT=0 git fetch --quiet --depth=1 origin "$base"; then
    echo "Fetching $base failed"
  fi
fi
if ! git cat-file -e "$base^{commit}" 2>/dev/null; then
  echo "Building: previous deployment commit $base is not in the clone"
  exit 1
fi

PNPM_BIN=$(mktemp -d)
corepack enable --install-directory "$PNPM_BIN" pnpm
PATH="$PNPM_BIN:$PATH" COREPACK_ENABLE_DOWNLOAD_PROMPT=0 exec pnpm dlx nx-ignore@latest portfolio-landpage --base="$base"
