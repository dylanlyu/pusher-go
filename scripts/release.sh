#!/usr/bin/env bash
# Usage: ./scripts/release.sh <version>  e.g. ./scripts/release.sh 1.0.3
#
# Release sequence:
#   1. Update internal dep versions in go.mod
#   2. Commit
#   3. Create root tag + subdirectory-prefixed tags (pointing to updated commit)
#   4. Push commits and tags
#
# GONOSUMDB bypasses the sum database during the bootstrap window
# (new tags take a few minutes to be indexed by sum.golang.org).
set -euo pipefail

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "Usage: $0 <version>  (e.g. 1.0.3)" >&2
  exit 1
fi

TAG="v${VERSION}"
MODULES=(channels beams internal config)
export GONOSUMDB="github.com/dylanlyu/pusher-go/*"

echo "==> Releasing ${TAG}"

# 1. Update internal dependency versions in consuming modules
for mod in channels beams; do
  sed -i '' \
    "s|github.com/dylanlyu/pusher-go/internal v[0-9]*\.[0-9]*\.[0-9]*|github.com/dylanlyu/pusher-go/internal ${TAG}|g" \
    "${mod}/go.mod"
  echo "    updated ${mod}/go.mod → internal ${TAG}"
done

# 2. Build + test with updated versions (workspace resolves locally)
echo "==> Verifying build and tests"
go -C channels build ./...
go -C beams build ./...
go -C channels test -race ./...
go -C beams test -race ./...
echo "    all OK"

# 3. Commit go.mod changes
git add channels/go.mod beams/go.mod
if ! git diff --cached --quiet; then
  git commit -m "chore(release): bump internal deps to ${TAG}"
fi

# 4. Create annotated root tag
LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
CHANGELOG=$(git log "${LAST_TAG}..HEAD" --oneline --no-merges 2>/dev/null || true)
git tag -a "${TAG}" -m "Release ${TAG}

${CHANGELOG}"
echo "    created tag ${TAG}"

# 5. Create subdirectory-prefixed tags
for mod in "${MODULES[@]}"; do
  git tag "${mod}/${TAG}"
  echo "    created tag ${mod}/${TAG}"
done

# 6. Push commits and all tags
git push origin "$(git branch --show-current)"
git push origin "${TAG}" $(printf '%s/%s ' "${MODULES[@]/%//${TAG}}")

echo ""
echo "Released: ${TAG}"
echo "Tags: ${TAG} $(printf '%s/%s ' "${MODULES[@]/%//${TAG}}")"
