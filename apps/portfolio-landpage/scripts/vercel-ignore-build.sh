#!/bin/sh
# Vercel ignored build step: exit 0 skips the deploy, exit 1 builds it.
#
# Builds when the app, or a root file that changes how it installs, differs from the last successful deploy of this
# branch (VERCEL_GIT_PREVIOUS_SHA, else the parent commit). The app imports nothing from libs/, so these paths are
# all of its inputs; add any shared folder here if that changes. Anything unexpected builds, since a needless
# deploy is cheaper than a missed one.
set -u

cd "$(dirname "$0")/.." || exit 1

base="${VERCEL_GIT_PREVIOUS_SHA:-HEAD^}"

if ! git cat-file -e "${base}^{commit}" 2>/dev/null; then
  echo "Base ${base} is not in this clone, building."
  exit 1
fi

set -- . ../../package.json ../../pnpm-lock.yaml ../../pnpm-workspace.yaml ../../.npmrc

git diff --quiet "$base" HEAD -- "$@"
status=$?

case $status in
  0)
    echo "No changes to portfolio-landpage since ${base}, skipping."
    exit 0
    ;;
  1)
    echo "portfolio-landpage changed since ${base}, building:"
    git diff --name-only "$base" HEAD -- "$@"
    exit 1
    ;;
  *)
    echo "git diff failed (exit ${status}), building."
    exit 1
    ;;
esac
