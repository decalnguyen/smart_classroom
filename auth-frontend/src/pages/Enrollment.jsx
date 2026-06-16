import { useState, useEffect, useCallback, useRef } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, TextField, Button, Chip, IconButton,
  Table, TableContainer, TableHead, TableBody, TableRow, TableCell, Paper,
  Snackbar, Alert, Skeleton, ToggleButton, ToggleButtonGroup, Dialog, DialogTitle,
  DialogContent, DialogActions, Tabs, Tab, Tooltip,
} from '@mui/material'
import AddAPhotoIcon from '@mui/icons-material/AddAPhoto'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import CameraAltIcon from '@mui/icons-material/CameraAlt'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import EmptyState from '../components/EmptyState'
import { enrollmentApi, apiError } from '../api/client'

export default function Enrollment() {
  const [rows, setRows] = useState([])
  const [enrolledTotal, setEnrolledTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [q, setQ] = useState('')
  const [only, setOnly] = useState('all')
  const [toast, setToast] = useState(null)
  const [target, setTarget] = useState(null) // student being enrolled

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const params = {}
      if (q.trim()) params.q = q.trim()
      if (only !== 'all') params.only = only
      const { data } = await enrollmentApi.status(params)
      setRows(Array.isArray(data.students) ? data.students : [])
      setEnrolledTotal(data.enrolled_total || 0)
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Không tải được danh sách.') })
    } finally {
      setLoading(false)
    }
  }, [q, only])

  // Debounce search.
  useEffect(() => {
    const t = setTimeout(load, 300)
    return () => clearTimeout(t)
  }, [load])

  const remove = async (s) => {
    if (!window.confirm(`Xoá khuôn mặt đã đăng ký của ${s.student_name}?`)) return
    try {
      await enrollmentApi.remove(s.student_id)
      setToast({ severity: 'success', msg: 'Đã xoá.' })
      load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xoá thất bại.') })
    }
  }

  return (
    <Box>
      <PageHeader title="Đăng ký khuôn mặt" subtitle="Quản lý dữ liệu khuôn mặt tham chiếu của sinh viên" />

      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr 1fr', sm: 'repeat(3, 1fr)' }, gap: 2, mb: 3 }}>
        <StatCard label="Đã đăng ký" value={enrolledTotal} />
        <StatCard label="Đang hiển thị" value={rows.length} />
        <StatCard label="Chưa đăng ký (đang lọc)" value={rows.filter((r) => !r.samples).length} />
      </Box>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }} justifyContent="space-between">
            <TextField size="small" label="Tìm MSSV / tên" value={q} onChange={(e) => setQ(e.target.value)} sx={{ minWidth: 260 }} />
            <ToggleButtonGroup size="small" exclusive value={only} onChange={(_, v) => v && setOnly(v)}>
              <ToggleButton value="all">Tất cả</ToggleButton>
              <ToggleButton value="enrolled">Đã đăng ký</ToggleButton>
              <ToggleButton value="missing">Chưa đăng ký</ToggleButton>
            </ToggleButtonGroup>
          </Stack>
        </CardContent>
      </Card>

      <Card>
        <CardContent>
          {loading ? (
            <Stack spacing={1}>{Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} variant="rounded" height={44} />)}</Stack>
          ) : rows.length === 0 ? (
            <EmptyState dense icon={<span style={{ fontSize: 28 }}>🧑‍🎓</span>} title="Không có sinh viên" description="Thử đổi từ khoá hoặc bộ lọc." />
          ) : (
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 600 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>MSSV</TableCell>
                    <TableCell>Họ tên</TableCell>
                    <TableCell align="center">Trạng thái</TableCell>
                    <TableCell align="center">Số mẫu</TableCell>
                    <TableCell align="right">Thao tác</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((s) => (
                    <TableRow key={s.student_id} hover>
                      <TableCell>{s.mssv}</TableCell>
                      <TableCell>{s.student_name}</TableCell>
                      <TableCell align="center">
                        {s.samples > 0
                          ? <Chip size="small" color="success" label="Đã đăng ký" />
                          : <Chip size="small" variant="outlined" label="Chưa" />}
                      </TableCell>
                      <TableCell align="center">{s.samples || 0}</TableCell>
                      <TableCell align="right">
                        <Stack direction="row" spacing={1} justifyContent="flex-end">
                          <Button size="small" variant="outlined" startIcon={<AddAPhotoIcon />} onClick={() => setTarget(s)}>
                            {s.samples > 0 ? 'Đăng ký lại' : 'Đăng ký'}
                          </Button>
                          {s.samples > 0 && (
                            <Tooltip title="Xoá khuôn mặt">
                              <IconButton size="small" color="error" onClick={() => remove(s)}><DeleteOutlineIcon fontSize="small" /></IconButton>
                            </Tooltip>
                          )}
                        </Stack>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      {target && (
        <EnrollDialog
          student={target}
          onClose={() => setTarget(null)}
          onDone={(msg) => { setToast({ severity: 'success', msg }); setTarget(null); load() }}
          onError={(msg) => setToast({ severity: 'error', msg })}
        />
      )}

      <Snackbar open={!!toast} autoHideDuration={4000} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}>
        {toast ? <Alert severity={toast.severity} onClose={() => setToast(null)}>{toast.msg}</Alert> : null}
      </Snackbar>
    </Box>
  )
}

