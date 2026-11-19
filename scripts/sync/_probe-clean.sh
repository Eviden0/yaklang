#!/usr/bin/env bash
# Read-only drill helper: find upstream commits whose path-filtered patch
# applies cleanly onto the current tree (git apply --check, no tree changes).
set -uo pipefail
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

paths=$(sed -e 's/#.*$//' -e 's/[[:space:]]*$//' scripts/sync/paths.txt | sed '/^$/d' | tr '\n' ' ')
extend="${1:-40}"
mapfile -t shas < <(git log --reverse --format='%H' refs/remotes/upstream/main -- $paths)
tmp=$(mktemp)
trap 'rm -f "$tmp"' EXIT
found=0
i=0
for sha in "${shas[@]}"; do
  i=$((i + 1))
  if [ "$i" -gt "$extend" ]; then break; fi
  git diff --binary "$sha^" "$sha" -- $paths > "$tmp" 2>/dev/null
  if [ -s "$tmp" ] && git apply --check --whitespace=nowarn "$tmp" 2>/dev/null; then
    subj=$(git log -1 --format='%s' "$sha")
    echo "CLEAN [$i] ${sha:0:12}  $subj"
    found=$((found + 1))
    [ "$found" -ge 5 ] && break
  fi
done
echo "scanned up to $extend candidates, clean=$found"
