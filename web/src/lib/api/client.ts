import type { ApiErrorBody } from '@/types/api'

const SESSION_KEY = 'tx_token'

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

export function getToken(): string | null {
  return localStorage.getItem(SESSION_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(SESSION_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(SESSION_KEY)
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = getToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
  }
  // Allow callers to override Content-Type (e.g. for raw CSV body)
  if (init.headers) {
    Object.assign(headers, init.headers)
  }

  const res = await fetch(`/v1${path}`, { ...init, headers })

  if (res.status === 204) return undefined as T

  if (!res.ok) {
    if (res.status === 401) {
      clearToken()
      window.location.href = '/login'
    }
    let msg = res.statusText
    try {
      const body: ApiErrorBody = await res.json()
      msg = body.error ?? msg
    } catch {
      // ignore parse error
    }
    throw new ApiError(res.status, msg)
  }

  return res.json() as Promise<T>
}

export const api = {
  get:  <T>(path: string) =>
    request<T>(path, { method: 'GET' }),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined }),

  delete: <T>(path: string) =>
    request<T>(path, { method: 'DELETE' }),

  postRaw: <T>(path: string, body: BodyInit, contentType: string) =>
    request<T>(path, {
      method: 'POST',
      body,
      headers: { 'Content-Type': contentType },
    }),
}
