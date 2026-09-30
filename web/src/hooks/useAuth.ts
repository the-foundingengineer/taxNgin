import { createContext, useContext, useState, useCallback } from 'react'
import { getToken, clearToken, setToken } from '@/lib/api/client'

interface AuthContextValue {
  token: string | null
  isAuthenticated: boolean
  login: (token: string) => void
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue>({
  token: null,
  isAuthenticated: false,
  login: () => {},
  logout: () => {},
})

export function useAuth() {
  return useContext(AuthContext)
}

export function useAuthState(): AuthContextValue {
  const [token, setTokenState] = useState<string | null>(() => getToken())

  const login = useCallback((t: string) => {
    setToken(t)
    setTokenState(t)
  }, [])

  const logout = useCallback(() => {
    clearToken()
    setTokenState(null)
  }, [])

  return { token, isAuthenticated: token !== null, login, logout }
}
