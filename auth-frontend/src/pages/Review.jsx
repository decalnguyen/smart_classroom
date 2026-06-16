import { useState, useEffect, useCallback } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, Button, Chip,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
  Snackbar, Alert, Skeleton, ToggleButton, ToggleButtonGroup, LinearProgress,
} from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import { reviewApi, apiError } from '../api/client'

const STATUS = {
  pending: { label: 'Chờ duyệt', color: 'warning' },
  confirmed: { label: 'Đã xác nhận', color: 'success' },
  rejected: { label: 'Đã từ chối', color: 'error' },
}

// Color the confidence bar: low → red, mid → amber, high → green.
function confColor(c) {
  if (c >= 0.7) return 'success'
  if (c >= 0.5) return 'warning'
  return 'error'
}

export default function Review() {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState('pending')
  const [toast, setToast] = useState(null)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const { data } = await reviewApi.list(filter)
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Không tải được hàng đợi duyệt.') })
    } finally {
      setLoading(false)
    }
  }, [filter])

  useEffect(() => { load() }, [load])

  const decide = async (id, decision) => {
    try {
      await reviewApi.decide(id, decision)
      setToast({ severity: 'success', msg: decision === 'confirm' ? 'Đã xác nhận điểm danh.' : 'Đã từ chối.' })
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xử lý thất bại.') })
    }
  }

  return (
    <Box>
      <PageHeader
        title="Duyệt nhận diện"
        subtitle="Các lượt nhận diện độ tin cậy thấp cần con người xác nhận"
      />

      <Stack direction="row" justifyContent="flex-end" mb={2}>
        <ToggleButtonGroup size="small" exclusive value={filter} onChange={(_, v) => v && setFilter(v)}>
          <ToggleButton value="pending">Chờ duyệt</ToggleButton>
          <ToggleButton value="confirmed">Đã xác nhận</ToggleButton>
          <ToggleButton value="rejected">Đã từ chối</ToggleButton>
        </ToggleButtonGroup>
      </Stack>

      <Card>
        <CardContent>
          {loading ? (
            <Stack spacing={1}>{Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="rounded" height={44} />)}</Stack>
          ) : rows.length === 0 ? (
            <EmptyState dense icon={<span style={{ fontSize: 28 }}>🧑‍💻</span>} title="Không có mục nào" description="Không có lượt nhận diện nào ở trạng thái này." />
          ) : (
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 600 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>Học sinh</TableCell>
                    <TableCell>Phòng / Môn</TableCell>
                    <TableCell>Thời điểm</TableCell>
                    <TableCell>Độ tin cậy</TableCell>
                    <TableCell>Trạng thái</TableCell>
                    <TableCell align="right">Quyết định</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((r) => {
                    const s = STATUS[r.status] || { label: r.status, color: 'default' }
                    const pct = Math.round((r.confidence || 0) * 100)
                    return (
                      <TableRow key={r.id} hover>
                        <TableCell>
                          {r.student_name}{' '}
                          <Typography component="span" variant="caption" color="text.secondary">{r.mssv}</Typography>
                        </TableCell>
                        <TableCell>#{r.classroom_id} · {r.subject || '—'}</TableCell>
                        <TableCell>{r.date} {r.detection_time}</TableCell>
                        <TableCell sx={{ minWidth: 140 }}>
                          <Stack spacing={0.5}>
                            <LinearProgress variant="determinate" value={pct} color={confColor(r.confidence || 0)} sx={{ height: 6, borderRadius: 3 }} />
                            <Typography variant="caption" color="text.secondary">{pct}%</Typography>
                          </Stack>
                        </TableCell>
                        <TableCell><Chip size="small" color={s.color} label={s.label} /></TableCell>
                        <TableCell align="right">
                          {r.status === 'pending' ? (
                            <Stack direction="row" spacing={1} justifyContent="flex-end">
                              <Button size="small" color="success" variant="outlined" startIcon={<CheckIcon />} onClick={() => decide(r.id, 'confirm')}>Đúng</Button>
                              <Button size="small" color="error" variant="outlined" startIcon={<CloseIcon />} onClick={() => decide(r.id, 'reject')}>Sai</Button>
                            </Stack>
                          ) : (
                            <Typography variant="caption" color="text.secondary">đã xử lý</Typography>
                          )}
                        </TableCell>
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
