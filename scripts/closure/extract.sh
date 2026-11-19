#!/usr/bin/env bash
# Copy every closure package directory from source into dest, preserving paths.
#
# Usage:
#   scripts/closure/extract.sh [source-repo] [dest-repo]
#
# source-repo defaults to $YAK_SOURCE, dest-repo defaults to this repo.
set -euo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
exec "${PYTHON:-python}" "$HERE/closure.py" extract "$@"
