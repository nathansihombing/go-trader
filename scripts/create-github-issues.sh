#!/usr/bin/env bash
# Create GitHub issues from docs/github_issues.json using GitHub CLI.
set -euo pipefail

usage() {
  cat <<'USAGE'
Usage: bash scripts/create-github-issues.sh [manifest] [--dry-run] [--markdown <file>]

Creates GitHub issues from a JSON manifest. Requires GitHub CLI (`gh`) unless
--dry-run or --markdown is used.

Arguments:
  manifest          JSON issue manifest (default: docs/github_issues.json)

Options:
  --dry-run         Print the gh commands that would be run; do not create issues.
  --markdown FILE   Export a human-readable Markdown issue list to FILE.
  -h, --help        Show this help.

Environment:
  GH_REPO           Override target repo, e.g. richkuo/go-trader.
USAGE
}

manifest="docs/github_issues.json"
dry_run=0
markdown_out=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --dry-run)
      dry_run=1
      shift
      ;;
    --markdown)
      if [[ $# -lt 2 ]]; then
        echo "--markdown requires a file path" >&2
        exit 2
      fi
      markdown_out="$2"
      shift 2
      ;;
    --*)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      manifest="$1"
      shift
      ;;
  esac
done

repo="${GH_REPO:-}"

if [[ ! -f "$manifest" ]]; then
  echo "manifest not found: $manifest" >&2
  exit 1
fi

if [[ -n "$markdown_out" ]]; then
  python3 - "$manifest" "$markdown_out" <<'PY'
import json
import pathlib
import sys

manifest = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
issues = json.loads(manifest.read_text())
lines = ["# GitHub Issue Backlog", ""]
for i, issue in enumerate(issues, 1):
    labels = ", ".join(issue.get("labels", [])) or "none"
    lines.extend([
        f"## {i}. {issue['title']}",
        "",
        f"Labels: {labels}",
        "",
        issue["body"].rstrip(),
        "",
    ])
out.parent.mkdir(parents=True, exist_ok=True) if out.parent != pathlib.Path("") else None
out.write_text("\n".join(lines).rstrip() + "\n")
print(f"Wrote {out}")
PY
fi

if [[ "$dry_run" != "1" ]]; then
  if ! command -v gh >/dev/null 2>&1; then
    echo "gh CLI is required. Install gh and run: gh auth login" >&2
    echo "Tip: use --dry-run or --markdown <file> without gh." >&2
    exit 1
  fi
  if [[ -z "$repo" ]]; then
    repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
  fi
fi

tmpdir=$(mktemp -d)
trap 'rm -rf "$tmpdir"' EXIT

python3 - "$manifest" "$tmpdir" <<'PY'
import json
import pathlib
import sys

manifest = pathlib.Path(sys.argv[1])
out = pathlib.Path(sys.argv[2])
issues = json.loads(manifest.read_text())
for i, issue in enumerate(issues, 1):
    (out / f"{i}.title").write_text(issue["title"])
    (out / f"{i}.body").write_text(issue["body"])
    (out / f"{i}.labels").write_text("\n".join(issue.get("labels", [])) + "\n")
PY

count=$(find "$tmpdir" -name '*.title' | wc -l | tr -d ' ')
for i in $(seq 1 "$count"); do
  title=$(cat "$tmpdir/$i.title")
  labels=()
  while IFS= read -r label; do
    [[ -n "$label" ]] && labels+=(--label "$label")
  done <"$tmpdir/$i.labels"
  if [[ "$dry_run" == "1" ]]; then
    printf 'DRY RUN: gh issue create'
    [[ -n "$repo" ]] && printf ' --repo %q' "$repo"
    printf ' --title %q --body-file %q' "$title" "$tmpdir/$i.body"
    printf ' %q' "${labels[@]}"
    printf '\n'
  else
    echo "Creating issue: $title"
    gh issue create --repo "$repo" --title "$title" --body-file "$tmpdir/$i.body" "${labels[@]}"
  fi
done
