import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  Button,
  TextField,
  MenuItem,
  CircularProgress,
  Alert,
  Snackbar,
  Chip,
  Skeleton,
} from '@mui/material'
import { useTheme, alpha } from '@mui/material/styles'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import RoomIcon from '@mui/icons-material/Room'
import AddIcon from '@mui/icons-material/Add'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { scheduleApi, apiError } from '../api/client'

// NOTE: scheduleApi.getWeekly() returns sessions WITHOUT ids, so this view
// only supports adding + viewing. Editing / deleting individual sessions is
// not possible here — do NOT call scheduleApi.remove from this page.

// Backend day keys (Monday..Sunday) mapped to Vietnamese column labels.
// Each day gets an accent color (icons / header chip / card border only).
const DAYS = [
  { key: 'Monday', label: 'Thứ 2', color: '#2563eb' },
  { key: 'Tuesday', label: 'Thứ 3', color: '#7c3aed' },
  { key: 'Wednesday', label: 'Thứ 4', color: '#0891b2' },
  { key: 'Thursday', label: 'Thứ 5', color: '#16a34a' },
  { key: 'Friday', label: 'Thứ 6', color: '#ea580c' },
  { key: 'Saturday', label: 'Thứ 7', color: '#db2777' },
  { key: 'Sunday', label: 'Chủ nhật', color: '#dc2626' },
]

const emptyForm = { day: 'Monday', time: '08:00', title: '', room: '', desc: '' }

const BOARD_COLS = {
  xs: '1fr',
  sm: 'repeat(2, 1fr)',
  md: 'repeat(4, 1fr)',
  lg: 'repeat(7, 1fr)',
}

