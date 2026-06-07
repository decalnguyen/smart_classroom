import { createTheme } from '@mui/material/styles'

// Build a theme for the given color mode ('light' | 'dark').
export function getTheme(mode = 'light') {
  const isDark = mode === 'dark'
  return createTheme({
    palette: {
      mode,
      primary: { main: '#2563eb' },
      secondary: { main: '#0891b2' },
      success: { main: '#16a34a' },
      warning: { main: '#ea580c' },
      error: { main: '#dc2626' },
      info: { main: '#0284c7' },
      background: isDark
        ? { default: '#0b1220', paper: '#111c34' }
        : { default: '#f1f5f9', paper: '#ffffff' },
      divider: isDark ? 'rgba(148,163,184,0.16)' : 'rgba(15,23,42,0.08)',
      text: isDark
        ? { primary: '#e2e8f0', secondary: '#94a3b8' }
        : { primary: '#0f172a', secondary: '#64748b' },
    },
    typography: {
      fontFamily: ['Inter', 'Roboto', 'Helvetica', 'Arial', 'sans-serif'].join(','),
      h4: { fontWeight: 800, letterSpacing: -0.5 },
      h5: { fontWeight: 800 },
      h6: { fontWeight: 700 },
      subtitle2: { fontWeight: 600 },
      button: { fontWeight: 600 },
    },
    shape: { borderRadius: 14 },
    components: {
      MuiCard: {
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${isDark ? 'rgba(148,163,184,0.12)' : 'rgba(15,23,42,0.06)'}`,
            boxShadow: isDark
              ? '0 1px 2px rgba(0,0,0,0.4)'
              : '0 1px 2px rgba(16,24,40,0.06), 0 1px 3px rgba(16,24,40,0.04)',
          },
        },
      },
      MuiButton: { styleOverrides: { root: { textTransform: 'none', borderRadius: 10 } }, defaultProps: { disableElevation: true } },
      MuiChip: { styleOverrides: { root: { fontWeight: 600 } } },
      MuiAppBar: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiTableCell: { styleOverrides: { head: { fontWeight: 700 } } },
      MuiListItemButton: { styleOverrides: { root: { borderRadius: 10 } } },
    },
  })
}

export default getTheme('light')
