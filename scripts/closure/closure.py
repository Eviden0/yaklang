#!/usr/bin/env python3
"""Compute and materialize the exact `ssa_only` build closure of cmd/yak-ssa.

The standalone SSA engine mirrors upstream yaklang paths one-to-one. Instead of
hand-picking directories, the closure is derived from the Go toolchain so it can
never drift from what `go build -tags ssa_only ./cmd/yak-ssa` actually needs.

Subcommands:
  gen     [source]        Recompute closure/*.txt from the source repo.
  extract [source] [dest] Copy every closure package directory into dest.
  check   [source]        Recompute and diff against the committed closure so
                          drift (new or dropped internal deps) is caught in CI.

Source defaults to $YAK_SOURCE, then the current repo root. Dest defaults to
the current repo root.
"""
import argparse
import os
import shutil
import subprocess
import sys

MODULE = "github.com/yaklang/yaklang"
TAGS = "ssa_only"
ENTRY = "./cmd/yak-ssa"

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", ".."))
CLOSURE_DIR = os.path.join(ROOT, "closure")
PACKAGES_FILE = os.path.join(CLOSURE_DIR, "closure.packages.txt")
DIRS_FILE = os.path.join(CLOSURE_DIR, "closure.dirs.txt")
BUILD_FILE = os.path.join(CLOSURE_DIR, "closure.build.txt")


def run_go_list(source):
    """Return [(rel_import_path, rel_dir, [go_files])] for internal packages."""
    fmt = "{{if .Standard}}{{else}}{{.ImportPath}}\t{{.Dir}}\t{{join .GoFiles \",\"}}{{end}}"
    proc = subprocess.run(
        ["go", "list", "-deps", "-tags", TAGS, "-f", fmt, ENTRY],
        cwd=source, capture_output=True, text=True,
    )
    if proc.returncode != 0:
        sys.stderr.write(proc.stderr)
        raise SystemExit("go list failed (exit %d)" % proc.returncode)

    rows = []
    prefix = MODULE + "/"
    for line in proc.stdout.splitlines():
        if not line.strip():
            continue
        parts = line.split("\t")
        if len(parts) != 3:
            continue
        import_path, go_dir, files = parts
        if not import_path.startswith(prefix):
            continue
        rel_import = import_path[len(prefix):]
        rel_dir = os.path.relpath(go_dir, source)
        go_files = [f for f in files.split(",") if f]
        rows.append((rel_import, rel_dir, go_files))
    rows.sort()
    return rows


def cmd_gen(args):
    source = args.source
    rows = run_go_list(source)
    if not rows:
        raise SystemExit("closure is empty; wrong source or tags?")

    os.makedirs(CLOSURE_DIR, exist_ok=True)
    with open(PACKAGES_FILE, "w", newline="\n") as f:
        for rel_import, _, _ in rows:
            f.write(rel_import + "\n")
    with open(DIRS_FILE, "w", newline="\n") as f:
        for _, rel_dir, _ in sorted(rows, key=lambda r: r[1]):
            f.write(rel_dir.replace("\\", "/") + "\n")
    with open(BUILD_FILE, "w", newline="\n") as f:
        f.write("# internal packages: %d\n" % len(rows))
        f.write("# non-test go files: %d\n" % sum(len(r[2]) for r in rows))
        f.write("# <rel_import>\t<non-test go file count>\n")
        for rel_import, _, go_files in rows:
            f.write("%s\t%d\n" % (rel_import, len(go_files)))

    print("wrote %d packages, %d non-test go files -> %s"
          % (len(rows), sum(len(r[2]) for r in rows), os.path.relpath(CLOSURE_DIR, ROOT)))
    return 0


def _copytree(src, dst):
    # dirs_exist_ok keeps overlapping package dirs (a dir is also an ancestor of
    # another closed-over package) from conflicting.
    shutil.copytree(src, dst, dirs_exist_ok=True)


def cmd_extract(args):
    source = args.source
    dest = args.dest
    if not os.path.isfile(os.path.join(source, "go.mod")):
        raise SystemExit("source has no go.mod: %s" % source)
    if not os.path.isfile(PACKAGES_FILE):
        raise SystemExit("run `closure.py gen` first (missing %s)" % PACKAGES_FILE)

    with open(DIRS_FILE, encoding="utf-8") as f:
        rel_dirs = [line.strip() for line in f if line.strip()]

    copied = 0
    for rel_dir in rel_dirs:
        src = os.path.join(source, rel_dir)
        dst = os.path.join(dest, rel_dir)
        if not os.path.isdir(src):
            raise SystemExit("closure dir missing in source: %s" % src)
        _copytree(src, dst)
        copied += 1

    # Module metadata + root-level CI helpers that the closure build needs.
    for name in ("go.mod", "go.sum"):
        src = os.path.join(source, name)
        if os.path.isfile(src):
            shutil.copy2(src, os.path.join(dest, name))

    print("copied %d package directories from %s -> %s" % (copied, source, dest))
    return 0


def cmd_check(args):
    source = args.source
    rows = run_go_list(source)
    current = {}
    for rel_import, rel_dir, _ in rows:
        current[rel_import] = rel_dir
    if not os.path.isfile(PACKAGES_FILE):
        raise SystemExit("no committed closure; run `closure.py gen` first")

    committed = {}
    with open(PACKAGES_FILE, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if line:
                committed[line] = True

    added = sorted(set(current) - set(committed))
    removed = sorted(set(committed) - set(current))
    if not added and not removed:
        print("closure OK: %d internal packages" % len(current))
        return 0
    for p in added:
        print("+ %s" % p)
    for p in removed:
        print("- %s" % p)
    print("closure drift detected; review and rerun `closure.py gen` in a dedicated PR", file=sys.stderr)
    return 1


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="cmd", required=True)

    def add_source(p):
        p.add_argument("source", nargs="?",
                       default=os.environ.get("YAK_SOURCE") or ROOT,
                       help="upstream/fork repo root (default: $YAK_SOURCE or repo root)")

    p_gen = sub.add_parser("gen", help="recompute closure manifests")
    add_source(p_gen)
    p_gen.set_defaults(func=cmd_gen)

    p_extract = sub.add_parser("extract", help="copy closure package dirs into dest")
    add_source(p_extract)
    p_extract.add_argument("dest", nargs="?", default=ROOT,
                           help="destination engine repo (default: this repo)")
    p_extract.set_defaults(func=cmd_extract)

    p_check = sub.add_parser("check", help="detect closure drift")
    add_source(p_check)
    p_check.set_defaults(func=cmd_check)

    args = parser.parse_args(argv)
    return args.func(args)


if __name__ == "__main__":
    sys.exit(main())
