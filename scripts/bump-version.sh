#!/usr/bin/env bash
# bump-version.sh — interactive release script for ftm.
#
# Mirrors the UX of curie's scripts/release.mjs: pick a bump (patch/minor/
# major/custom) or pass an explicit x.y.z, see a plan, confirm, and let the
# script write the version into every place that needs it, commit, tag, and
# push.
#
# Updates:
#   - internal/version/version.go    (var Version = "...")
#   - web/package.json        ("version": "...")
#   - cmd/ftm-desktop/wails.json             ("version": "..." AND "productVersion": "...")
#
# Usage:
#   scripts/bump-version.sh                       # interactive menu
#   scripts/bump-version.sh patch                 # bump patch level
#   scripts/bump-version.sh 0.11.0                # set explicit version
#   scripts/bump-version.sh patch --dry-run       # plan only, no writes
#   scripts/bump-version.sh patch --no-push       # commit + tag, skip push
#   scripts/bump-version.sh patch --allow-dirty   # proceed if working tree dirty
#   scripts/bump-version.sh patch --yes           # skip confirmation prompt
#
# Exit codes:
#   0  success (or dry-run)
#   1  user error, validation failure, or git failure

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$ROOT"

# --- colors (auto-disabled when not a TTY) ---------------------------------
if [[ -t 1 ]]; then
  C_RESET=$'\033[0m'
  C_BOLD=$'\033[1m'
  C_DIM=$'\033[2m'
  C_RED=$'\033[31m'
  C_GREEN=$'\033[32m'
  C_YELLOW=$'\033[33m'
  C_CYAN=$'\033[36m'
  C_WHITE=$'\033[37m'
else
  C_RESET=''; C_BOLD=''; C_DIM=''; C_RED=''; C_GREEN=''; C_YELLOW=''; C_CYAN=''; C_WHITE=''
fi

paint() { printf '%s%s%s' "$2" "$1" "$C_RESET"; }
fail()  { printf '\n %s %s\n' "$(paint '✗' "$C_RED$C_BOLD")" "$1" >&2; exit 1; }
info()  { printf ' %s %s\n' "$(paint '…' "$C_DIM")" "$1"; }

# --- arg parsing -----------------------------------------------------------
DRY_RUN=0
NO_PUSH=0
ALLOW_DIRTY=0
YES=0
BUMP_ARG=""

usage() {
  cat <<EOF
Usage: scripts/bump-version.sh [bump|version] [flags]

Bump (interactive menu if omitted):
  patch | minor | major | x.y.z

Flags:
  --dry-run        Show plan and exit without writing or committing
  --no-push        Commit + tag but don't push to origin
  --allow-dirty    Proceed even with a dirty working tree
  --yes, -y        Skip confirmation prompt
  -h, --help       Show this help

Examples:
  scripts/bump-version.sh                   # interactive
  scripts/bump-version.sh patch             # bump patch
  scripts/bump-version.sh 0.11.0            # set explicit version
  scripts/bump-version.sh patch --dry-run
EOF
}

for arg in "$@"; do
  case "$arg" in
    --dry-run)      DRY_RUN=1 ;;
    --no-push)      NO_PUSH=1 ;;
    --allow-dirty)  ALLOW_DIRTY=1 ;;
    --yes|-y)       YES=1 ;;
    -h|--help)      usage; exit 0 ;;
    --)             shift; break ;;
    -*)             fail "unknown flag: $arg" ;;
    *)              BUMP_ARG="$arg" ;;
  esac
done

# --- current version (source of truth: internal/version/version.go) -------
VERSION_FILE="internal/version/version.go"
[[ -f "$VERSION_FILE" ]] || fail "missing $VERSION_FILE"

CURRENT=$(grep -E '^var Version = "' "$VERSION_FILE" | sed -E 's/^var Version = "([^"]+)"$/\1/')
[[ -n "$CURRENT" ]] || fail "could not parse current version from $VERSION_FILE"
[[ "$CURRENT" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "current version '$CURRENT' is not semver"

# --- bash 3.2 compat -------------------------------------------------------
# macOS still ships bash 3.2.57 (last GPLv2). `${var,,}` lowercase is bash 4+,
# so use tr. Same for other features; keep this in mind when extending.
to_lower() {
  printf '%s' "$1" | tr '[:upper:]' '[:lower:]'
}

# --- semver helpers --------------------------------------------------------
parse_semver() {
  local v="$1"
  [[ "$v" =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)$ ]] || fail "invalid semver: $v"
  MAJOR=${BASH_REMATCH[1]}; MINOR=${BASH_REMATCH[2]}; PATCH=${BASH_REMATCH[3]}
}

next_version() {
  local current="$1" bump="$2"
  if [[ "$bump" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    echo "$bump"
    return
  fi
  parse_semver "$current"
  case "$bump" in
    major) echo "$((MAJOR+1)).0.0" ;;
    minor) echo "${MAJOR}.$((MINOR+1)).0" ;;
    patch) echo "${MAJOR}.${MINOR}.$((PATCH+1))" ;;
    *)     fail "unknown bump: $bump (use patch|minor|major|x.y.z)" ;;
  esac
}

