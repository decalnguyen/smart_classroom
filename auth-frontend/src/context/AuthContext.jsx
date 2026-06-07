import { createContext, useContext, useState, useCallback, useMemo } from 'react'
import { authApi } from '../api/client'

const AuthContext = createContext(null)

function loadUser() {
  try {
    const raw = localStorage.getItem('user')
    return raw ? JSON.parse(raw) : null
  } catch {
    return null
  }
}

export function AuthProvider({ children }) {
  const [user, setUser] = useState(loadUser)

  const login = useCallback(async (username, password) => {
    const { data } = await authApi.login(username, password)
    const u = { account_id: data.account_id, username: data.username, role: data.role }
    localStorage.setItem('token', data.token)
    localStorage.setItem('user', JSON.stringify(u))
    setUser(u)
    return u
  }, [])

  const signup = useCallback(async (username, password, role) => {
    await authApi.signup(username, password, role)
  }, [])

  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } catch {
      /* ignore network errors on logout */
    }
    localStorage.removeItem('token')
    localStorage.removeItem('user')
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({
      user,
      role: user?.role || null,
      isAuthenticated: !!user && !!localStorage.getItem('token'),
      login,
      signup,
      logout,
    }),
    [user, login, signup, logout]
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
