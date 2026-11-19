#!/usr/bin/env bash
# Detect drift between closure/*.txt and what ./cmd/yak-ssa actually imports.
# Exits non-zero when internal dependencies were added or dropped, so closure
# expansion always lands in its own reviewable PR.
#
# Usage:
#   scripts/closure/check-closure.sh [source-repo]
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${PYTHON:-python}" "$HERE/closure.py" check "$@"
