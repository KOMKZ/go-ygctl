import type {
  ApiResponse,
  BackendAdminItem,
  BackendPageResponse,
  PaginatedResult,
  PaginationParams,
  AdminRecord,
} from './types'

/**
 * 成功码
 */
export const SUCCESS_CODE = 0

/**
 * 未授权码
 */
export const UNAUTHORIZED_CODE = 401

/**
 * 判断响应是否成功
 */
export function isSuccess<T>(response: ApiResponse<T>): boolean {
  return response.code === SUCCESS_CODE
}

/**
 * 判断响应是否为未授权
 */
export function isUnauthorized<T>(response: ApiResponse<T>): boolean {
  return response.code === UNAUTHORIZED_CODE
}

/**
 * 解析错误消息（兼容 msg/message 双字段，支持 unknown 类型）
 * 优先级：msg > message > error > 默认
 * 空字符串会跳过，继续检查下一个字段
 */
export function parseErrorMessage(error: unknown): string {
  if (error === null || error === undefined) {
    return '未知错误'
  }

  if (typeof error === 'string') {
    return error || '未知错误'
  }

  if (error instanceof Error) {
    return error.message || '未知错误'
  }

  if (typeof error === 'object') {
    const e = error as Record<string, unknown>
    // 优先从 responseBody 提取（HTTP 客户端错误）
    if (e.responseBody && typeof e.responseBody === 'object') {
      const rb = e.responseBody as Record<string, unknown>
      if (typeof rb.msg === 'string' && rb.msg) return rb.msg
      if (typeof rb.message === 'string' && rb.message) return rb.message
      if (typeof rb.error === 'string' && rb.error) return rb.error
    }
    if (typeof e.msg === 'string' && e.msg) return e.msg
    if (typeof e.message === 'string' && e.message) return e.message
    if (typeof e.error === 'string' && e.error) return e.error
  }

  return '未知错误'
}

/**
 * 将前端分页参数转换为后端 Admin 模块分页参数
 */
export function toBackendPageParams(params: PaginationParams): { current: number; size: number } {
  return {
    current: params.page,
    size: params.pageSize,
  }
}

/**
 * 将后端分页响应转换为前端统一分页模型
 * 后端分页元数据嵌套在 page_meta 下（httpx/types.PageMeta）
 */
export function fromBackendPageResponse<T>(response: BackendPageResponse<T>): PaginatedResult<T> {
  return {
    list: response.records,
    total: response.page_meta.total,
    page: response.page_meta.current,
    pageSize: response.page_meta.size,
    totalPages: response.page_meta.pages,
  }
}

/**
 * 将后端 PascalCase 记录归一化为前端 snake_case 模型（adapter 层吸收后端序列化差异）
 */
export function normalizeAdminRecord(raw: BackendAdminItem): AdminRecord {
  return {
    id: raw.ID,
    username: raw.Username,
    real_name: raw.RealName,
    email: raw.Email,
    phone: raw.Phone,
    avatar: raw.Avatar,
    avatar_storage_id: raw.AvatarStorageID,
    role: raw.Role,
    status: raw.Status,
    last_login_at: raw.LastLoginAt,
    created_at: raw.CreatedAt,
    updated_at: raw.UpdatedAt,
  }
}

/**
 * 断言响应成功，否则抛出错误
 */
export function assertSuccess<T>(response: ApiResponse<T>): asserts response is ApiResponse<T> & { data: T } {
  if (!isSuccess(response)) {
    throw new Error(parseErrorMessage(response))
  }
}

/**
 * 提取响应数据，失败时抛出错误
 */
export function extractData<T>(response: ApiResponse<T>): T {
  assertSuccess(response)
  return response.data as T
}

/**
 * 处理 API 调用结果，返回统一格式
 */
export interface ApiResult<T> {
  success: boolean
  data?: T
  error?: string
  code: number
}

export function wrapApiResult<T>(response: ApiResponse<T>): ApiResult<T> {
  if (isSuccess(response)) {
    return {
      success: true,
      data: response.data,
      code: response.code,
    }
  }
  return {
    success: false,
    error: parseErrorMessage(response),
    code: response.code,
  }
}
