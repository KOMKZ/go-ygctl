#!/usr/bin/env bash
set -euo pipefail

repo_path="$(pwd)"
status="FAIL"
reason=""
current_branch=""
commit_result="not executed"
merge_result="not executed"
final_branch=""

print_report() {
  printf '%s\n' '=== Local Git Merge Report ==='
  printf 'Repository: %s\n' "${repo_path}"
  if [[ -n "${current_branch}" ]]; then
    printf 'Source branch: %s\n' "${current_branch}"
  else
    printf '%s\n' 'Source branch: N/A'
  fi
  printf 'Commit: %s\n' "${commit_result}"
  printf 'Merge: %s\n' "${merge_result}"
  if [[ -n "${final_branch}" ]]; then
    printf 'Current branch now: %s\n' "${final_branch}"
  fi
  if [[ -n "${reason}" ]]; then
    printf 'Reason: %s\n' "${reason}"
  fi
  printf 'Result: %s\n' "${status}"
}

fail() {
  reason="$1"
  print_report
  exit 1
}

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  fail "current directory is not a git repository"
fi

repo_path="$(git rev-parse --show-toplevel)"
current_branch="$(git branch --show-current)"
if [[ -z "${current_branch}" ]]; then
  fail "detached HEAD is not supported"
fi

if [[ -n "$(git status --porcelain)" ]]; then
  git add -A
  if git diff --cached --quiet; then
    commit_result="no committable changes after staging"
  else
    git commit -m "chore: checkpoint workspace before merge to main" >/dev/null
    commit_result="created commit $(git rev-parse --short HEAD) on ${current_branch}"
  fi
else
  commit_result="no workspace changes to commit"
fi

if [[ "${current_branch}" == "main" ]]; then
  merge_result="already on main, merge skipped"
  final_branch="main"
  status="PASS"
  print_report
  exit 0
fi

if ! git show-ref --verify --quiet refs/heads/main; then
  fail "local branch 'main' does not exist"
fi

if ! git checkout main >/dev/null 2>&1; then
  fail "failed to checkout main"
fi

if ! git merge --no-ff "${current_branch}" -m "merge: integrate ${current_branch} into main" >/dev/null 2>&1; then
  final_branch="main"
  fail "merge conflict when merging '${current_branch}' into 'main'"
fi

merge_result="merged ${current_branch} into main at $(git rev-parse --short HEAD)"

if ! git checkout "${current_branch}" >/dev/null 2>&1; then
  final_branch="main"
  fail "merged into main, but failed to switch back to ${current_branch}"
fi

final_branch="$(git branch --show-current)"
status="PASS"
print_report
