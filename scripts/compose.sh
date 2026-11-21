#!/usr/bin/env bash
# Merges upstream main with the branches listed in features.txt onto a
# throwaway local `build` branch and writes the build parameters to
# $GITHUB_OUTPUT. Run it from a checkout of fork-ci with full history.
#
# The merge commits use a fixed identity and the upstream commit date, so every
# job composing the same inputs gets the same commit hash.
#
# The fork also carries every upstream branch, so features are listed
# explicitly in features.txt instead of matched by name. Read before switching
# to the build branch, where the file does not exist.
set -euo pipefail

: "${UPSTREAM_REPO:?}" "${BUILD_REV:?}"
UPSTREAM_URL="${UPSTREAM_URL:-https://github.com/${UPSTREAM_REPO}.git}"

features=$(grep -vE '^[[:space:]]*(#|$)' features.txt || true)

git remote add upstream "$UPSTREAM_URL"
git fetch --no-tags upstream main

up=$(git rev-parse upstream/main)
when=$(git show -s --format=%cI "$up")
export GIT_AUTHOR_NAME=fork-ci GIT_COMMITTER_NAME=fork-ci
export GIT_AUTHOR_EMAIL=fork-ci@users.noreply.github.com GIT_COMMITTER_EMAIL=fork-ci@users.noreply.github.com
export GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when"

git checkout -q -B build "$up"
fingerprint_input="$BUILD_REV $up"
merged=""

for name in $features; do
  ref="origin/$name"
  if ! git rev-parse --verify -q "refs/remotes/$ref" > /dev/null; then
    echo "::error::features.txt lists $name but $ref does not exist in the fork."
    exit 1
  fi
  echo "Merging $ref"
  if ! git merge --no-edit --no-ff "$ref"; then
    git merge --abort || true
    echo "::error::Conflict merging $ref into upstream/main ($up). Rebase that branch on upstream and push it again."
    exit 1
  fi
  fingerprint_input="$fingerprint_input $(git rev-parse "$ref")"
  merged="$merged $ref"
done

owner=$(echo "$GITHUB_REPOSITORY_OWNER" | tr '[:upper:]' '[:lower:]')
{
  echo "fp=$(echo "$fingerprint_input" | sha256sum | cut -c1-12)"
  echo "up7=${up:0:7}"
  echo "date=$(date -u +%Y%m%d)"
  echo "prefix=ghcr.io/${owner}"
  echo "build_commit=$(git rev-parse HEAD)"
} >> "$GITHUB_OUTPUT"
{
  echo "### Composed tree"
  echo "- upstream/main: \`$up\`"
  echo "- features:${merged:- none}"
} >> "$GITHUB_STEP_SUMMARY"
