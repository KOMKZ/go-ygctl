import type { HttpClient } from '@rong/admin-ui/app-request'
import type { ProfilePermissionsDTO } from './types'
import { extractData } from './adapter'

export function createProfileApi(http: HttpClient) {
  return {
    /**
     * 获取当前用户权限载荷（路由/菜单/元素权限）
     */
    async getPermissions(): Promise<ProfilePermissionsDTO> {
      const response = await http.get<ProfilePermissionsDTO>('/api/admin/profile/permissions')
      return extractData(response)
    },
  }
}

export type ProfileApi = ReturnType<typeof createProfileApi>
