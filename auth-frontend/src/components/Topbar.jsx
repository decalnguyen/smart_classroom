import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  AppBar,
  Toolbar,
  IconButton,
  Typography,
  Badge,
  Box,
  Avatar,
  Menu,
  MenuItem,
  Chip,
  Tooltip,
  ListItemIcon,
  Popover,
  List,
  ListItem,
  ListItemText,
  Divider,
  Button,
  Stack,
} from '@mui/material'
import MenuIcon from '@mui/icons-material/Menu'
import NotificationsIcon from '@mui/icons-material/Notifications'
import LogoutIcon from '@mui/icons-material/Logout'
import CircleIcon from '@mui/icons-material/Circle'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import DoneAllIcon from '@mui/icons-material/DoneAll'
import { useAuth } from '../context/AuthContext'
import { useRealtime } from '../context/RealtimeContext'
import { useColorMode } from '../context/ColorModeContext'

const roleLabel = { admin: 'Quản trị viên', teacher: 'Giáo viên', student: 'Học sinh' }
const NOTIF_LABELS = { alert: 'Cảnh báo an toàn', leave: 'Đơn xin nghỉ', attendance: 'Điểm danh' }

export default function Topbar({ onMenuClick }) {
  const navigate = useNavigate()
  const { user, role, logout } = useAuth()
  const { unreadCount, wsStatus, notifications, markRead } = useRealtime()
  const { mode, toggle } = useColorMode()
  const [anchorEl, setAnchorEl] = useState(null)
  const [notifEl, setNotifEl] = useState(null)

  const handleLogout = async () => {
    setAnchorEl(null)
    await logout()
    navigate('/login', { replace: true })
  }

  const online = wsStatus === 'open'
  const recent = notifications.slice(0, 6)

  return (
    <AppBar
      position="fixed"
      elevation={0}
      sx={{
        zIndex: (t) => t.zIndex.drawer + 1,
        bgcolor: 'background.paper',
        color: 'text.primary',
        borderBottom: '1px solid',
        borderColor: 'divider',
        backdropFilter: 'blur(6px)',
      }}
    >
      <Toolbar sx={{ gap: 1 }}>
        <IconButton edge="start" onClick={onMenuClick} sx={{ display: { md: 'none' } }}>
          <MenuIcon />
        </IconButton>
        <Typography variant="h6" sx={{ flexGrow: 1 }} fontWeight={700} noWrap>
          Hệ thống lớp học thông minh
        </Typography>

        <Tooltip title={online ? 'Realtime đang kết nối' : 'Mất kết nối realtime'}>
          <Chip
            size="small"
            icon={<CircleIcon sx={{ fontSize: 11 }} />}
            label={online ? 'Trực tuyến' : 'Ngoại tuyến'}
            color={online ? 'success' : 'default'}
            variant="outlined"
            sx={{ display: { xs: 'none', sm: 'inline-flex' } }}
          />
        </Tooltip>

        <Tooltip title={mode === 'dark' ? 'Chế độ sáng' : 'Chế độ tối'}>
          <IconButton onClick={toggle}>{mode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}</IconButton>
        </Tooltip>

        <Tooltip title="Thông báo">
          <IconButton onClick={(e) => setNotifEl(e.currentTarget)} aria-label={`Thông báo, ${unreadCount} chưa đọc`}>
            <Badge badgeContent={unreadCount} color="error">
              <NotificationsIcon />
            </Badge>
          </IconButton>
        </Tooltip>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Box sx={{ textAlign: 'right', display: { xs: 'none', sm: 'block' } }}>
            <Typography variant="body2" fontWeight={700} lineHeight={1.1}>
              {user?.username}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {roleLabel[role] || role}
            </Typography>
          </Box>
          <IconButton onClick={(e) => setAnchorEl(e.currentTarget)}>
            <Avatar sx={{ bgcolor: 'primary.main', width: 36, height: 36 }}>
              {(user?.username || '?').charAt(0).toUpperCase()}
            </Avatar>
          </IconButton>
        </Box>

        {/* Notifications popover */}
        <Popover
          open={!!notifEl}
          anchorEl={notifEl}
          onClose={() => setNotifEl(null)}
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          transformOrigin={{ vertical: 'top', horizontal: 'right' }}
          slotProps={{ paper: { sx: { width: 360, maxWidth: '90vw' } } }}
        >
          <Stack direction="row" alignItems="center" justifyContent="space-between" sx={{ px: 2, py: 1.5 }}>
            <Typography variant="subtitle1" fontWeight={700}>Thông báo</Typography>
            <Chip size="small" label={`${unreadCount} mới`} color={unreadCount ? 'error' : 'default'} />
          </Stack>
          <Divider />
          {recent.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
              Chưa có thông báo
            </Typography>
          ) : (
            <List dense sx={{ maxHeight: 360, overflow: 'auto', py: 0 }}>
              {recent.map((n) => (
                <ListItem
                  key={n.id}
                  divider
                  onClick={() => { if (!n.is_read) markRead(n.id) }}
                  sx={{ bgcolor: n.is_read ? 'transparent' : 'action.hover', cursor: 'pointer', alignItems: 'flex-start' }}
                >
                  <ListItemIcon sx={{ minWidth: 34, mt: 0.5 }}>
                    {n.title === 'alert' ? <WarningAmberIcon color="error" fontSize="small" /> : <InfoOutlinedIcon color="info" fontSize="small" />}
                  </ListItemIcon>
                  <ListItemText
                    primary={NOTIF_LABELS[n.title] || n.title || 'Thông báo'}
                    secondary={n.message}
                    primaryTypographyProps={{ fontWeight: n.is_read ? 500 : 700, fontSize: 14 }}
                    secondaryTypographyProps={{ noWrap: false, sx: { fontSize: 12.5 } }}
                  />
                </ListItem>
              ))}
            </List>
          )}
          <Divider />
          <Button
            fullWidth
            startIcon={<DoneAllIcon />}
            onClick={() => { setNotifEl(null); navigate('/notifications') }}
            sx={{ py: 1.25, borderRadius: 0 }}
          >
            Xem tất cả thông báo
          </Button>
        </Popover>

        {/* Profile menu */}
        <Menu anchorEl={anchorEl} open={!!anchorEl} onClose={() => setAnchorEl(null)}>
          <Box sx={{ px: 2, py: 1 }}>
            <Typography variant="subtitle2" fontWeight={700}>{user?.username}</Typography>
            <Typography variant="caption" color="text.secondary">{roleLabel[role] || role}</Typography>
          </Box>
          <Divider />
          <MenuItem onClick={() => { setAnchorEl(null); toggle() }}>
            <ListItemIcon>{mode === 'dark' ? <LightModeIcon fontSize="small" /> : <DarkModeIcon fontSize="small" />}</ListItemIcon>
            {mode === 'dark' ? 'Chế độ sáng' : 'Chế độ tối'}
          </MenuItem>
          <MenuItem onClick={handleLogout}>
            <ListItemIcon><LogoutIcon fontSize="small" /></ListItemIcon>
            Đăng xuất
          </MenuItem>
        </Menu>
      </Toolbar>
    </AppBar>
  )
}
