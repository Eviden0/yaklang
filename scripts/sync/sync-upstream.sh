#!/usr/bin/env bash
# Path-filtered cherry-pick of upstream SSA commits into this standalone engine.
#
# This repo's history is a single squashed extraction commit, NOT a fork of
# upstream, so `git cherry-pick` across histories is not used. Instead each
# upstream commit's diff is restricted to the SSA pathspecs in paths.txt and
# applied with `git apply` (context based, with a 3-way attempt first).
#
# Usage:
#   scripts/sync/sync-upstream.sh                 # dry run: list candidate commits
#   scripts/sync/sync-upstream.sh --apply         # apply sequentially, stop at first conflict
#   scripts/sync/sync-upstream.sh --apply --max 2 # apply at most 2 (drills)
#   scripts/sync/sync-upstream.sh --apply --only <sha>  # apply one specific commit
#
# Env:
#   UPSTREAM_REMOTE   default: upstream
#   UPSTREAM_BRANCH   default: main
#   UPSTREAM_DEPTH    shallow fetch depth, default 300
#
# On a failed apply the script writes scripts/sync/conflicts/<sha>.md and exits
# non-zero so a human can resolve it before continuing.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
PATHS_FILE="$HERE/paths.txt"
STATE="$HERE/upstream-state.yaml"
CONFLICTS_DIR="$HERE/conflicts"

UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_BRANCH="${UPSTREAM_BRANCH:-main}"
UPSTREAM_DEPTH="${UPSTREAM_DEPTH:-300}"

#   --only <sha>      apply only this upstream commit (repeatable); for drills
#   --max <n>         stop after applying at most n commits (0 = unlimited)
APPLY=0
ONLY_SHAS=()
MAX_APPLY=0
while [ $# -gt 0 ]; do
  case "$1" in
    --apply) APPLY=1; shift ;;
    ""|--dry-run) APPLY=0; shift ;;
    --only) ONLY_SHAS+=("${2:?--only needs a sha}"); shift 2 ;;
    --max) MAX_APPLY="${2:?--max needs a number}"; shift 2 ;;
    -h|--help) awk 'NR>1{if(/^#/){sub(/^# ?/,"");print}else{exit}}' "$0"; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

cd "$ROOT"

# pathspecs: skip comments and blank lines
mapfile -t paths < <(sed -e 's/#.*$//' -e 's/[[:space:]]*$//' "$PATHS_FILE" | sed '/^$/d')
if [ "${#paths[@]}" -eq 0 ]; then
  echo "no pathspecs in $PATHS_FILE" >&2
  exit 1
fi

last_synced="$(awk -F': *' '/^last_synced:/{print $2}' "$STATE" | awk '{print $1}')"

echo "== fetching $UPSTREAM_REMOTE/$UPSTREAM_BRANCH (depth=$UPSTREAM_DEPTH, blob:none) =="
git fetch --filter=blob:none --depth="$UPSTREAM_DEPTH" \
  "$UPSTREAM_REMOTE" "$UPSTREAM_BRANCH" >/dev/null
upstream_sha="$(git rev-parse "refs/remotes/$UPSTREAM_REMOTE/$UPSTREAM_BRANCH" 2>/dev/null \
  || git rev-parse FETCH_HEAD)"

echo "last_synced = $last_synced"
echo "upstream    = $upstream_sha"

if git cat-file -e "${last_synced}^{commit}" 2>/dev/null; then
  range="${last_synced}..${upstream_sha}"
  echo "candidate range: $range"
else
  range="${upstream_sha}"
  echo "WARN: last_synced ($last_synced) is not in the fetched history (depth=$UPSTREAM_DEPTH)." >&2
  echo "      Falling back to listing the newest commits touching SSA paths; deepen with" >&2
  echo "      UPSTREAM_DEPTH=1000 if you need the full range since last_synced." >&2
fi

# Commits touching our paths, oldest first. --name-only needs only tree objects,
# so this stays cheap even with a blob:none fetch.
mapfile -t commits < <(git log --reverse --format='%H%x09%s' "$range" -- "${paths[@]}")
if [ "${#commits[@]}" -eq 0 ]; then
  echo "no upstream commits touch SSA paths in this range."
  exit 0
