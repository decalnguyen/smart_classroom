import { useState, useMemo } from 'react'
import {
  Box,
  Card,
  CardContent,
  Stack,
  Typography,
  Chip,
  Button,
  IconButton,
  Tooltip,
  Snackbar,
  Alert,
  CircularProgress,
  Tabs,
  Tab,
  alpha,
} from '@mui/material'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import InfoOutlinedIcon from '@mui/icons-material/InfoOutlined'
import DoneAllIcon from '@mui/icons-material/DoneAll'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import NotificationsNoneIcon from '@mui/icons-material/NotificationsNone'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/vi'
import { useRealtime } from '../context/RealtimeContext'
import { notificationApi } from '../api/client'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'

dayjs.extend(relativeTime)
dayjs.locale('vi')

function relTime(value) {
  if (!value) return ''
  const d = dayjs(value)
  if (!d.isValid()) return ''
  return d.fromNow()
}

// Map raw notification.title keys to human-readable Vietnamese labels.
const TITLE_LABELS = {
  alert: 'Cảnh báo an toàn',
  leave: 'Đơn xin nghỉ',
  attendance: 'Điểm danh',
}
function titleLabel(t) {
  return TITLE_LABELS[t] || t || 'Thông báo'
}

export default function Notifications() {
  const { notifications, unreadCount, markRead, refresh } = useRealtime()
  const [busyId, setBusyId] = useState(null)
  const [markingAll, setMarkingAll] = useState(false)
  const [error, setError] = useState('')
  const [tab, setTab] = useState('all')

  const sorted = useMemo(
    () =>
      [...notifications].sort(
        (a, b) => dayjs(b.created_at).valueOf() - dayjs(a.created_at).valueOf()
      ),
    [notifications]
  )

  const alertCount = useMemo(
    () => notifications.filter((n) => n.title === 'alert').length,
    [notifications]
  )

  const filtered = useMemo(() => {
    if (tab === 'unread') return sorted.filter((n) => !n.is_read)
    if (tab === 'alert') return sorted.filter((n) => n.title === 'alert')
    return sorted
  }, [sorted, tab])

  const handleMarkRead = async (id) => {
    try {
      await markRead(id)
    } catch {
      setError('Không thể đánh dấu đã đọc.')
    }
  }

  const handleMarkAll = async () => {
    const unread = notifications.filter((n) => !n.is_read)
    if (unread.length === 0) return
    setMarkingAll(true)
    try {
      await Promise.all(unread.map((n) => markRead(n.id)))
    } catch {
      setError('Không thể đánh dấu tất cả đã đọc.')
    } finally {
      setMarkingAll(false)
    }
  }

  const handleRemove = async (id) => {
    setBusyId(id)
    try {
      await notificationApi.remove(id)
      await refresh()
    } catch {
      setError('Không thể xoá thông báo.')
    } finally {
      setBusyId(null)
    }
  }

  const subtitle =
    unreadCount > 0
      ? `Bạn có ${unreadCount} thông báo chưa đọc`
      : 'Bạn đã đọc hết thông báo'

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, maxWidth: 900, mx: 'auto' }}>
      <PageHeader
        title="Thông báo"
        subtitle={subtitle}
        action={
          <Button
            variant="outlined"
            size="small"
            startIcon={
              markingAll ? <CircularProgress size={16} /> : <DoneAllIcon />
            }
            onClick={handleMarkAll}
            disabled={unreadCount === 0 || markingAll}
          >
            Đánh dấu tất cả đã đọc
          </Button>
        }
      />

      <Tabs
        value={tab}
        onChange={(_e, v) => setTab(v)}
        sx={{ mb: 2, borderBottom: 1, borderColor: 'divider' }}
      >
        <Tab
          value="all"
          label={
            <Stack direction="row" spacing={1} alignItems="center">
              <span>Tất cả</span>
              <Chip label={notifications.length} size="small" />
            </Stack>
          }
        />
        <Tab
          value="unread"
          label={
            <Stack direction="row" spacing={1} alignItems="center">
              <span>Chưa đọc</span>
              <Chip
                label={unreadCount}
                size="small"
                color={unreadCount > 0 ? 'primary' : 'default'}
              />
            </Stack>
          }
        />
        <Tab
          value="alert"
          label={
            <Stack direction="row" spacing={1} alignItems="center">
              <span>Cảnh báo</span>
              <Chip
                label={alertCount}
                size="small"
                color={alertCount > 0 ? 'error' : 'default'}
              />
            </Stack>
          }
        />
      </Tabs>

      {filtered.length === 0 ? (
        <Card variant="outlined">
          <CardContent>
            <EmptyState
              icon={<NotificationsNoneIcon />}
              title={
                tab === 'unread'
                  ? 'Không có thông báo chưa đọc'
                  : tab === 'alert'
                  ? 'Không có cảnh báo'
                  : 'Chưa có thông báo nào'
              }
              description={
                tab === 'unread'
                  ? 'Bạn đã đọc hết tất cả thông báo.'
                  : tab === 'alert'
                  ? 'Hiện chưa có cảnh báo nào được ghi nhận.'
                  : 'Các thông báo mới sẽ xuất hiện ở đây.'
              }
              dense
            />
          </CardContent>
        </Card>
      ) : (
        <Stack spacing={1.5}>
          {filtered.map((n) => {
            const isAlert = n.title === 'alert'
            const unread = !n.is_read
            return (
              <Card
                key={n.id}
                variant="outlined"
                sx={{
                  borderLeft: 4,
                  borderLeftColor: unread
                    ? isAlert
                      ? 'error.main'
                      : 'primary.main'
                    : 'transparent',
                  bgcolor: (t) =>
                    isAlert
                      ? alpha(t.palette.error.main, 0.08)
                      : unread
                      ? alpha(t.palette.primary.main, 0.06)
                      : 'background.paper',
                  transition: 'border-color 0.2s ease, background-color 0.2s ease',
                }}
              >
                <CardContent sx={{ '&:last-child': { pb: 2 } }}>
                  <Stack direction="row" spacing={2} alignItems="flex-start">
                    {isAlert ? (
                      <WarningAmberIcon color="error" sx={{ mt: 0.3 }} />
                    ) : (
                      <InfoOutlinedIcon color="info" sx={{ mt: 0.3 }} />
                    )}
                    <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                      <Stack
                        direction="row"
                        spacing={1}
                        alignItems="center"
                        sx={{ flexWrap: 'wrap' }}
                      >
                        <Typography
                          variant="subtitle1"
                          fontWeight={unread ? 700 : 500}
                          color={isAlert ? 'error.main' : 'text.primary'}
                        >
                          {titleLabel(n.title)}
                        </Typography>
                        {unread && (
                          <Chip label="Mới" color="primary" size="small" />
                        )}
                      </Stack>
                      <Typography
                        variant="body2"
                        color="text.secondary"
                        sx={{
                          mt: 0.5,
                          whiteSpace: 'pre-wrap',
                          wordBreak: 'break-word',
                        }}
                      >
                        {n.message}
                      </Typography>
                      <Typography
                        variant="caption"
                        color="text.disabled"
                        sx={{ mt: 0.5, display: 'block' }}
                      >
                        {relTime(n.created_at)}
                      </Typography>
                    </Box>
                    <Stack direction="row" spacing={0.5}>
                      {unread && (
                        <Tooltip title="Đánh dấu đã đọc">
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={() => handleMarkRead(n.id)}
                          >
                            <DoneAllIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                      <Tooltip title="Xoá">
                        <span>
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => handleRemove(n.id)}
                            disabled={busyId === n.id}
                          >
                            {busyId === n.id ? (
                              <CircularProgress size={18} />
                            ) : (
                              <DeleteOutlineIcon fontSize="small" />
                            )}
                          </IconButton>
                        </span>
                      </Tooltip>
                    </Stack>
                  </Stack>
                </CardContent>
              </Card>
            )
          })}
        </Stack>
      )}

      <Snackbar
        open={Boolean(error)}
        autoHideDuration={4000}
        onClose={() => setError('')}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert severity="error" variant="filled" onClose={() => setError('')}>
          {error}
        </Alert>
      </Snackbar>
    </Box>
  )
}
