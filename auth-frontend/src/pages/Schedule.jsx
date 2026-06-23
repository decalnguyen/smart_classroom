import { useState, useEffect, useCallback, useMemo } from 'react'
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
  IconButton,
  Tooltip,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import { useTheme, alpha } from '@mui/material/styles'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import RoomIcon from '@mui/icons-material/Room'
import MeetingRoomIcon from '@mui/icons-material/MeetingRoom'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import CalendarMonthIcon from '@mui/icons-material/CalendarMonth'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { scheduleApi, apiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

// Personal sessions (returned with an id + editable:true) can be edited/deleted
// here; class-derived sessions (editable:false, no id) are read-only.

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
  const { role } = useAuth()
  const isTeacher = role === 'teacher'
  const isAdmin = role === 'admin'
  const [weekly, setWeekly] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [form, setForm] = useState(emptyForm)
  const [editingId, setEditingId] = useState(null) // null = create mode
  const [submitting, setSubmitting] = useState(false)
  const [toast, setToast] = useState(null)
  const [roomFilter, setRoomFilter] = useState('all') // admin room-usage filter
  // 'day' = week board (per weekday); 'room' = grouped by classroom. Admin's
  // room-usage view defaults to by-room (easier to read which room is used when).
  const [viewMode, setViewMode] = useState(isAdmin ? 'room' : 'day')

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
    const payload = {
      title: form.title.trim(),
      desc: form.desc.trim(),
      room: form.room.trim(),
      day: form.day,
      time: form.time.trim(),
    }
    try {
      if (editingId != null) {
        await scheduleApi.update(editingId, payload)
        setToast({ severity: 'success', msg: 'Đã cập nhật tiết học.' })
      } else {
        await scheduleApi.create(payload)
        setToast({ severity: 'success', msg: 'Đã thêm tiết học.' })
      }
      setForm(emptyForm)
      setEditingId(null)
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Lưu tiết học thất bại.') })
    } finally {
      setSubmitting(false)
    }
  }

  // Start editing a personal session: prefill the form (do NOT send id/account_id).
  const startEdit = (dayKey, s) => {
    setEditingId(s.id)
    setForm({ day: dayKey, time: s.time || '', title: s.title || '', room: s.room || '', desc: s.desc || '' })
    if (typeof window !== 'undefined') window.scrollTo({ top: document.body.scrollHeight, behavior: 'smooth' })
  }
  const cancelEdit = () => {
    setEditingId(null)
    setForm(emptyForm)
  }
  const removeSession = async (s) => {
    // eslint-disable-next-line no-alert
    if (!window.confirm(`Xoá tiết "${s.title || 'tiết học'}"?`)) return
    try {
      await scheduleApi.remove(s.id)
      setToast({ severity: 'success', msg: 'Đã xoá tiết học.' })
      if (editingId === s.id) cancelEdit()
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xoá tiết học thất bại.') })
    }
  }

  // Distinct rooms across the week (admin room-usage filter).
  const roomList = weekly
    ? [...new Set(DAYS.flatMap((d) => (Array.isArray(weekly[d.key]) ? weekly[d.key] : []).map((s) => s.room).filter(Boolean)))].sort()
    : []

  // Sessions for a day, applying the admin room filter.
  const sessionsForDay = (dayKey) => {
    const list = Array.isArray(weekly?.[dayKey]) ? weekly[dayKey] : []
    if (isAdmin && roomFilter !== 'all') return list.filter((s) => s.room === roomFilter)
    return list
  }

  // Total sessions across the whole week (for the empty state; respects filter).
  const totalSessions = weekly
    ? DAYS.reduce((sum, d) => sum + sessionsForDay(d.key).length, 0)
    : 0

  // Accent color for a day key (room view borders/chips).
  const dayColor = (key) => DAYS.find((d) => d.key === key)?.color || theme.palette.primary.main

  // Group every session by classroom (room view). Respects the admin room filter.
  // Each room → its days that have sessions, each day's sessions sorted by time.
  const byRoom = useMemo(() => {
    if (!weekly) return []
    const map = {}
    DAYS.forEach((d) => {
      let list = Array.isArray(weekly[d.key]) ? weekly[d.key] : []
      if (isAdmin && roomFilter !== 'all') list = list.filter((s) => s.room === roomFilter)
      list.forEach((s) => {
        const room = s.room || 'Chưa gán phòng'
        if (!map[room]) map[room] = {}
        if (!map[room][d.key]) map[room][d.key] = []
        map[room][d.key].push(s)
      })
    })
    return Object.keys(map)
      .sort()
      .map((room) => ({
        room,
        count: Object.values(map[room]).reduce((n, arr) => n + arr.length, 0),
        days: DAYS.filter((d) => map[room][d.key]?.length).map((d) => ({
          ...d,
          sessions: [...map[room][d.key]].sort((a, b) => (a.time || '').localeCompare(b.time || '')),
        })),
      }))
  }, [weekly, roomFilter, isAdmin])

  // One session card, shared by both views. showRoom=false in room view (redundant).
  const renderSessionCard = (s, dayKey, color, keyStr, showRoom = true) => (
    <Card
      key={keyStr}
      variant="outlined"
      sx={{
        borderLeft: 4,
        borderLeftColor: color,
        bgcolor: 'background.paper',
        transition: 'box-shadow .15s, border-color .15s',
        '&:hover': { boxShadow: 2 },
      }}
    >
      <CardContent sx={{ p: 1.5, '&:last-child': { pb: 1.5 } }}>
        <Stack direction="row" alignItems="center" spacing={0.5} sx={{ mb: 0.5 }}>
          <AccessTimeIcon fontSize="small" sx={{ color }} />
          <Typography variant="caption" color="text.secondary">
            {s.time || '--'}
          </Typography>
          {s.editable && s.id != null && (
            <Stack direction="row" spacing={0} sx={{ ml: 'auto' }}>
              <Tooltip title="Sửa">
                <IconButton size="small" onClick={() => startEdit(dayKey, s)}>
                  <EditIcon sx={{ fontSize: 16 }} />
                </IconButton>
              </Tooltip>
              <Tooltip title="Xoá">
                <IconButton size="small" color="error" onClick={() => removeSession(s)}>
                  <DeleteIcon sx={{ fontSize: 16 }} />
                </IconButton>
              </Tooltip>
            </Stack>
          )}
        </Stack>
        <Typography variant="body2" fontWeight={600} color="text.primary">
          {s.title || 'Tiết học'}
        </Typography>
        {showRoom && s.room ? (
          <Chip size="small" icon={<RoomIcon />} label={s.room} variant="outlined" sx={{ mt: 0.5 }} />
        ) : null}
        {s.desc ? (
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
            {s.desc}
          </Typography>
        ) : null}
      </CardContent>
    </Card>
  )

  return (
    <Box>
      <PageHeader
        title={isAdmin ? 'Lịch sử dụng phòng học' : isTeacher ? 'Lịch dạy' : 'Lịch học cá nhân'}
        subtitle={isAdmin ? 'Thời khóa biểu tất cả phòng học trong tuần (môn · tiết · phòng · GV)' : isTeacher ? 'Thời khóa biểu giảng dạy + tiết cá nhân' : 'Thời khóa biểu theo tài khoản'}
        action={
          <Stack direction="row" spacing={1.5} alignItems="center" flexWrap="wrap" useFlexGap>
            <ToggleButtonGroup size="small" exclusive value={viewMode} onChange={(_, v) => v && setViewMode(v)}>
              <ToggleButton value="day">Theo thứ</ToggleButton>
              <ToggleButton value="room">Theo phòng</ToggleButton>
            </ToggleButtonGroup>
            {isAdmin && roomList.length ? (
              <TextField select size="small" label="Phòng" value={roomFilter} onChange={(e) => setRoomFilter(e.target.value)} sx={{ minWidth: 160 }}>
                <MenuItem value="all">Tất cả phòng</MenuItem>
                {roomList.map((r) => <MenuItem key={r} value={r}>{r}</MenuItem>)}
              </TextField>
            ) : null}
          </Stack>
        }
      />

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
              description={isAdmin ? 'Chưa có lớp học nào trong thời khóa biểu. Thêm lớp ở trang Quản trị.' : 'Tuần này chưa có tiết học. Hãy thêm tiết học ở biểu mẫu bên dưới.'}
              dense
            />
          </CardContent>
        </Card>
      ) : viewMode === 'room' ? (
        /* Grouped by classroom: one card per room, days with sessions inside. */
        <Box
          sx={{
            display: 'grid',
            gap: 2,
            gridTemplateColumns: { xs: '1fr', md: 'repeat(2, 1fr)', lg: 'repeat(3, 1fr)' },
            mb: 4,
          }}
        >
          {byRoom.map((r) => (
            <Card key={r.room} variant="outlined" sx={{ overflow: 'hidden' }}>
              <Box
                sx={{
                  px: 2,
                  py: 1.25,
                  display: 'flex',
                  alignItems: 'center',
                  gap: 1,
                  bgcolor: alpha(theme.palette.primary.main, 0.1),
                  borderBottom: 1,
                  borderColor: 'divider',
                }}
              >
                <MeetingRoomIcon fontSize="small" color="primary" />
                <Typography variant="subtitle1" fontWeight={700}>{r.room}</Typography>
                <Chip size="small" label={`${r.count} tiết`} sx={{ ml: 'auto' }} />
              </Box>
              <CardContent sx={{ p: 1.5 }}>
                <Stack spacing={1.5}>
                  {r.days.map((d) => (
                    <Box key={d.key}>
                      <Typography variant="caption" fontWeight={700} sx={{ color: d.color, display: 'block', mb: 0.5 }}>
                        {d.label}
                      </Typography>
                      <Stack spacing={1}>
                        {d.sessions.map((s, i) =>
                          renderSessionCard(s, d.key, d.color, `${r.room}-${d.key}-${s.time}-${s.title}-${i}`, false),
                        )}
                      </Stack>
                    </Box>
                  ))}
                </Stack>
              </CardContent>
            </Card>
          ))}
        </Box>
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
            const sessions = sessionsForDay(d.key)
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
                    sessions.map((s, i) =>
                      renderSessionCard(s, d.key, d.color, `${d.key}-${s.time}-${s.title}-${i}`, true),
                    )
                  )}
                </Stack>
              </Box>
            )
          })}
        </Box>
      )}

      {/* Add session form — personal timetable (students/teachers only). Admin's
          room-usage view is managed via the Quản trị (classes) page, not here. */}
      {!isAdmin && (
      <Card>
        <CardContent>
          <Typography variant="h6" mb={2}>
            {editingId != null ? 'Sửa tiết học' : 'Thêm tiết học'}
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
            <Stack direction="row" spacing={1}>
              <Button
                type="submit"
                variant="contained"
                startIcon={submitting ? <CircularProgress size={18} color="inherit" /> : editingId != null ? <EditIcon /> : <AddIcon />}
                disabled={submitting}
              >
                {submitting ? 'Đang lưu...' : editingId != null ? 'Cập nhật' : 'Thêm tiết học'}
              </Button>
              {editingId != null && (
                <Button variant="text" onClick={cancelEdit} disabled={submitting}>Huỷ</Button>
              )}
            </Stack>
          </Box>
        </CardContent>
      </Card>
      )}

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
