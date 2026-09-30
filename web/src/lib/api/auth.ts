import { api, setToken, clearToken } from './client'

export interface RegisterRequest {
  nin: string
  phone: string
  email?: string
  password: string
}

export interface LoginRequest {
  nin: string
  password: string
}

export interface RecoveryInitResponse {
  session_id: string
  otp: string   // dev only — in production this is delivered out-of-band
  message?: string
}

export const authApi = {
  register: (req: RegisterRequest) =>
    api.post<{ account_id: string }>('/accounts', req),

  login: async (req: LoginRequest): Promise<string> => {
    const res = await api.post<{ token: string }>('/sessions', req)
    setToken(res.token)
    return res.token
  },

  logout: async (): Promise<void> => {
    await api.delete<void>('/sessions').catch(() => {})
    clearToken()
  },

  initiateRecovery: (nin: string, method = 'OTP_SMS') =>
    api.post<RecoveryInitResponse>('/recovery', { nin, method }),

  submitOTP: (sessionId: string, otp: string) =>
    api.post<{ state: string }>(`/recovery/${sessionId}/otp`, { otp }),

  completeRecovery: (sessionId: string, newPassword: string) =>
    api.post<{ state: string }>(`/recovery/${sessionId}/complete`, { new_password: newPassword }),
}
