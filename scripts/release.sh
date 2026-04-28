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

# All independently-tagged modules (excludes pusher which is an empty shell)
ALL_MODULES="channels beams internal config"

# Returns the space-separated list of consumer modules for a given library module.
consumers_of() {
  case "$1" in
    internal) echo "channels beams" ;;
    config)   echo "channels beams" ;;
    *)        echo "" ;;
  esac
}

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
  for m in $ALL_MODULES; do
    [[ "$m" == "$TARGET" ]] && VALID=true && break
  done
  if [[ "$VALID" == false ]]; then
    echo "Unknown module '${TARGET}'. Valid: ${ALL_MODULES}" >&2
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
  MODULES_TO_TAG="$ALL_MODULES"
  MODS_TO_UPDATE="channels beams"
  UPDATE_DEPS="internal config"
else
  MODULES_TO_TAG="$TARGET"
  MODS_TO_UPDATE="$(consumers_of "$TARGET")"
  UPDATE_DEPS=""
  [[ -n "$MODS_TO_UPDATE" ]] && UPDATE_DEPS="$TARGET"
fi

# ---------------------------------------------------------------------------
# Step 1: Update go.mod in consumer modules
# ---------------------------------------------------------------------------
if [[ -n "$MODS_TO_UPDATE" ]]; then
  echo "==> Updating go.mod"
  for dep in $UPDATE_DEPS; do
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
# Use a temporary go.work so local modules resolve without needing
# the new tags to exist in VCS yet.
# ---------------------------------------------------------------------------
echo "==> Verifying build and tests"
BUILD_MODULES="$MODULES_TO_TAG"
for m in $MODS_TO_UPDATE; do
  case " $BUILD_MODULES " in
    *" $m "*) ;;
    *) BUILD_MODULES="$BUILD_MODULES $m" ;;
  esac
done

# Create temporary go.work for local resolution during build.
# Stash any existing go.work so we can restore it after.
GOWORK_EXISTED=false
if [[ -f go.work ]]; then
  mv go.work go.work.bak
  mv go.work.sum go.work.sum.bak 2>/dev/null || true
  GOWORK_EXISTED=true
fi
go work init
for m in $ALL_MODULES pusher; do
  go work use "./$m"
  # Force local resolution for the current release tag even before it exists in VCS.
  # This prevents Go from trying to fetch the untagged version from the proxy.
  go work edit -replace "github.com/dylanlyu/pusher-go/${m}@${TAG}=./${m}"
done
restore_gowork() {
  rm -f go.work go.work.sum
  if [[ "$GOWORK_EXISTED" == true ]]; then
    mv go.work.bak go.work
    mv go.work.sum.bak go.work.sum 2>/dev/null || true
  fi
}
trap restore_gowork EXIT

for mod in $BUILD_MODULES; do
  go -C "$mod" build ./...
  go -C "$mod" test -race ./...
  echo "    ${mod}: OK"
done

restore_gowork
trap - EXIT
echo ""

# ---------------------------------------------------------------------------
# Step 3: Commit go.mod changes (if any)
# ---------------------------------------------------------------------------
CHANGED_MODS=""
for mod in $MODS_TO_UPDATE; do
  git diff --quiet "${mod}/go.mod" || CHANGED_MODS="$CHANGED_MODS ${mod}/go.mod"
done

if [[ -n "$CHANGED_MODS" ]]; then
  git add $CHANGED_MODS
  git commit -m "chore(release): bump internal deps to ${TAG}"
  echo "==> Committed go.mod updates"
fi

# ---------------------------------------------------------------------------
# Step 4: Create tags
# ---------------------------------------------------------------------------
echo "==> Creating tags"

if [[ "$TARGET" == "all" ]]; then
  LAST_TAG=$(git describe --tags --match 'v[0-9]*.[0-9]*.[0-9]*' --abbrev=0 2>/dev/null || git rev-list --max-parents=0 HEAD)
  CHANGELOG=$(git log "${LAST_TAG}..HEAD" --oneline --no-merges 2>/dev/null || true)
  git tag -a "${TAG}" -m "Release ${TAG}

${CHANGELOG}"
  echo "    ${TAG}"
fi

for mod in $MODULES_TO_TAG; do
  git tag "${mod}/${TAG}"
  echo "    ${mod}/${TAG}"
done
echo ""

# ---------------------------------------------------------------------------
# Step 5: Push
# ---------------------------------------------------------------------------
echo "==> Pushing"
git push origin "$(git branch --show-current)"

TAGS_TO_PUSH=""
[[ "$TARGET" == "all" ]] && TAGS_TO_PUSH="$TAG"
for mod in $MODULES_TO_TAG; do
  TAGS_TO_PUSH="$TAGS_TO_PUSH ${mod}/${TAG}"
done
git push origin $TAGS_TO_PUSH

echo ""
echo "Released: ${TAG}"
echo "Tags    :${TAGS_TO_PUSH}"
