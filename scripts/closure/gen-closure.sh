#!/usr/bin/env bash
# Recompute the ssa_only build closure of ./cmd/yak-ssa into closure/*.txt.
#
# Usage:
#   scripts/closure/gen-closure.sh [source-repo]
#
# source-repo defaults to $YAK_SOURCE, then this repo. The fork checkout
# (which carries go.sum and a warm module cache) is the usual source.
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${PYTHON:-python}" "$HERE/closure.py" gen "$@"