# --- banner -----------------------------------------------------------------
banner() {
  printf '\n'
  printf ' %s %s\n' \
    "$(paint 'ftm' "$C_BOLD$C_CYAN")" \
    "$(paint 'release' "$C_BOLD$C_WHITE")"
  printf '\n'
  printf ' %s %s\n' "$(paint 'current' "$C_DIM")" "$(paint "v$CURRENT" "$C_BOLD$C_WHITE")"
  if (( DRY_RUN )); then
    printf ' %s %s\n' "$(paint 'mode' "$C_YELLOW")" "dry-run (no writes)"
  fi
  printf '\n'
}

# --- interactive menu ------------------------------------------------------
choose_bump() {
  local patch minor major
  patch=$(next_version "$CURRENT" patch)
  minor=$(next_version "$CURRENT" minor)
  major=$(next_version "$CURRENT" major)

  {
    printf ' %s\n\n' "$(paint 'pick a bump' "$C_DIM")"
    printf ' %s %s %s %s %s\n' \
      "$(paint '1' "$C_CYAN")" "$(paint 'patch' "$C_BOLD")" \
      "$(paint "v$CURRENT →" "$C_DIM")" "$(paint "v$patch" "$C_GREEN")" \
      "$(paint '(bugfixes)' "$C_DIM")"
    printf ' %s %s %s %s %s\n' \
      "$(paint '2' "$C_CYAN")" "$(paint 'minor' "$C_BOLD")" \
      "$(paint "v$CURRENT →" "$C_DIM")" "$(paint "v$minor" "$C_GREEN")" \
      "$(paint '(features)' "$C_DIM")"
    printf ' %s %s %s %s %s\n' \
      "$(paint '3' "$C_CYAN")" "$(paint 'major' "$C_BOLD")" \
      "$(paint "v$CURRENT →" "$C_DIM")" "$(paint "v$major" "$C_GREEN")" \
      "$(paint '(breaking)' "$C_DIM")"
    printf ' %s %s %s\n' \
      "$(paint '4' "$C_CYAN")" "$(paint 'custom' "$C_BOLD")" \
      "$(paint 'type x.y.z' "$C_DIM")"
    printf ' %s %s\n\n' "$(paint 'q' "$C_CYAN")" "$(paint 'quit' "$C_DIM")"
  } >&2

  local answer
  read -r -p " $(paint '?' "$C_BOLD") choice $(paint '[1/2/3/4/q]' "$C_DIM"): " answer
  answer=$(to_lower "$answer")
  case "$answer" in
    q|quit|"")
      # We're inside $(...), so exit/exit N only exits the subshell, not
      # the parent script. Print a sentinel and let main catch it. The
      # subshell must still produce something on stdout for the caller to
      # read, otherwise BUMP ends up empty. Main prints the "aborted"
      # message once it sees the sentinel.
      echo "__ABORT__"
      ;;
    1|p|patch) echo "patch" ;;
    2|m|minor) echo "minor" ;;
    3|j|major) echo "major" ;;
    4|c|custom)
      local custom
      read -r -p " $(paint '?' "$C_BOLD") version $(paint '(x.y.z)' "$C_DIM"): " custom
      [[ "$custom" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "invalid version: $custom"
      echo "$custom"
      ;;
    *)
      # Allow typing the semver directly as the "choice".
      if [[ "$answer" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "$answer"
      else
        fail "invalid choice: $answer"
      fi
      ;;
  esac
}

# --- plan + confirmation ---------------------------------------------------
confirm() {
  local next="$1" tag="$2"
  if (( YES )); then return 0; fi

  printf '\n'
  printf ' %s\n' "$(paint 'plan' "$C_DIM")"
  printf ' %s\n' "$(paint '────' "$C_DIM")"
  printf ' %s %s\n' "$(paint 'version' "$C_DIM")" "$(paint "v$next" "$C_BOLD$C_GREEN")"
  printf ' %s %s\n' "$(paint 'tag' "$C_DIM")"     "$(paint "$tag" "$C_BOLD$C_CYAN")"
  printf ' %s %s\n' "$(paint 'files' "$C_DIM")" \
    "internal/version/version.go · web/package.json · cmd/ftm-desktop/wails.json"
  if (( NO_PUSH )); then
    printf ' %s %s\n' "$(paint 'git' "$C_DIM")" "commit + annotated tag $(paint '(no push)' "$C_YELLOW")"
  else
    printf ' %s %s\n' "$(paint 'git' "$C_DIM")" "commit + annotated tag + push"
  fi
  if (( DRY_RUN )); then
    printf ' %s\n' "$(paint 'dry-run — nothing will be written' "$C_YELLOW")"
  fi
  printf '\n'

  local answer
  read -r -p " $(paint '?' "$C_BOLD") proceed? $(paint '[y/N]' "$C_DIM"): " answer
  [[ "$(to_lower "$answer")" =~ ^y(es)?$ ]]
}

# --- file updaters ---------------------------------------------------------
# sed -i.bak works on both BSD (macOS) and GNU (Linux) sed. We delete the
# backup immediately so no .bak files leak into the working tree.

update_file() {
  local file="$1" pattern="$2" replacement="$3" label="$4"
  if [[ ! -f "$file" ]]; then fail "missing file: $file"; fi
  if ! sed -i.bak -E "$pattern" "$file"; then
    rm -f "${file}.bak"
    fail "failed to update $label in $file"
  fi
  rm -f "${file}.bak"
}

set_go_version() {
  local v="$1"
  update_file "$VERSION_FILE" \
    "s|^var Version = \".*\"|var Version = \"$v\"|" \
    "" "Version in $VERSION_FILE"
}

set_npm_version() {
  local file="$1" v="$2"
  update_file "$file" \
    "s|\"version\": \"[^\"]*\"|\"version\": \"$v\"|" \
    "" "\"version\" in $file"
}

set_wails_versions() {
  local file="$1" v="$2"
  # Both top-level "version" AND "productVersion" must move together so the
  # MSIX/Windows installer and the Wails build stay in sync.
  update_file "$file" \
    "s|\"version\": \"[^\"]*\"|\"version\": \"$v\"|" \
    "" "\"version\" in $file"
  update_file "$file" \
    "s|\"productVersion\": \"[^\"]*\"|\"productVersion\": \"$v\"|" \
    "" "\"productVersion\" in $file"
}

# --- main -------------------------------------------------------------------
banner

BUMP="${BUMP_ARG:-$(choose_bump)}"
if [[ "$BUMP" == "__ABORT__" || -z "$BUMP" ]]; then
  printf ' %s\n\n' "$(paint 'aborted' "$C_DIM")"
  exit 0
fi
NEXT=$(next_version "$CURRENT" "$BUMP")
TAG="v$NEXT"

if [[ -n "$BUMP_ARG" ]]; then
  printf ' %s %s %s %s\n\n' \
    "$(paint 'bump' "$C_DIM")" \
    "$(paint "$BUMP" "$C_BOLD")" \
    "$(paint "v$CURRENT →" "$C_DIM")" \
    "$(paint "v$NEXT" "$C_GREEN")"
fi

if ! confirm "$NEXT" "$TAG"; then
  printf '\n %s\n\n' "$(paint 'aborted' "$C_DIM")"
  exit 0
fi

if (( DRY_RUN )); then
  printf '\n%s %s\n\n' \
    "$(paint '○' "$C_YELLOW")" \
    "dry-run complete — would release $(paint "$TAG" "$C_BOLD")"
  exit 0
fi

# Working tree cleanliness
if [[ -n "$(git status --porcelain)" ]] && (( ! ALLOW_DIRTY )); then
  fail "working tree is dirty; commit/stash first or pass --allow-dirty"
fi

# Tag uniqueness
if git tag -l "$TAG" | grep -q .; then
  fail "tag $TAG already exists"
fi

info "writing version files"
set_go_version       "$NEXT"
set_npm_version      "web/package.json" "$NEXT"
set_wails_versions   "cmd/ftm-desktop/wails.json"     "$NEXT"

info "git commit"
git add "$VERSION_FILE" "web/package.json" "cmd/ftm-desktop/wails.json"
git commit -m "chore(release): ${TAG}"

info "tagging $TAG"
git tag -a "$TAG" -m "$TAG"

if (( ! NO_PUSH )); then
  info "pushing branch + tag"
  git push
  git push origin "$TAG"
  printf '\n%s %s\n%s\n\n' \
    "$(paint '✓' "$C_GREEN$C_BOLD")" \
    "released $(paint "$TAG" "$C_BOLD")" \
    "$(paint 'branch + tag pushed' "$C_DIM")"
else
  printf '\n%s %s\n%s\n\n' \
    "$(paint '✓' "$C_GREEN$C_BOLD")" \
    "released $(paint "$TAG" "$C_BOLD")" \
    "$(paint 'local only — push when ready' "$C_DIM")"
fi