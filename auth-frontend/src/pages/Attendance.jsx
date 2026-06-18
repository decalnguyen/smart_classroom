import { useState, useEffect, useCallback, useMemo } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  Chip,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  TextField,
  Button,
  Autocomplete,
  Table,
  TableContainer,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  Paper,
  Alert,
  Snackbar,
  Divider,
  Skeleton,
} from '@mui/material'
import GroupsIcon from '@mui/icons-material/Groups'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CancelIcon from '@mui/icons-material/Cancel'
import AccessTimeIcon from '@mui/icons-material/AccessTime'
import EventBusyIcon from '@mui/icons-material/EventBusy'
import FaceRetouchingNaturalIcon from '@mui/icons-material/FaceRetouchingNatural'
import MeetingRoomIcon from '@mui/icons-material/MeetingRoom'
import HowToRegIcon from '@mui/icons-material/HowToReg'
import { useAuth } from '../context/AuthContext'
import { schoolApi, attendanceApi, apiError } from '../api/client'
import useAttendanceStream from '../hooks/useAttendanceStream'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import StatCard from '../components/StatCard'

export default function Attendance() {
  const { role } = useAuth()
  const canEdit = role === 'admin' || role === 'teacher'

  const [classrooms, setClassrooms] = useState([])
  const [classroomId, setClassroomId] = useState('')
  const [students, setStudents] = useState([])

  const [rows, setRows] = useState([])
  const [classInfo, setClassInfo] = useState(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [noClass, setNoClass] = useState(false)
  const [toast, setToast] = useState(null)

  // Mark-attendance form state
  const [formStudent, setFormStudent] = useState(null)
  const [formStatus, setFormStatus] = useState('present')
  const [formDevice, setFormDevice] = useState('')
  const [submitting, setSubmitting] = useState(false)

  // Realtime face-scan recognition feed (system-wide).
  const { events, status: liveStatus } = useAttendanceStream()
  const classroomName = useMemo(() => {
    const m = {}
    classrooms.forEach((c) => (m[c.classroom_id] = c.classroom_name))
    return m
  }, [classrooms])
  const liveForRoom = useMemo(
    () => events.filter((e) => String(e.classroom_id) === String(classroomId)),
    [events, classroomId]
  )

  // Load classrooms once.
  useEffect(() => {
    let active = true
    ;(async () => {
      try {
        // Scoped: teacher sees only assigned classrooms; admin/student see all.
        const { data } = await schoolApi.getMyClassrooms()
        const list = Array.isArray(data) ? data : []
        if (!active) return
        setClassrooms(list)
        if (list.length) setClassroomId(list[0].classroom_id)
      } catch (err) {
        if (active) setError(apiError(err, 'Không tải được danh sách phòng học.'))
      }
    })()
    return () => {
      active = false
    }
  }, [])

  // The manual-attendance form may ONLY target students enrolled in the class
  // currently in session in this room — i.e. the loaded roster (`rows`), not the
  // whole school. Keep `students` mirrored to the roster so the picker is scoped.
  useEffect(() => {
    setStudents(rows)
  }, [rows])

  const loadAttendance = useCallback(async (id) => {
    if (id === '' || id == null) {
      setRows([])
      setClassInfo(null)
      return
    }
    setLoading(true)
    setError('')
    setNoClass(false)
    try {
      const { data } = await attendanceApi.list(id)
      const list = data && Array.isArray(data.students) ? data.students : Array.isArray(data) ? data : []
      setRows(list)
      setClassInfo(data?.class || null)
    } catch (err) {
      if (err?.response?.status === 404) {
        setRows([])
        setClassInfo(null)
        setNoClass(true)
      } else {
        setError(apiError(err, 'Không tải được dữ liệu điểm danh.'))
        setRows([])
        setClassInfo(null)
      }
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    loadAttendance(classroomId)
  }, [classroomId, loadAttendance])

  // Edit a record's status / delete it (staff only; only rows with a real id).
  const editStatus = async (r, newStatus) => {
    if (!r.id || newStatus === r.status) return
    try {
      await attendanceApi.update(r.id, { attendance_status: newStatus })
      setToast({ severity: 'success', msg: 'Đã cập nhật trạng thái.' })
      await loadAttendance(classroomId)
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Cập nhật thất bại.') })
    }
  }
  const removeRow = async (r) => {
    if (!r.id) return
    // eslint-disable-next-line no-alert
    if (!window.confirm(`Xoá bản ghi điểm danh của ${r.student_name}?`)) return
    try {
      await attendanceApi.remove(r.id)
      setToast({ severity: 'success', msg: 'Đã xoá bản ghi.' })
      await loadAttendance(classroomId)
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xoá thất bại.') })
    }
  }

  // When a face scan arrives for the selected classroom, refresh the list.
  useEffect(() => {
    if (liveForRoom.length) loadAttendance(classroomId)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [liveForRoom.length])

  // formDevice defaults to "<room>-cam" when classroom changes
  useEffect(() => {
    if (classroomId && classroomName[classroomId]) {
      setFormDevice(classroomName[classroomId] + '-cam')
    }
  }, [classroomId, classroomName])

  const summary = useMemo(() => {
    const total = rows.length
    const late = rows.filter((r) => r.status === 'late').length
    const presentOnly = rows.filter((r) => r.status === 'present').length
    const excused = rows.filter((r) => r.status === 'excused').length
    const absent = rows.filter((r) => r.status === 'absent').length
    // "Có mặt" = ai có mặt dù đúng giờ hay muộn
    return { total, present: presentOnly + late, late, excused, absent }
  }, [rows])

  const handleSubmit = async () => {
    if (!formStudent || classroomId === '') {
      setToast({ severity: 'warning', msg: 'Vui lòng chọn sinh viên và phòng học.' })
      return
    }
    setSubmitting(true)
    try {
      await attendanceApi.create({
        student_id: Number(formStudent.student_id),
        classroom_id: Number(classroomId),
        attendance_status: formStatus,
        device_id: formDevice,
      })
      setToast({ severity: 'success', msg: 'Đã ghi nhận điểm danh.' })
      setFormStudent(null)
      await loadAttendance(classroomId)
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Ghi nhận điểm danh thất bại.') })
    } finally {
      setSubmitting(false)
    }
  }

  const classroomSelect = (
    <FormControl size="small" sx={{ minWidth: 240 }}>
      <InputLabel id="classroom-select-label">Phòng học</InputLabel>
      <Select
        labelId="classroom-select-label"
        label="Phòng học"
        value={classrooms.length ? classroomId : ''}
        onChange={(e) => setClassroomId(e.target.value)}
        disabled={!classrooms.length}
      >
        {classrooms.map((c) => (
          <MenuItem key={c.classroom_id} value={c.classroom_id}>
            {c.classroom_name}
          </MenuItem>
        ))}
      </Select>
    </FormControl>
  )

  return (
    <Box>
      <PageHeader
        title="Điểm danh"
        subtitle="Nhận diện khuôn mặt & điểm danh thời gian thực"
        action={classroomSelect}
      />

      {/* KPI summary */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(5, 1fr)' }, mb: 3 }}>
        <StatCard icon={<GroupsIcon />} value={summary.total} label="Sĩ số" color="#2563eb" />
        <StatCard icon={<CheckCircleIcon />} value={summary.present} label="Có mặt" color="#16a34a" />
        <StatCard icon={<AccessTimeIcon />} value={summary.late} label="Đi muộn" color="#ea580c" />
        <StatCard icon={<EventBusyIcon />} value={summary.excused} label="Có phép" color="#0891b2" />
        <StatCard icon={<CancelIcon />} value={summary.absent} label="Vắng" color="#dc2626" />
      </Box>

      {/* Realtime face-recognition feed */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Stack direction="row" alignItems="center" spacing={1} mb={1}>
            <FaceRetouchingNaturalIcon color="primary" />
            <Typography variant="h6">
              Nhận diện khuôn mặt (thời gian thực){classroomName[classroomId] ? ` · ${classroomName[classroomId]}` : ''}
            </Typography>
            <Chip
              size="small"
              label={liveStatus === 'open' ? 'Đang nhận' : 'Ngoại tuyến'}
              color={liveStatus === 'open' ? 'success' : 'default'}
              variant="outlined"
            />
          </Stack>
          <Divider sx={{ mb: 1 }} />
          {liveForRoom.length === 0 ? (
            <EmptyState
              dense
              icon={<FaceRetouchingNaturalIcon />}
              title="Đang chờ camera nhận diện"
              description="Hệ thống sẽ tự động cập nhật khi camera nhận diện được học sinh trong phòng này."
            />
          ) : (
            <Stack spacing={1} sx={{ maxHeight: 240, overflow: 'auto' }}>
              {liveForRoom.slice(0, 15).map((e, i) => (
                <Stack
                  key={`${e.student_id}-${e._ts}-${i}`}
                  direction="row"
                  alignItems="center"
                  spacing={1.5}
                  flexWrap="wrap"
                  sx={{
                    px: 1.5,
                    py: 1,
                    borderRadius: 1.5,
                    bgcolor: 'action.hover',
                  }}
                >
                  {e.face_image && (
                    <Box
                      component="img"
                      src={`data:image/jpeg;base64,${e.face_image}`}
                      alt={e.student_name}
                      sx={{ width: 44, height: 44, borderRadius: 1, objectFit: 'cover', flexShrink: 0 }}
                    />
                  )}
                  {e.attendance_status === 'late' ? (
                    <Chip size="small" color="warning" label="⏱ Đi muộn" />
                  ) : (
                    <Chip size="small" color="success" label="✓ Có mặt" />
                  )}
                  <Typography variant="body2" fontWeight={600}>
                    {e.student_name}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    MSSV {e.mssv} · {e.subject} · {e.detection_time}
                  </Typography>
                </Stack>
              ))}
            </Stack>
          )}
        </CardContent>
      </Card>

      {/* Mark-attendance form (staff only) — scoped to the ongoing class roster */}
      {canEdit && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" spacing={1} mb={2} flexWrap="wrap">
              <Typography variant="h6">Ghi nhận điểm danh thủ công</Typography>
              {classInfo && (
                <Chip size="small" variant="outlined" color="primary"
                  label={`${classInfo.subject}${classInfo.period ? ` · Tiết ${classInfo.period}` : ''}${classInfo.time ? ` · ${classInfo.time}` : ''}`} />
              )}
            </Stack>
            {noClass ? (
              <Alert severity="info">Hiện không có tiết học đang diễn ra trong phòng này — không thể ghi nhận điểm danh.</Alert>
            ) : (
            <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }}>
              <Autocomplete
                size="small"
                sx={{ minWidth: 280 }}
                options={students}
                value={formStudent}
                onChange={(_, v) => setFormStudent(v)}
                getOptionLabel={(s) => (s ? `${s.mssv || s.student_id} - ${s.student_name}` : '')}
                isOptionEqualToValue={(o, v) => o.student_id === v.student_id}
                noOptionsText="Không có sinh viên trong lớp này"
                renderInput={(params) => <TextField {...params} label="Sinh viên trong lớp (MSSV/tên)" />}
              />
              <FormControl size="small" sx={{ minWidth: 160 }}>
                <InputLabel id="status-select-label">Trạng thái</InputLabel>
                <Select
                  labelId="status-select-label"
                  label="Trạng thái"
                  value={formStatus}
                  onChange={(e) => setFormStatus(e.target.value)}
                >
                  <MenuItem value="present">Có mặt</MenuItem>
                  <MenuItem value="late">Đi muộn</MenuItem>
                  <MenuItem value="excused">Có phép</MenuItem>
                  <MenuItem value="absent">Vắng</MenuItem>
                </Select>
              </FormControl>
              <TextField
                size="small"
                label="Mã thiết bị"
                value={formDevice}
                onChange={(e) => setFormDevice(e.target.value)}
                sx={{ minWidth: 160 }}
              />
              <Button variant="contained" onClick={handleSubmit} disabled={submitting || !classrooms.length || !students.length}>
                {submitting ? 'Đang lưu...' : 'Ghi nhận'}
              </Button>
            </Stack>
            )}
          </CardContent>
        </Card>
      )}

      {/* Attendance table */}
      <Card>
        <CardContent>
          <Stack direction="row" alignItems="center" spacing={1} mb={2} flexWrap="wrap">
            <Typography variant="h6">Danh sách điểm danh</Typography>
            {classInfo && (
              <Chip size="small" variant="outlined" color="primary"
                label={`${classInfo.subject}${classInfo.period ? ` · Tiết ${classInfo.period}` : ''}${classInfo.time ? ` · ${classInfo.time}` : ''}`} />
            )}
          </Stack>
          {loading ? (
            <Stack spacing={1}>
              {Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} variant="rounded" height={44} />
              ))}
            </Stack>
          ) : error ? (
            <Alert severity="error">{error}</Alert>
          ) : noClass ? (
            <EmptyState
              dense
              icon={<MeetingRoomIcon />}
              title="Không có lớp đang diễn ra"
              description="Hiện không có lớp đang diễn ra trong phòng này."
            />
          ) : rows.length === 0 ? (
            <EmptyState
              dense
              icon={<HowToRegIcon />}
              title="Chưa có dữ liệu điểm danh"
              description="Dữ liệu sẽ xuất hiện khi học sinh được nhận diện hoặc điểm danh thủ công."
            />
          ) : (
            <TableContainer component={Paper} variant="outlined" sx={{ maxHeight: 520 }}>
              <Table size="small" stickyHeader>
                <TableHead>
                  <TableRow>
                    <TableCell>MSSV</TableCell>
                    <TableCell>Họ tên</TableCell>
                    <TableCell>Trạng thái</TableCell>
                    <TableCell>SĐT</TableCell>
                    <TableCell>Email</TableCell>
                    {canEdit && <TableCell align="right">Thao tác</TableCell>}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {rows.map((r) => (
                    <TableRow key={r.student_id} hover>
                      <TableCell>{r.mssv || r.student_id}</TableCell>
                      <TableCell>{r.student_name}</TableCell>
                      <TableCell>
                        {r.status === 'present' ? (
                          <Chip size="small" color="success" label="Có mặt" />
                        ) : r.status === 'late' ? (
                          <Chip size="small" color="warning" label="Đi muộn" />
                        ) : r.status === 'excused' ? (
                          <Chip size="small" color="info" label="Có phép" />
                        ) : (
                          <Chip size="small" color="error" label="Vắng" />
                        )}
                      </TableCell>
                      <TableCell>{r.phone || '—'}</TableCell>
                      <TableCell>{r.email || '—'}</TableCell>
                      {canEdit && (
                        <TableCell align="right">
                          {r.id ? (
                            <Stack direction="row" spacing={1} justifyContent="flex-end" alignItems="center">
                              <Select
                                size="small"
                                value={r.status}
                                onChange={(e) => editStatus(r, e.target.value)}
                                sx={{ minWidth: 120 }}
                              >
                                <MenuItem value="present">Có mặt</MenuItem>
                                <MenuItem value="late">Đi muộn</MenuItem>
                                <MenuItem value="excused">Có phép</MenuItem>
                                <MenuItem value="absent">Vắng</MenuItem>
                              </Select>
                              <Button size="small" color="error" onClick={() => removeRow(r)}>Xoá</Button>
                            </Stack>
                          ) : (
                            <Typography variant="caption" color="text.secondary">—</Typography>
                          )}
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>

      <Snackbar
        open={!!toast}
        autoHideDuration={3000}
        onClose={() => setToast(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        {toast ? (
          <Alert severity={toast.severity} onClose={() => setToast(null)}>
            {toast.msg}
          </Alert>
        ) : null}
      </Snackbar>
    </Box>
  )
}
