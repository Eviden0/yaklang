# Upstream sync

Follow `yaklang/yaklang` `main` by replaying **only** the upstream commits that
touch SSA paths onto this standalone tree. This repo is a squashed extraction,
not a fork, so `git cherry-pick` across histories is not used; instead each
upstream commit's diff is restricted to the pathspecs in `paths.txt` and
applied with `git apply`.

## Files

- `paths.txt` — SSA/SyntaxFlow pathspecs. Only commits touching these paths are
  considered. Changes to shared code outside them (ypb/consts/schema) are **not**
  auto-applied; evaluate a closure expansion in a separate PR instead.
- `upstream-state.yaml` — `last_synced` records the newest upstream commit the
  tree reflects. Only advanced after a full, conflict-free `--apply`.
- `sync-upstream.sh` — the driver.
- `_probe-clean.sh` — drill helper: read-only scan for upstream commits whose
  path-filtered patch applies cleanly onto the current tree (`git apply --check`,
  no tree mutation). Handy to find a safe commit before applying for real.

## Workflow

```bash
# 1. See what upstream has that we don't (dry run, lists oldest-first)
scripts/sync/sync-upstream.sh

# 2. Land them (stops at the first conflict, writes conflicts/<sha>.md, exits 1)
scripts/sync/sync-upstream.sh --apply

# Drills / controlled landing:
scripts/sync/sync-upstream.sh --apply --max 2          # at most 2 commits
scripts/sync/sync-upstream.sh --apply --only <sha>     # one specific commit
```

`--only` / `--max` are **partial** applies: they do **not** advance
`last_synced` (the tree still does not reflect upstream HEAD).

## Depth caveat

`last_synced` is far behind `main`, so the default `UPSTREAM_DEPTH=300` shallow
fetch may not reach it. The script then prints a `WARN` and falls back to
listing the newest SSA commits in the fetched window. To compute the true range
`last_synced..main`, deepen:

```bash
UPSTREAM_DEPTH=1000 scripts/sync/sync-upstream.sh
```

## Conflict handling

On a failed apply the script writes `conflicts/<sha>.md` (commit, subject, all
touched files, files inside pathspecs, suggested action) and exits non-zero.
Resolve by hand, commit, then re-run. Conflict reports are gitignored.

## Drill status (2026-11-19)

Validated end-to-end against `foo/upstream@a448eb5` (fetched `depth=300`):

- **Clean apply** — `8707c93b3d1c feat(syntaxflow): add authentication/authorization
  builtin rules` landed as a single `sync(upstream): …` commit containing only the
  6 rule files under `common/syntaxflow/…`, with an `upstream: <sha>` trailer.
  `last_synced` correctly left unchanged (partial apply). Commit then reverted to
  keep the extracted baseline pristine.
- **Conflict** — `1369e16ea83c fix(ssadb): …` produced a readable
  `conflicts/1369e16…md`; the commit references files (`const_table.go`, test
  files) that post-date our `238faf8` base, so `git apply` reports them as
  missing and writes the report as designed. No working-tree residue.
