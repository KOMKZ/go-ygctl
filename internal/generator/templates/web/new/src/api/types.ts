/**
 * 后端响应 Envelope（成功/业务错误）
 */
export interface ApiResponse<T = unknown> {
  code: number
  msg?: string
  message?: string
  data?: T
  error?: string
}

/**
 * 统一分页模型（前端使用）
 */
export interface PaginatedResult<T> {
  list: T[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

/**
 * 后端分页元数据（httpx/types.PageMeta：嵌套在 data.page_meta 下）
 */
export interface BackendPageMeta {
  total: number
  size: number
  current: number
  pages: number
}

/**
 * 后端分页响应（hrise-admin-api：data = { page_meta, records }，page_meta 为嵌套结构）
 */
export interface BackendPageResponse<T> {
  page_meta: BackendPageMeta
  records: T[]
}

/**
 * 分页查询参数（前端使用）
 */
export interface PaginationParams {
  page: number
  pageSize: number
}

/**
 * 登录请求
 */
export interface LoginRequest {
  email?: string
  phone?: string
  login_type?: 'email' | 'phone'
  verification_code?: string
  password: string
  captcha_verify_param?: string
}

/**
 * 登录响应
 */
export interface LoginResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  refresh_after_seconds?: number
  token_type: string
  user_id: number
  username: string
}

/**
 * Token 刷新请求
 */
export interface RefreshTokenRequest {
  refresh_token: string
}

/**
 * Token 刷新响应
 */
export interface RefreshTokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  refresh_after_seconds?: number
  token_type: string
}

export interface SendVerificationCodeRequest {
  type: 'email' | 'phone'
  target: string
  trigger_code?: string
}

export interface SendVerificationCodeResponse {
  ttl_seconds: number
  next_send_after_seconds: number
  debug_code?: string
}

/**
 * 用户 UI 权限载荷（来自 /api/admin/profile/permissions）
 */
export interface ProfileUIAuth {
  visible_routes: string[]
  visible_menus: string[]
  element_permissions: Record<string, string[]>
  page_actions?: Record<string, string[]>
}

export interface ProfilePermissionsDTO {
  permissions: string[]
  super_admin: boolean
  ui_auth: ProfileUIAuth
}

/**
 * 后端 AdminItem 原始形态（service 层结构体无 json tag，序列化为 PascalCase）
 */
export interface BackendAdminItem {
  ID: number
  Username: string
  RealName: string
  Email: string
  Phone: string
  Avatar: string
  AvatarStorageID: string
  Role: number
  Status: number
  LastLoginAt: string | null
  CreatedAt: string
  UpdatedAt: string
}

/**
 * 前端统一 admin 记录模型（snake_case，与 defs DSL 字段名一致）
 */
export interface AdminRecord {
  id: number
  username: string
  real_name: string
  email: string
  phone: string
  avatar: string
  avatar_storage_id: string
  role: number
  status: number
  last_login_at: string | null
  created_at: string
  updated_at: string
}

/**
 * admin 表单模型（新建：含 password；编辑：无 password —— Update 端点不接收密码）
 */
export interface AdminFormModel {
  username: string
  password?: string
  real_name: string
  email: string
  phone: string
  avatar: string
  avatar_storage_id: string
  role: number
  status: number
}
