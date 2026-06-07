import { createContext, useContext, useMemo, useState, useCallback } from 'react'
import { ThemeProvider, CssBaseline } from '@mui/material'
import { getTheme } from '../theme'

const ColorModeContext = createContext({ mode: 'light', toggle: () => {} })

export function ColorModeProvider({ children }) {
  const [mode, setMode] = useState(() => localStorage.getItem('color-mode') || 'light')

  const toggle = useCallback(() => {
    setMode((m) => {
      const next = m === 'light' ? 'dark' : 'light'
      localStorage.setItem('color-mode', next)
      return next
    })
  }, [])

  const theme = useMemo(() => getTheme(mode), [mode])
  const value = useMemo(() => ({ mode, toggle }), [mode, toggle])

  return (
    <ColorModeContext.Provider value={value}>
      <ThemeProvider theme={theme}>
        <CssBaseline />
        {children}
      </ThemeProvider>
    </ColorModeContext.Provider>
  )
}

export const useColorMode = () => useContext(ColorModeContext)