fi

# Restrict to explicitly requested commits (drills), matching by short or full sha.
if [ "${#ONLY_SHAS[@]}" -gt 0 ]; then
  filtered=()
  for line in "${commits[@]}"; do
    sha="${line%%$'\t'*}"
    for want in "${ONLY_SHAS[@]}"; do
      case "$sha" in
        "$want"*) filtered+=("$line"); break ;;
      esac
    done
  done
  commits=("${filtered[@]}")
  if [ "${#commits[@]}" -eq 0 ]; then
    echo "none of the --only shas touch SSA paths in range $range" >&2
    exit 1
  fi
fi

echo "== ${#commits[@]} candidate commit(s) touching SSA paths =="
for line in "${commits[@]}"; do
  sha="${line%%$'\t'*}"; subj="${line#*$'\t'}"
  printf '%s  %s\n' "${sha:0:12}" "$subj"
done

if [ "$APPLY" -eq 0 ]; then
  echo
  echo "dry run only. Re-run with --apply to land these (stops at first conflict)."
  exit 0
fi

mkdir -p "$CONFLICTS_DIR"
tmp_patch="$(mktemp)"
trap 'rm -f "$tmp_patch"' EXIT

applied=0
for line in "${commits[@]}"; do
  if [ "$MAX_APPLY" -gt 0 ] && [ "$applied" -ge "$MAX_APPLY" ]; then
    echo "--max $MAX_APPLY reached; stopping."
    break
  fi
  sha="${line%%$'\t'*}"; subj="${line#*$'\t'}"
  short="${sha:0:12}"
  echo "== applying $short: $subj =="
  git diff --binary "$sha^" "$sha" -- "${paths[@]}" > "$tmp_patch"
  if git apply --3way --whitespace=nowarn "$tmp_patch" 2>/dev/null \
     || git apply --whitespace=nowarn "$tmp_patch"; then
    # Stage only SSA-path changes so the sync commit never sweeps unrelated
    # untracked files (local-patches/, scripts/, etc.) into it. Skip pathspecs
    # that don't exist in this tree (e.g. cmd/ssa-scan/ was not extracted), since
    # `git add` treats a non-matching pathspec as a fatal error.
    add_paths=()
    for p in "${paths[@]}"; do [ -e "$p" ] && add_paths+=("$p"); done
    if [ "${#add_paths[@]}" -gt 0 ]; then
      git add -A -- "${add_paths[@]}"
    fi
    git commit -q -m "sync(upstream): $subj

upstream: $sha"
    echo "   applied and committed"
    applied=$((applied + 1))
  else
    report="$CONFLICTS_DIR/$sha.md"
    {
      echo "# Upstream sync conflict: $short"
      echo
      echo "- upstream commit: \`$sha\`"
      echo "- subject: $subj"
      echo "- pathspecs: \`$PATHS_FILE\`"
      echo
      echo "## Files touched by the upstream commit (all paths)"
      git diff --name-only "$sha^" "$sha"
      echo
      echo "## Files touched inside our SSA pathspecs"
      git diff --name-only "$sha^" "$sha" -- "${paths[@]}"
      echo
      echo "## Suggested action"
      echo "Resolve conflicts by hand, commit, then re-run \`sync-upstream.sh --apply\`."
      echo "If the commit also edits shared code outside the SSA paths, evaluate a"
      echo "separate closure expansion instead of widening this path filter."
    } > "$report"
    echo "   CONFLICT: see $report" >&2
    exit 1
  fi
done

# Only advance last_synced when the WHOLE range landed. Partial applies
# (--only / --max) must not claim the tree reflects upstream HEAD.
if [ "${#ONLY_SHAS[@]}" -gt 0 ] || [ "$MAX_APPLY" -gt 0 ]; then
  echo "== partial apply: leaving last_synced at $(awk -F': *' '/^last_synced:/{print $2}' "$STATE") =="
  echo "done (partial; last_synced unchanged)."
  exit 0
fi

echo "== updating last_synced -> $upstream_sha =="
sed -i "s/^last_synced:.*/last_synced: $upstream_sha/" "$STATE"
git add "$STATE"
git commit -q -m "sync(upstream): record last_synced=$upstream_sha"

echo "done."
