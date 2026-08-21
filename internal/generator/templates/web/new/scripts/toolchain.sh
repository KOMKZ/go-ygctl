#!/usr/bin/env bash
# Managed by ygctl (web new / web ensure); regenerate to update.

load_toolchain() {
  if command -v pnpm >/dev/null 2>&1; then
    return 0
  fi

  local env_file="${HOME}/.config/yogan-dev-env.sh"
  if [[ -f "${env_file}" ]]; then
    # shellcheck disable=SC1091
    . "${env_file}"
  fi

  if ! command -v pnpm >/dev/null 2>&1 && [[ -s "${HOME}/.nvm/nvm.sh" ]]; then
    export NVM_DIR="${HOME}/.nvm"
    # shellcheck disable=SC1091
    . "${NVM_DIR}/nvm.sh"
    nvm use default >/dev/null 2>&1 || nvm use 22 >/dev/null 2>&1 || true
  fi

  if ! command -v pnpm >/dev/null 2>&1; then
    printf "pnpm not found. Install Node/pnpm or create %s\n" "${env_file}" >&2
    exit 127
  fi
}
