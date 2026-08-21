import { ref } from 'vue'
import type { HttpClient } from '@rong/admin-ui/app-request'
import type { PermissionServiceInstance, UIAuthPayload } from '@rong/admin-ui/app-permission'
import { createProfileApi } from '@/api/profile'
import { parseErrorMessage } from '@/api/adapter'

const uiAuth = ref<UIAuthPayload | null>(null)
const initialized = ref(false)

let inflight: Promise<void> | null = null

const DEFAULT_RETRY_DELAY_MS = 300

type ProfilePermissionsApi = Pick<ReturnType<typeof createProfileApi>, 'getPermissions'>

interface PermissionInitOptions {
  profileApi?: ProfilePermissionsApi
  retryDelayMs?: number
}

export function createPermissionInit(
  httpClient: HttpClient,
  permissionService: PermissionServiceInstance,
  options: PermissionInitOptions = {},
) {
  const profileApi = options.profileApi ?? createProfileApi(httpClient)
  const retryDelayMs = options.retryDelayMs ?? DEFAULT_RETRY_DELAY_MS

  async function init(force = false): Promise<void> {
    if (initialized.value && !force) return
    if (inflight) return inflight

    inflight = (async () => {
      try {
        const payload = await getPermissionsWithRetry(profileApi, retryDelayMs)

        permissionService.setPermissions(
          payload.permissions.map((code) => ({ action: code, scope: 'route' as const })),
        )
        permissionService.setSuperAdmin(payload.super_admin)

        uiAuth.value = payload.ui_auth ?? null
        initialized.value = true
      } catch (error) {
        permissionService.clear()
        uiAuth.value = null
        initialized.value = false
        throw new Error(`权限初始化失败：${parseErrorMessage(error)}`)
      }
    })()

    try {
      await inflight
    } finally {
      inflight = null
    }
  }

  function reset(): void {
    permissionService.clear()
    uiAuth.value = null
    initialized.value = false
  }

  return {
    init,
    reset,
    uiAuth,
    initialized,
  }
}

export function usePermissionUIAuth() {
  return uiAuth
}

async function getPermissionsWithRetry(profileApi: ProfilePermissionsApi, retryDelayMs: number) {
  let lastError: unknown = null
  for (let attempt = 0; attempt < 2; attempt += 1) {
    try {
      return await profileApi.getPermissions()
    } catch (error) {
      lastError = error
      if (attempt === 0) {
        await sleep(retryDelayMs)
      }
    }
  }

  throw lastError ?? new Error('权限初始化失败')
}

function sleep(ms: number): Promise<void> {
  if (ms <= 0) {
    return Promise.resolve()
  }
  return new Promise((resolve) => setTimeout(resolve, ms))
}
