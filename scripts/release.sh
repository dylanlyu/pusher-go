#!/usr/bin/env bash
# Release script for pusher-go multi-module monorepo.
#
# Usage:
#   ./scripts/release.sh <version>            release all modules at the same version
#   ./scripts/release.sh <module> <version>   release a single module
#
# Examples:
#   ./scripts/release.sh 1.0.3
#   ./scripts/release.sh channels 1.1.0
#   ./scripts/release.sh internal 1.0.3
#
# When releasing a library module (internal, config), the script also
# updates the require version in all consumer go.mod files.
set -euo pipefail

# ---------------------------------------------------------------------------
# Dependency graph: which modules consume each library module
# ---------------------------------------------------------------------------
declare -A CONSUMERS
CONSUMERS[internal]="channels beams"
CONSUMERS[config]="channels beams"

# All independently-tagged modules (excludes pusher which is an empty shell)
ALL_MODULES=(channels beams internal config)

# ---------------------------------------------------------------------------
# Parse arguments
# ---------------------------------------------------------------------------
if [[ $# -eq 1 ]]; then
  TARGET="all"
  VERSION="$1"
elif [[ $# -eq 2 ]]; then
  TARGET="$1"
  VERSION="$2"
else
  echo "Usage:" >&2
  echo "  $0 <version>            — release all modules" >&2
  echo "  $0 <module> <version>   — release a single module" >&2
  exit 1
fi

# Validate module name for single releases
if [[ "$TARGET" != "all" ]]; then
  VALID=false
  for m in "${ALL_MODULES[@]}"; do
    [[ "$m" == "$TARGET" ]] && VALID=true && break
  done
  if [[ "$VALID" == false ]]; then
    echo "Unknown module '${TARGET}'. Valid: ${ALL_MODULES[*]}" >&2
    exit 1
  fi
fi

TAG="v${VERSION}"
export GONOSUMDB="github.com/dylanlyu/pusher-go/*"

echo "==> Target : ${TARGET}"
echo "==> Version: ${TAG}"
echo ""

# ---------------------------------------------------------------------------
# Determine which modules to tag and which go.mod files need updating
# ---------------------------------------------------------------------------
if [[ "$TARGET" == "all" ]]; then
  MODULES_TO_TAG=("${ALL_MODULES[@]}")
  # Update all consumer go.mod files to reference the new shared version
  MODS_TO_UPDATE="channels beams"
  UPDATE_DEPS=(internal config)
else
  MODULES_TO_TAG=("$TARGET")
  # If the released module is a library, update its consumers
  MODS_TO_UPDATE=""
  UPDATE_DEPS=()
  if [[ -n "${CONSUMERS[$TARGET]+x}" ]]; then
    MODS_TO_UPDATE="${CONSUMERS[$TARGET]}"
    UPDATE_DEPS=("$TARGET")
  fi
fi

# ---------------------------------------------------------------------------
# Step 1: Update go.mod in consumer modules
# ---------------------------------------------------------------------------
if [[ -n "$MODS_TO_UPDATE" ]]; then
  echo "==> Updating go.mod"
  for dep in "${UPDATE_DEPS[@]}"; do
    for mod in $MODS_TO_UPDATE; do
      if grep -q "github.com/dylanlyu/pusher-go/${dep}" "${mod}/go.mod"; then
        sed -i '' \
          "s|github.com/dylanlyu/pusher-go/${dep} v[0-9]*\.[0-9]*\.[0-9]*|github.com/dylanlyu/pusher-go/${dep} ${TAG}|g" \
          "${mod}/go.mod"
        echo "    ${mod}/go.mod → ${dep} ${TAG}"
      fi
    done
  done
  echo ""
fi

# ---------------------------------------------------------------------------
# Step 2: Build + test affected modules
# ---------------------------------------------------------------------------
echo "==> Verifying build and tests"
if [[ "$TARGET" == "all" ]]; then
  BUILD_MODULES=("${ALL_MODULES[@]}")
else
  BUILD_MODULES=("$TARGET")
  # Also build/test consumers if go.mod changed
  if [[ -n "$MODS_TO_UPDATE" ]]; then
    for m in $MODS_TO_UPDATE; do
      BUILD_MODULES+=("$m")
    done
  fi
fi

for mod in "${BUILD_MODULES[@]}"; do
  go -C "$mod" build ./...
  go -C "$mod" test -race ./...
  echo "    ${mod}: OK"
done
echo ""

# ---------------------------------------------------------------------------
# Step 3: Commit go.mod changes (if any)
# ---------------------------------------------------------------------------
CHANGED_MODS=()
for mod in $MODS_TO_UPDATE; do
  git diff --quiet "${mod}/go.mod" || CHANGED_MODS+=("${mod}/go.mod")
done

if [[ ${#CHANGED_MODS[@]} -gt 0 ]]; then
  git add "${CHANGED_MODS[@]}"
  git commit -m "chore(release): bump internal deps to ${TAG}"
  echo "==> Committed go.mod updates"
fi

# ---------------------------------------------------------------------------
# Step 4: Create tags
# ---------------------------------------------------------------------------
echo "==> Creating tags"

if [[ "$TARGET" == "all" ]]; then
  # Root flat tag only for all-modules releases
  LAST_TAG=$(git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
  CHANGELOG=$(git log "${LAST_TAG}..HEAD" --oneline --no-merges 2>/dev/null || true)
  git tag -a "${TAG}" -m "Release ${TAG}

${CHANGELOG}"
  echo "    ${TAG}"
fi

for mod in "${MODULES_TO_TAG[@]}"; do
  git tag "${mod}/${TAG}"
  echo "    ${mod}/${TAG}"
done
echo ""

# ---------------------------------------------------------------------------
# Step 5: Push
# ---------------------------------------------------------------------------
echo "==> Pushing"
git push origin "$(git branch --show-current)"

TAGS_TO_PUSH=()
[[ "$TARGET" == "all" ]] && TAGS_TO_PUSH+=("${TAG}")
for mod in "${MODULES_TO_TAG[@]}"; do
  TAGS_TO_PUSH+=("${mod}/${TAG}")
done
git push origin "${TAGS_TO_PUSH[@]}"

echo ""
echo "Released: ${TAG}"
echo "Tags    : ${TAGS_TO_PUSH[*]}"
