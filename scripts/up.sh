#!/bin/sh
# Sync fork with upstream and reinstall: upstream/main -> main -> mine -> ~/.local/bin/ttt
set -e

cd "$(git rev-parse --show-toplevel)"

if [ -n "$(git status --porcelain)" ]; then
  echo "Working tree not clean — commit or stash first:"
  git status -s
  exit 1
fi

# go via mise if not on PATH
GO_PREFIX=""
command -v go >/dev/null 2>&1 || GO_PREFIX="mise exec go --"

git fetch upstream

git checkout -q main
git merge --ff-only upstream/main
git push -q origin main

git checkout -q mine
if ! git merge --no-edit main; then
  echo ""
  echo "Merge conflict — resolve, commit, then run: make install"
  exit 1
fi
git push -q origin mine

$GO_PREFIX make install
