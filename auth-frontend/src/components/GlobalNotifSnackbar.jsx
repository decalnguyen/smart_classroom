import { Snackbar, Alert, Typography } from '@mui/material'
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive'
import AssignmentLateIcon from '@mui/icons-material/AssignmentLate'
import { useRealtime } from '../context/RealtimeContext'

const TITLE_MAP = {
  leave: { label: 'Đơn xin nghỉ mới', icon: <AssignmentLateIcon fontSize="small" />, severity: 'warning' },
  attendance: { label: 'Điểm danh', icon: <NotificationsActiveIcon fontSize="small" />, severity: 'info' },
}

export default function GlobalNotifSnackbar() {
  const { latestNotif, dismissNotif } = useRealtime()
  if (!latestNotif) return null

  const meta = TITLE_MAP[latestNotif.title] || { label: latestNotif.title || 'Thông báo', icon: <NotificationsActiveIcon fontSize="small" />, severity: 'info' }

  return (
    <Snackbar
      open={!!latestNotif}
      autoHideDuration={6000}
      onClose={dismissNotif}
      anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      sx={{ mt: 7 }}
    >
      <Alert
        severity={meta.severity}
        icon={meta.icon}
        onClose={dismissNotif}
        variant="filled"
        sx={{ minWidth: 280, maxWidth: 400 }}
      >
        <Typography variant="subtitle2" fontWeight={700}>{meta.label}</Typography>
        <Typography variant="body2">{latestNotif.message}</Typography>
      </Alert>
    </Snackbar>
  )
}
