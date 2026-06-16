import DashboardIcon from '@mui/icons-material/Dashboard'
import SensorsIcon from '@mui/icons-material/Sensors'
import FactCheckIcon from '@mui/icons-material/FactCheck'
import HowToRegIcon from '@mui/icons-material/HowToReg'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'
import NotificationsIcon from '@mui/icons-material/Notifications'
import AssessmentIcon from '@mui/icons-material/Assessment'
import FaceRetouchingNaturalIcon from '@mui/icons-material/FaceRetouchingNatural'
import FaceIcon from '@mui/icons-material/Face'
import EventBusyIcon from '@mui/icons-material/EventBusy'
import HistoryIcon from '@mui/icons-material/History'
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings'

// Single source of truth for navigation; `roles` undefined = all roles.
export const navItems = [
  { label: 'Tổng quan', path: '/', icon: <DashboardIcon /> },
  { label: 'Cảm biến & Thiết bị', path: '/sensors', icon: <SensorsIcon /> },
  { label: 'Điểm danh', path: '/attendance', icon: <FactCheckIcon />, roles: ['admin', 'teacher'] },
  { label: 'Điểm danh của tôi', path: '/my-attendance', icon: <HowToRegIcon />, roles: ['student'] },
  { label: 'Lịch học', path: '/schedule', icon: <CalendarMonthIcon /> },
  { label: 'Đơn xin nghỉ', path: '/leaves', icon: <EventBusyIcon /> },
  { label: 'Thông báo', path: '/notifications', icon: <NotificationsIcon /> },
  { label: 'Báo cáo', path: '/reports', icon: <AssessmentIcon />, roles: ['admin', 'teacher'] },
  { label: 'Duyệt nhận diện', path: '/review', icon: <FaceRetouchingNaturalIcon />, roles: ['admin', 'teacher'] },
  { label: 'Đăng ký khuôn mặt', path: '/enrollment', icon: <FaceIcon />, roles: ['admin'] },
  { label: 'Nhật ký', path: '/audit', icon: <HistoryIcon />, roles: ['admin'] },
  { label: 'Quản trị', path: '/admin', icon: <AdminPanelSettingsIcon />, roles: ['admin'] },
]

export function visibleNavItems(role) {
  return navItems.filter((i) => !i.roles || i.roles.includes(role))
}
