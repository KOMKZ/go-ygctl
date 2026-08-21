import type { HttpClient } from '@rong/admin-ui/app-request'
import type {
  ApiResponse,
  LoginRequest,
  LoginResponse,
  RefreshTokenRequest,
  RefreshTokenResponse,
  SendVerificationCodeRequest,
  SendVerificationCodeResponse,
} from './types'
import { extractData } from './adapter'

export function createAuthApi(http: HttpClient) {
  return {
    /**
     * 登录
     */
    async login(params: LoginRequest): Promise<ApiResponse<LoginResponse>> {
      return http.post<LoginResponse>('/api/auth/login', params)
    },

    /**
     * 刷新 Token
     */
    async refreshToken(params: RefreshTokenRequest): Promise<ApiResponse<RefreshTokenResponse>> {
      return http.post<RefreshTokenResponse>('/api/auth/refresh', params)
    },

    /**
     * 发送验证码（手机/邮箱登录）
     */
    async sendVerificationCode(
      params: SendVerificationCodeRequest,
    ): Promise<SendVerificationCodeResponse> {
      const response = await http.post<SendVerificationCodeResponse>(
        '/api/auth/verification-code/send',
        params,
      )
      return extractData(response)
    },
  }
}

export type AuthApi = ReturnType<typeof createAuthApi>
