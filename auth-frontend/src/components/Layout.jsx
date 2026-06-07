import { useState } from 'react'
import { Outlet } from 'react-router-dom'
import { Box, Toolbar } from '@mui/material'
import Sidebar, { DRAWER_WIDTH } from './Sidebar'
import Topbar from './Topbar'
import AlarmBanner from './AlarmBanner'

export default function Layout() {
  const [mobileOpen, setMobileOpen] = useState(false)

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: 'background.default' }}>
      <Topbar onMenuClick={() => setMobileOpen((v) => !v)} />
      <Sidebar mobileOpen={mobileOpen} onClose={() => setMobileOpen(false)} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          width: { md: `calc(100% - ${DRAWER_WIDTH}px)` },
          p: { xs: 2, md: 3 },
        }}
      >
        <Toolbar />
        <AlarmBanner />
        <Outlet />
      </Box>
    </Box>
  )
}
