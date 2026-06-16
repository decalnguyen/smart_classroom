import { useState, useEffect, useCallback } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, TextField, Button, Chip,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
  Snackbar, Alert, Skeleton,
} from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { leaveApi, apiError } from '../api/client'
import { useAuth } from '../context/AuthContext'

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
