#!/usr/bin/env bash
# Managed by ygctl (web new / web ensure); regenerate to update.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/toolchain.sh"

command="${1:-}"
if [[ -z "$command" ]]; then
  printf "Usage: %s <command> [args]\n" "$(basename "$0")" >&2
  exit 1
fi
shift || true

ensure_runtime_dir() {
  local runtime_dir="$1"
  mkdir -p "$runtime_dir"
}

stop_pidfile() {
  local pid_file="$1"

  if [[ -f "$pid_file" ]]; then
    local pid
    pid="$(cat "$pid_file")"
    if [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1; then
      kill "$pid" >/dev/null 2>&1 || true
      sleep 1
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
    rm -f "$pid_file"
  fi
}

start_background_process() {
  local runtime_dir="$1"
  local pid_file="$2"
  local log_file="$3"
  local label="$4"
  local port="$5"
  shift 5

  mkdir -p "$runtime_dir"
  stop_pidfile "$pid_file"
  nohup setsid "$@" >"$log_file" 2>&1 </dev/null & echo $! >"$pid_file"
  sleep 1
  printf "Started %s server on :%s (pid: %s)\n" "$label" "$port" "$(cat "$pid_file")"
}

start_dev_bg() {
  local runtime_dir="$1"
  local pid_file="$2"
  local log_file="$3"
  local port="$4"
  load_toolchain
  start_background_process "$runtime_dir" "$pid_file" "$log_file" "dev" "$port" pnpm exec vite --host 127.0.0.1 --port "$port" --strictPort
}

start_preview_bg() {
  local runtime_dir="$1"
  local pid_file="$2"
  local log_file="$3"
  local port="$4"
  load_toolchain
  start_background_process "$runtime_dir" "$pid_file" "$log_file" "preview" "$port" pnpm exec vite preview --host 127.0.0.1 --port "$port" --strictPort
}

kill_ports() {
  local current_dir="$1"
  shift

  for port in "$@"; do
    local pids
    pids="$(lsof -ti tcp:"$port" 2>/dev/null || true)"

    for pid in $pids; do
      local cmd
      cmd="$(ps -o command= -p "$pid" 2>/dev/null || true)"
      if [[ "$cmd" == *"vite"* ]] || [[ "$cmd" == *"$current_dir"* ]]; then
        printf "Reclaiming port %s (pid: %s)\n" "$port" "$pid"
        kill "$pid" >/dev/null 2>&1 || true
      else
        printf "Skip port %s pid %s (non-vite process)\n" "$port" "$pid"
      fi
    done
  done
}

show_status() {
  local dev_pid_file="$1"
  local preview_pid_file="$2"

  if [[ -f "$dev_pid_file" ]]; then
    printf "dev pid: %s\n" "$(cat "$dev_pid_file")"
  else
    printf "dev pid: none\n"
  fi

  if [[ -f "$preview_pid_file" ]]; then
    printf "preview pid: %s\n" "$(cat "$preview_pid_file")"
  else
    printf "preview pid: none\n"
  fi
}

show_ports() {
  for port in "$@"; do
    local pids
    pids="$(lsof -ti tcp:"$port" 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
      printf "port %s in use by pids: %s\n" "$port" "$pids"
    else
      printf "port %s free\n" "$port"
    fi
  done
}

show_logs() {
  local dev_log_file="$1"
  local preview_log_file="$2"

  if [[ -f "$dev_log_file" ]]; then
    tail -n 120 "$dev_log_file"
  else
    printf "No dev log file\n"
  fi

  if [[ -f "$preview_log_file" ]]; then
    tail -n 120 "$preview_log_file"
  else
    printf "No preview log file\n"
  fi
}

clean_runtime() {
  local runtime_dir="$1"
  rm -rf "$runtime_dir"
}

case "$command" in
  ensure-runtime-dir)
    ensure_runtime_dir "$@"
    ;;
  stop-pidfile)
    stop_pidfile "$@"
    ;;
  start-dev-bg)
    start_dev_bg "$@"
    ;;
  start-preview-bg)
    start_preview_bg "$@"
    ;;
  kill-ports)
    kill_ports "$@"
    ;;
  show-status)
    show_status "$@"
    ;;
  show-ports)
    show_ports "$@"
    ;;
  show-logs)
    show_logs "$@"
    ;;
  clean-runtime)
    clean_runtime "$@"
    ;;
  *)
    printf "Unknown command: %s\n" "$command" >&2
    exit 1
    ;;
esac
