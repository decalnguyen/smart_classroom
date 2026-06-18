import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, TextField, Button, Chip,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
  Snackbar, Alert, Skeleton,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
import {
  ResponsiveContainer, PieChart, Pie, Cell, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, Legend,
} from 'recharts'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { leaveApi, apiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

// Monday-of-week (local) timestamp for an ISO date string — for weekly bucketing.
function weekStart(dateStr) {
  const d = new Date(`${dateStr}T00:00:00`)
  if (Number.isNaN(d.getTime())) return 0
  const dow = (d.getDay() + 6) % 7 // Monday = 0
  d.setDate(d.getDate() - dow)
  d.setHours(0, 0, 0, 0)
  return d.getTime()
}

const STATUS = {
  pending: { label: 'Chờ duyệt', color: 'warning' },
  approved: { label: 'Đã duyệt', color: 'success' },
  rejected: { label: 'Từ chối', color: 'error' },
}

function todayStr() {
  return new Date().toISOString().slice(0, 10)
}

export default function Leaves() {
  const { role } = useAuth()
  const theme = useTheme()
  const isStaff = role === 'admin' || role === 'teacher'
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [date, setDate] = useState(todayStr())
  const [reason, setReason] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [toast, setToast] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const { data } = await leaveApi.list()
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Không tải được danh sách đơn.') })
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => { load() }, [load])

  const submit = async () => {
    if (!date) {
      setToast({ severity: 'warning', msg: 'Chọn ngày nghỉ.' })
      return
    }
    setSubmitting(true)
    try {
      await leaveApi.create({ date, reason })
      setToast({ severity: 'success', msg: 'Đã gửi đơn xin nghỉ.' })
      setReason('')
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Gửi đơn thất bại.') })
    } finally {
      setSubmitting(false)
    }
  }

  const review = async (id, status) => {
    try {
      await leaveApi.review(id, status)
      setToast({ severity: 'success', msg: status === 'approved' ? 'Đã duyệt đơn.' : 'Đã từ chối đơn.' })
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xử lý đơn thất bại.') })
    }
  }

  // Status breakdown (donut) + weekly volume (stacked bar) — staff analytics.
  const statusData = useMemo(() => {
    const counts = rows.reduce((a, r) => { a[r.status] = (a[r.status] || 0) + 1; return a }, {})
    return ['pending', 'approved', 'rejected']
      .filter((k) => counts[k])
      .map((k) => ({ key: k, name: STATUS[k].label, value: counts[k], fill: theme.palette[STATUS[k].color]?.main || theme.palette.grey[500] }))
  }, [rows, theme])
  const pendingCount = useMemo(() => rows.filter((r) => r.status === 'pending').length, [rows])
  const weeklyData = useMemo(() => {
    const m = new Map()
    rows.forEach((r) => {
      const w = weekStart(r.date)
      const o = m.get(w) || { w, pending: 0, approved: 0, rejected: 0 }
      if (o[r.status] != null) o[r.status] += 1
      m.set(w, o)
    })
    return [...m.values()].sort((a, b) => a.w - b.w)
  }, [rows])

  return (
    <Box>
      <PageHeader
        title="Đơn xin nghỉ"
        subtitle={isStaff ? 'Duyệt đơn xin nghỉ của học sinh' : 'Gửi và theo dõi đơn xin nghỉ của bạn'}
      />

      {/* Student: create form */}
      {!isStaff && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="h6" mb={2}>Tạo đơn xin nghỉ</Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }}>
              <TextField size="small" type="date" label="Ngày nghỉ" value={date} onChange={(e) => setDate(e.target.value)} InputLabelProps={{ shrink: true }} />
              <TextField size="small" label="Lý do" value={reason} onChange={(e) => setReason(e.target.value)} sx={{ flex: 1, minWidth: 220 }} />
              <Button variant="contained" onClick={submit} disabled={submitting}>{submitting ? 'Đang gửi...' : 'Gửi đơn'}</Button>
            </Stack>
          </CardContent>
        </Card>
      )}

      {/* Staff analytics: status breakdown + weekly volume */}
      {isStaff && !loading && rows.length > 0 && (
        <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: '1fr 2fr' }, mb: 3 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" mb={1}>Trạng thái đơn xin nghỉ</Typography>
              <Box sx={{ position: 'relative', height: 240 }}>
                <ResponsiveContainer width="100%" height="100%">
                  <PieChart>
                    <Pie data={statusData} dataKey="value" nameKey="name" innerRadius={62} outerRadius={92} paddingAngle={2}>
                      {statusData.map((d) => <Cell key={d.key} fill={d.fill} />)}
                    </Pie>
                    <RTooltip formatter={(v, n) => [`${v} đơn`, n]} contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                    <Legend />
                  </PieChart>
                </ResponsiveContainer>
                <Box sx={{ position: 'absolute', top: '42%', left: 0, right: 0, textAlign: 'center', transform: 'translateY(-50%)', pointerEvents: 'none' }}>
                  <Typography variant="h4" fontWeight={800} color={pendingCount ? 'warning.main' : 'text.primary'}>{pendingCount}</Typography>
                  <Typography variant="caption" color="text.secondary">đang chờ duyệt · {rows.length} đơn</Typography>
                </Box>
              </Box>
            </CardContent>
          </Card>
          <Card>
            <CardContent>
              <Typography variant="h6" mb={1}>Số đơn xin nghỉ theo tuần</Typography>
              <Box sx={{ height: 240 }}>
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={weeklyData} margin={{ top: 8, right: 12, bottom: 0, left: -12 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'} />
                    <XAxis dataKey="w" tickFormatter={(d) => new Date(d).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit' })} tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                    <YAxis allowDecimals={false} tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                    <RTooltip labelFormatter={(d) => `Tuần từ ${new Date(d).toLocaleDateString('vi-VN')}`} formatter={(v, n) => [`${v} đơn`, n]} contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                    <Legend />
                    <Bar dataKey="pending" stackId="s" name="Chờ duyệt" fill={theme.palette.warning.main} />
                    <Bar dataKey="approved" stackId="s" name="Đã duyệt" fill={theme.palette.success.main} />
                    <Bar dataKey="rejected" stackId="s" name="Từ chối" fill={theme.palette.error.main} radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </Box>
            </CardContent>
          </Card>
        </Box>
      )}

      <Card>
        <CardContent>
          <Typography variant="h6" mb={2}>{isStaff ? 'Tất cả đơn xin nghỉ' : 'Đơn của tôi'}</Typography>
          {loading ? (
            <Stack spacing={1}>{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="rounded" height={44} />)}</Stack>
          ) : rows.length === 0 ? (
            <EmptyState dense icon={<EventBusyPlaceholder />} title="Chưa có đơn" description="Chưa có đơn xin nghỉ nào." />
          ) : (
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 560 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    {isStaff && <TableCell>Học sinh</TableCell>}
                    <TableCell>Ngày nghỉ</TableCell>
                    <TableCell>Lý do</TableCell>
                    <TableCell>Trạng thái</TableCell>
                    {isStaff && <TableCell align="right">Duyệt</TableCell>}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((r) => {
                    const s = STATUS[r.status] || { label: r.status, color: 'default' }
                    return (
                      <TableRow key={r.id} hover>
                        {isStaff && <TableCell>{r.student_name} <Typography component="span" variant="caption" color="text.secondary">#{r.student_id}</Typography></TableCell>}
                        <TableCell>{r.date}</TableCell>
                        <TableCell>{r.reason || '—'}</TableCell>
                        <TableCell><Chip size="small" color={s.color} label={s.label} /></TableCell>
                        {isStaff && (
                          <TableCell align="right">
                            {r.status === 'pending' ? (
                              <Stack direction="row" spacing={1} justifyContent="flex-end">
                                <Button size="small" color="success" variant="outlined" startIcon={<CheckIcon />} onClick={() => review(r.id, 'approved')}>Duyệt</Button>
                                <Button size="small" color="error" variant="outlined" startIcon={<CloseIcon />} onClick={() => review(r.id, 'rejected')}>Từ chối</Button>
                              </Stack>
                            ) : (
                              <Typography variant="caption" color="text.secondary">đã xử lý</Typography>
                            )}
                          </TableCell>
                        )}
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Snackbar open={!!toast} autoHideDuration={3000} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}>
        {toast ? <Alert severity={toast.severity} onClose={() => setToast(null)}>{toast.msg}</Alert> : null}
      </Snackbar>
    </Box>
  )
}

function EventBusyPlaceholder() {
  return <span style={{ fontSize: 28 }}>🗓️</span>
}
