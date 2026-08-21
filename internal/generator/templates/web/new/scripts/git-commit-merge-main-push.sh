#!/usr/bin/env bash

set -euo pipefail

repo_path="$(pwd)"
status="FAIL"
reason=""
local_sync_result="not executed"
upstream_result="not executed"
push_result="not executed"

print_report() {
  echo "=== Git Merge And Push Report ==="
  echo "Repository: ${repo_path}"
  echo "Local sync: ${local_sync_result}"
  echo "Upstream check: ${upstream_result}"
  echo "Push: ${push_result}"
  if [ -n "${reason}" ]; then
    echo "Reason: ${reason}"
  fi
  echo "Result: ${status}"
}

on_exit() {
  local rc=$?
  if [ "${rc}" -ne 0 ] && [ -z "${reason}" ]; then
    reason="git operation failed (check git output above)"
  fi
  print_report
}

trap on_exit EXIT

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  reason="current directory is not a git repository"
  exit 1
fi

if ! git show-ref --verify --quiet refs/heads/main; then
  reason="local branch 'main' does not exist"
  exit 1
fi

local_sync_script="$(dirname "${BASH_SOURCE[0]}")/git_merge_to_main_report.sh"
if ! "${local_sync_script}"; then
  local_sync_result="failed"
  reason="local commit/merge step failed"
  exit 1
fi
local_sync_result="completed"

if ! git checkout main >/dev/null 2>&1; then
  reason="failed to checkout main before push"
  exit 1
fi

if ! git remote get-url origin >/dev/null 2>&1; then
  reason="origin remote is not configured"
  exit 1
fi

if ! git fetch --quiet origin; then
  reason="failed to fetch origin"
  exit 1
fi

if upstream_ref="$(git rev-parse --abbrev-ref --symbolic-full-name @{u} 2>/dev/null)"; then
  counts="$(git rev-list --left-right --count "${upstream_ref}...HEAD")"
  behind=0
  ahead=0
  IFS=$' \t' read -r behind ahead <<< "$counts"
  behind="${behind:-0}"
  ahead="${ahead:-0}"
  upstream_result="${upstream_ref} (behind:${behind} ahead:${ahead})"
  if [ "${behind}" -gt 0 ]; then
    reason="main is behind ${upstream_ref} by ${behind} commit(s); pull first"
    exit 1
  fi
  git push
else
  if git show-ref --verify --quiet refs/remotes/origin/main; then
    counts="$(git rev-list --left-right --count "origin/main...HEAD")"
    behind=0
    ahead=0
    IFS=$' \t' read -r behind ahead <<< "$counts"
    behind="${behind:-0}"
    ahead="${ahead:-0}"
    upstream_result="no-upstream, origin/main (behind:${behind} ahead:${ahead})"
    if [ "${behind}" -gt 0 ]; then
      reason="main is behind origin/main by ${behind} commit(s); pull first"
      exit 1
    fi
  else
    upstream_result="no-upstream, origin/main not found (first push)"
  fi
  git push -u origin main
fi

push_result="pushed main to origin"
status="PASS"
