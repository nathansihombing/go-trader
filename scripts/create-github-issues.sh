#!/usr/bin/env bash
# Create GitHub issues from docs/github_issues.json using GitHub CLI.
set -euo pipefail

manifest="${1:-docs/github_issues.json}"
repo="${GH_REPO:-}"

if [[ ! -f "$manifest" ]]; then
  echo "manifest not found: $manifest" >&2
  exit 1
fi
if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required. Install gh and run: gh auth login" >&2
  exit 1
fi
if [[ -z "$repo" ]]; then
  repo=$(gh repo view --json nameWithOwner --jq .nameWithOwner)
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
    (out / f"{i}.labels").write_text("\n".join(issue.get("labels", [])))
print(len(issues))
PY

count=$(find "$tmpdir" -name '*.title' | wc -l | tr -d ' ')
for i in $(seq 1 "$count"); do
  title=$(cat "$tmpdir/$i.title")
  labels=()
  while IFS= read -r label; do
    [[ -n "$label" ]] && labels+=(--label "$label")
  done <"$tmpdir/$i.labels"
  echo "Creating issue: $title"
  gh issue create --repo "$repo" --title "$title" --body-file "$tmpdir/$i.body" "${labels[@]}"
done