export default function Schedule() {
  const theme = useTheme()
  const [weekly, setWeekly] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [form, setForm] = useState(emptyForm)
  const [submitting, setSubmitting] = useState(false)
  const [toast, setToast] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const res = await scheduleApi.getWeekly()
      setWeekly(res.data || {})
    } catch (err) {
      setError(apiError(err, 'Không thể tải lịch học. Vui lòng thử lại.'))
      setWeekly(null)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const handleChange = (field) => (e) =>
    setForm((prev) => ({ ...prev, [field]: e.target.value }))

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!form.title.trim() || !form.time.trim()) {
      setToast({ severity: 'warning', msg: 'Vui lòng nhập tên tiết học và giờ.' })
      return
    }
    setSubmitting(true)
    try {
      await scheduleApi.create({
        title: form.title.trim(),
        desc: form.desc.trim(),
        room: form.room.trim(),
        day: form.day,
        time: form.time.trim(),
      })
      setToast({ severity: 'success', msg: 'Đã thêm tiết học.' })
      setForm(emptyForm)
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Thêm tiết học thất bại.') })
    } finally {
      setSubmitting(false)
    }
  }

  // Total sessions across the whole week (for the empty state).
  const totalSessions = weekly
    ? DAYS.reduce((sum, d) => sum + (Array.isArray(weekly[d.key]) ? weekly[d.key].length : 0), 0)
    : 0

  return (
    <Box>
      <PageHeader title="Lịch học cá nhân" subtitle="Thời khóa biểu theo tài khoản" />

      {/* Week board */}
      {loading ? (
        <Box
          sx={{
            display: 'grid',
            gap: 2,
            gridTemplateColumns: BOARD_COLS,
            mb: 4,
          }}
        >
          {DAYS.map((d) => (
            <Stack key={d.key} spacing={1}>
              <Skeleton variant="rounded" height={32} />
              <Skeleton variant="rounded" height={92} />
              <Skeleton variant="rounded" height={92} />
            </Stack>
          ))}
        </Box>
      ) : error ? (
        <Alert
          severity="error"
          sx={{ mb: 3 }}
          action={
            <Button color="inherit" size="small" onClick={load}>
              Thử lại
            </Button>
          }
        >
          {error}
        </Alert>
      ) : totalSessions === 0 ? (
        <Card sx={{ mb: 4 }}>
          <CardContent>
            <EmptyState
              icon={<CalendarMonthIcon />}
              title="Chưa có tiết học nào"
              description="Tuần này chưa có tiết học. Hãy thêm tiết học ở biểu mẫu bên dưới."
              dense
            />
          </CardContent>
        </Card>
      ) : (
        <Box
          sx={{
            display: 'grid',
            gap: 2,
            gridTemplateColumns: BOARD_COLS,
            mb: 4,
          }}
        >
          {DAYS.map((d) => {
            const sessions = Array.isArray(weekly[d.key]) ? weekly[d.key] : []
            return (
              <Box key={d.key}>
                {/* Subtle colored header chip per day */}
                <Box
                  sx={{
                    mb: 1,
                    py: 0.75,
                    px: 1,
                    borderRadius: 1.5,
                    textAlign: 'center',
                    bgcolor: alpha(d.color, 0.12),
                    border: 1,
                    borderColor: alpha(d.color, 0.35),
                  }}
                >
                  <Typography variant="subtitle2" fontWeight={700} sx={{ color: d.color }}>
                    {d.label}
                  </Typography>
                  <Typography variant="caption" sx={{ color: d.color, opacity: 0.85 }}>
                    {sessions.length} tiết
                  </Typography>
                </Box>
                <Stack spacing={1}>
                  {sessions.length === 0 ? (
                    <Typography
                      variant="caption"
                      color="text.secondary"
                      align="center"
                      sx={{ py: 1 }}
                    >
                      Không có tiết
                    </Typography>
                  ) : (
                    sessions.map((s, i) => (
                      <Card
                        key={`${d.key}-${s.time}-${s.title}-${i}`}
                        variant="outlined"
                        sx={{
                          borderLeft: 4,
                          borderLeftColor: d.color,
                          bgcolor: 'background.paper',
                          transition: 'box-shadow .15s, border-color .15s',
                          '&:hover': { boxShadow: 2 },
                        }}
                      >
                        <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
                          <Stack direction="row" alignItems="center" spacing={0.5} sx={{ mb: 0.5 }}>
                            <AccessTimeIcon fontSize="small" sx={{ color: d.color }} />
                            <Typography variant="caption" color="text.secondary">
                              {s.time || '--'}
                            </Typography>
                          </Stack>
                          <Typography variant="body2" fontWeight={600} color="text.primary">
                            {s.title || 'Tiết học'}
                          </Typography>
                          {s.room ? (
                            <Chip
                              size="small"
                              icon={<RoomIcon />}
                              label={s.room}
                              variant="outlined"
                              sx={{ mt: 0.5 }}
                            />
                          ) : null}
                          {s.desc ? (
                            <Typography
                              variant="caption"
                              color="text.secondary"
                              sx={{ display: 'block', mt: 0.5 }}
                            >
                              {s.desc}
                            </Typography>
                          ) : null}
                        </CardContent>
                      </Card>
                    ))
                  )}
                </Stack>
              </Box>
            )
          })}
        </Box>
      )}

      {/* Add session form (any role can add their own) */}
      <Card>
        <CardContent>
          <Typography variant="h6" mb={2}>
            Thêm tiết học
          </Typography>
          <Box component="form" onSubmit={handleSubmit}>
            <Box
              sx={{
                display: 'grid',
                gap: 2,
                gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)' },
                mb: 2,
              }}
            >
              <TextField
                select
                label="Thứ"
                value={form.day}
                onChange={handleChange('day')}
                fullWidth
              >
                {DAYS.map((d) => (
                  <MenuItem key={d.key} value={d.key}>
                    {d.label}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                label="Giờ"
                placeholder="08:00"
                value={form.time}
                onChange={handleChange('time')}
                fullWidth
                required
              />
              <TextField
                label="Tên tiết học"
                value={form.title}
                onChange={handleChange('title')}
                fullWidth
                required
              />
              <TextField
                label="Phòng"
                value={form.room}
                onChange={handleChange('room')}
                fullWidth
              />
              <TextField
                label="Mô tả"
                value={form.desc}
                onChange={handleChange('desc')}
                fullWidth
                sx={{ gridColumn: { sm: 'span 2', md: 'span 2' } }}
              />
            </Box>
            <Button
              type="submit"
              variant="contained"
              startIcon={submitting ? <CircularProgress size={18} color="inherit" /> : <AddIcon />}
              disabled={submitting}
            >
              {submitting ? 'Đang thêm...' : 'Thêm tiết học'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Snackbar
        open={Boolean(toast)}
        autoHideDuration={3000}
        onClose={() => setToast(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        {toast ? (
          <Alert severity={toast.severity} onClose={() => setToast(null)} sx={{ width: '100%' }}>
            {toast.msg}
          </Alert>
        ) : undefined}
      </Snackbar>
    </Box>
  )
}