// EnrollDialog: capture from webcam OR upload a file, then send to backend.
function EnrollDialog({ student, onClose, onDone, onError }) {
  const [tab, setTab] = useState(0)
  const [blob, setBlob] = useState(null)
  const [preview, setPreview] = useState(null)
  const [submitting, setSubmitting] = useState(false)
  const videoRef = useRef(null)
  const canvasRef = useRef(null)
  const streamRef = useRef(null)

  useEffect(() => {
    if (tab !== 0) { stopCam(); return }
    let active = true
    navigator.mediaDevices?.getUserMedia({ video: { width: 640, height: 480 } })
      .then((stream) => {
        if (!active) { stream.getTracks().forEach((t) => t.stop()); return }
        streamRef.current = stream
        if (videoRef.current) { videoRef.current.srcObject = stream; videoRef.current.play() }
      })
      .catch(() => onError('Không mở được webcam. Hãy dùng tab "Tải ảnh lên".'))
    return () => { active = false; stopCam() }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab])

  const stopCam = () => {
    streamRef.current?.getTracks().forEach((t) => t.stop())
    streamRef.current = null
  }

  const capture = () => {
    const v = videoRef.current, c = canvasRef.current
    if (!v || !c) return
    c.width = v.videoWidth || 640
    c.height = v.videoHeight || 480
    c.getContext('2d').drawImage(v, 0, 0, c.width, c.height)
    c.toBlob((b) => { setBlob(b); setPreview(URL.createObjectURL(b)) }, 'image/jpeg', 0.92)
  }

  const onFile = (e) => {
    const f = e.target.files?.[0]
    if (f) { setBlob(f); setPreview(URL.createObjectURL(f)) }
  }

  const submit = async () => {
    if (!blob) { onError('Chưa có ảnh.'); return }
    setSubmitting(true)
    try {
      const { data } = await enrollmentApi.enrollPhoto(student.student_id, blob)
      onDone(`Đã đăng ký ${student.student_name} (${data.samples} mẫu).`)
    } catch (err) {
      onError(apiError(err, 'Đăng ký thất bại. Kiểm tra dịch vụ face-enroll.'))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onClose={() => { stopCam(); onClose() }} maxWidth="sm" fullWidth>
      <DialogTitle>Đăng ký khuôn mặt — {student.student_name} <Typography component="span" variant="caption" color="text.secondary">({student.mssv})</Typography></DialogTitle>
      <DialogContent dividers>
        <Tabs value={tab} onChange={(_, v) => { setTab(v); setBlob(null); setPreview(null) }} sx={{ mb: 2 }}>
          <Tab icon={<CameraAltIcon />} iconPosition="start" label="Webcam" />
          <Tab icon={<AddAPhotoIcon />} iconPosition="start" label="Tải ảnh lên" />
        </Tabs>

        {tab === 0 && (
          <Stack spacing={1} alignItems="center">
            <video ref={videoRef} style={{ width: '100%', maxHeight: 320, borderRadius: 8, background: '#000' }} muted playsInline />
            <Button variant="outlined" startIcon={<CameraAltIcon />} onClick={capture}>Chụp</Button>
          </Stack>
        )}
        {tab === 1 && (
          <Button variant="outlined" component="label" startIcon={<AddAPhotoIcon />}>
            Chọn ảnh
            <input hidden type="file" accept="image/*" onChange={onFile} />
          </Button>
        )}

        <canvas ref={canvasRef} style={{ display: 'none' }} />
        {preview && (
          <Box mt={2} textAlign="center">
            <Typography variant="caption" color="text.secondary">Xem trước</Typography>
            <Box component="img" src={preview} alt="preview" sx={{ display: 'block', mx: 'auto', mt: 1, maxHeight: 240, borderRadius: 1 }} />
          </Box>
        )}
        <Alert severity="info" sx={{ mt: 2 }}>
          Hệ thống dùng đúng model nhận diện để trích xuất đặc trưng (cần bật dịch vụ <code>face-enroll</code>).
          Ảnh rõ mặt, chính diện, đủ sáng sẽ cho kết quả tốt nhất.
        </Alert>
      </DialogContent>
      <DialogActions>
        <Button onClick={() => { stopCam(); onClose() }}>Huỷ</Button>
        <Button variant="contained" onClick={submit} disabled={!blob || submitting}>
          {submitting ? 'Đang xử lý...' : 'Lưu đăng ký'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
