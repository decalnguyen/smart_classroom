import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Stack,
  Tabs,
  Tab,
  Table,
  TableContainer,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  Paper,
  IconButton,
  Tooltip,
  CircularProgress,
  Alert,
  Snackbar,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  TablePagination,
  InputAdornment,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Autocomplete,
  Chip,
} from '@mui/material'
import SearchIcon from '@mui/icons-material/Search'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import RefreshIcon from '@mui/icons-material/Refresh'
import LinkOffIcon from '@mui/icons-material/LinkOff'
import { useAuth } from '../context/AuthContext'
import { schoolApi, classroomTeacherApi, holidayApi, makeupApi, classApi, apiError } from '../api/client'

// minutes-from-midnight <-> HH:MM helpers (period times are stored as minutes).
const minToHHMM = (m) => `${String(Math.floor((m || 0) / 60)).padStart(2, '0')}:${String((m || 0) % 60).padStart(2, '0')}`
const hhmmToMin = (s) => {
  const [h, m] = String(s || '').split(':').map(Number)
  return (h || 0) * 60 + (m || 0)
}
const classLabel = (c) =>
  `#${c.class_id} · ${c.classroom_name || ''} — ${c.subject || ''} (${c.day_of_week || ''} ${minToHHMM(c.start_min)}-${minToHHMM(c.end_min)})`

// ---- Tab configuration: each tab declares its key, label, columns + api ----
const TABS = [
  {
    label: 'Tòa nhà',
    idKey: 'building_id',
    columns: [
      { key: 'building_id', label: 'Mã', editable: false },
      { key: 'building_name', label: 'Tên tòa nhà', type: 'text', required: true },
      { key: 'location', label: 'Vị trí', type: 'text', required: true },
    ],
    api: {
      list: schoolApi.getBuildings,
      create: schoolApi.createBuilding,
      update: schoolApi.updateBuilding,
      remove: schoolApi.deleteBuilding,
    },
  },
  {
    label: 'Phòng học',
    idKey: 'classroom_id',
    columns: [
      { key: 'classroom_id', label: 'Mã', editable: false },
      { key: 'classroom_name', label: 'Tên phòng học', type: 'text', required: true },
      { key: 'subject', label: 'Môn học', type: 'text', required: true },
      { key: 'building_id', label: 'Mã tòa nhà', type: 'number', required: true },
      { key: 'capacity', label: 'Sức chứa', type: 'number' },
    ],
    api: {
      list: schoolApi.getClassrooms,
      create: schoolApi.createClassroom,
      update: schoolApi.updateClassroom,
      remove: schoolApi.deleteClassroom,
    },
  },
  {
    label: 'Học sinh',
    idKey: 'student_id',
    columns: [
      { key: 'student_id', label: 'Mã', editable: false },
      { key: 'mssv', label: 'MSSV', type: 'text', required: true },
      { key: 'student_name', label: 'Họ và tên', type: 'text', required: true },
      { key: 'age', label: 'Tuổi', type: 'number', required: true },
      { key: 'phone', label: 'Số điện thoại', type: 'text' },
      { key: 'email', label: 'Email', type: 'text' },
    ],
    api: {
      list: schoolApi.getStudents,
      create: schoolApi.createStudent,
      update: schoolApi.updateStudent,
      remove: schoolApi.deleteStudent,
    },
  },
  {
    label: 'Giáo viên',
    idKey: 'teacher_id',
    columns: [
      { key: 'teacher_id', label: 'Mã', editable: false },
      { key: 'teacher_name', label: 'Họ và tên', type: 'text', required: true },
      { key: 'subject', label: 'Môn giảng dạy', type: 'text', required: true },
      { key: 'account_id', label: 'Tài khoản (account_id)', type: 'text' },
    ],
    api: {
      list: schoolApi.getTeachers,
      create: schoolApi.createTeacher,
      update: schoolApi.updateTeacher,
      remove: schoolApi.deleteTeacher,
    },
  },
  { label: 'Phân công GV', custom: 'assignments' },
  { label: 'Ngày lễ', custom: 'holidays' },
  { label: 'Buổi bù', custom: 'makeups' },
  { label: 'Ghi danh lớp', custom: 'enroll' },
]

// Holiday manager (attendance is skipped on these dates).
function HolidaysPanel({ notify }) {
  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [date, setDate] = useState('')
  const [name, setName] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const { data } = await holidayApi.list()
      setRows(Array.isArray(data) ? data : [])
    } catch (err) {
      notify('error', apiError(err, 'Không tải được ngày lễ.'))
    } finally {
      setLoading(false)
    }
  }, [notify])
  useEffect(() => { load() }, [load])

  const add = async () => {
    if (!date) { notify('warning', 'Chọn ngày.'); return }
    try {
      await holidayApi.create(date, name)
      notify('success', 'Đã thêm ngày lễ.')
      setDate(''); setName('')
      await load()
    } catch (err) { notify('error', apiError(err, 'Thêm thất bại.')) }
  }
  const remove = async (h) => {
    if (!window.confirm(`Xoá ngày lễ ${h.date}?`)) return
    try { await holidayApi.remove(h.id); notify('success', 'Đã xoá.'); await load() }
    catch (err) { notify('error', apiError(err, 'Xoá thất bại.')) }
  }

  return (
    <Box>
      <Typography variant="h6" mb={2}>Ngày lễ / nghỉ (không tính điểm danh)</Typography>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }} mb={2}>
        <TextField size="small" type="date" label="Ngày" value={date} onChange={(e) => setDate(e.target.value)} InputLabelProps={{ shrink: true }} />
        <TextField size="small" label="Tên ngày lễ" value={name} onChange={(e) => setName(e.target.value)} sx={{ minWidth: 220 }} />
        <Button variant="contained" startIcon={<AddIcon />} onClick={add}>Thêm</Button>
      </Stack>
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}><CircularProgress /></Box>
      ) : rows.length === 0 ? (
        <Alert severity="info">Chưa có ngày lễ nào.</Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead><TableRow><TableCell>Ngày</TableCell><TableCell>Tên</TableCell><TableCell align="right">Thao tác</TableCell></TableRow></TableHead>
            <TableBody>
              {rows.map((h) => (
                <TableRow key={h.id} hover>
                  <TableCell>{h.date}</TableCell>
                  <TableCell>{h.name}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="Xoá"><IconButton size="small" color="error" onClick={() => remove(h)}><DeleteIcon fontSize="small" /></IconButton></Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  )
}

// Makeup-session manager (buổi bù — an extra class on a specific date).
function MakeupsPanel({ notify }) {
  const [rows, setRows] = useState([])
  const [classes, setClasses] = useState([])
  const [loading, setLoading] = useState(true)
  const [classId, setClassId] = useState('')
  const [date, setDate] = useState('')
  const [start, setStart] = useState('07:30')
  const [end, setEnd] = useState('09:00')
  const [note, setNote] = useState('')

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [m, cl] = await Promise.all([makeupApi.list(), classApi.listClasses()])
      setRows(Array.isArray(m.data) ? m.data : [])
      setClasses(Array.isArray(cl.data) ? cl.data : [])
    } catch (err) {
      notify('error', apiError(err, 'Không tải được buổi bù.'))
    } finally {
      setLoading(false)
    }
  }, [notify])
  useEffect(() => { load() }, [load])

  const classText = (cid) => {
    const c = classes.find((x) => String(x.class_id) === String(cid))
    return c ? classLabel(c) : `Lớp #${cid}`
  }

  const add = async () => {
    if (!classId || !date) { notify('warning', 'Chọn lớp và ngày.'); return }
    try {
      await makeupApi.create({ class_id: Number(classId), date, start_min: hhmmToMin(start), end_min: hhmmToMin(end), note })
      notify('success', 'Đã thêm buổi bù.')
      setNote('')
      await load()
    } catch (err) { notify('error', apiError(err, 'Thêm thất bại.')) }
  }
  const remove = async (m) => {
    if (!window.confirm(`Xoá buổi bù ngày ${m.date}?`)) return
    try { await makeupApi.remove(m.id); notify('success', 'Đã xoá.'); await load() }
    catch (err) { notify('error', apiError(err, 'Xoá thất bại.')) }
  }

  return (
    <Box>
      <Typography variant="h6" mb={2}>Buổi bù (tiết học bổ sung vào một ngày cụ thể)</Typography>
      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }} mb={2} flexWrap="wrap" useFlexGap>
        <FormControl size="small" sx={{ minWidth: 320 }}>
          <InputLabel>Lớp</InputLabel>
          <Select label="Lớp" value={classId} onChange={(e) => setClassId(e.target.value)}>
            {classes.map((c) => <MenuItem key={c.class_id} value={c.class_id}>{classLabel(c)}</MenuItem>)}
          </Select>
        </FormControl>
        <TextField size="small" type="date" label="Ngày" value={date} onChange={(e) => setDate(e.target.value)} InputLabelProps={{ shrink: true }} />
        <TextField size="small" type="time" label="Bắt đầu" value={start} onChange={(e) => setStart(e.target.value)} InputLabelProps={{ shrink: true }} />
        <TextField size="small" type="time" label="Kết thúc" value={end} onChange={(e) => setEnd(e.target.value)} InputLabelProps={{ shrink: true }} />
        <TextField size="small" label="Ghi chú" value={note} onChange={(e) => setNote(e.target.value)} sx={{ minWidth: 180 }} />
        <Button variant="contained" startIcon={<AddIcon />} onClick={add}>Thêm</Button>
      </Stack>
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}><CircularProgress /></Box>
      ) : rows.length === 0 ? (
        <Alert severity="info">Chưa có buổi bù nào.</Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead><TableRow><TableCell>Ngày</TableCell><TableCell>Lớp</TableCell><TableCell>Giờ</TableCell><TableCell>Ghi chú</TableCell><TableCell align="right">Thao tác</TableCell></TableRow></TableHead>
            <TableBody>
              {rows.map((m) => (
                <TableRow key={m.id} hover>
                  <TableCell>{m.date}</TableCell>
                  <TableCell>{classText(m.class_id)}</TableCell>
                  <TableCell>{minToHHMM(m.start_min)}–{minToHHMM(m.end_min)}</TableCell>
                  <TableCell>{m.note}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="Xoá"><IconButton size="small" color="error" onClick={() => remove(m)}><DeleteIcon fontSize="small" /></IconButton></Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  )
}

// Class-enrollment manager: pick a class, view its roster, add/remove students.
function EnrollPanel({ notify }) {
  const [classes, setClasses] = useState([])
  const [classId, setClassId] = useState('')
  const [roster, setRoster] = useState([])
  const [students, setStudents] = useState([])
  const [picked, setPicked] = useState(null)
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)

  const loadBase = useCallback(async () => {
    setLoading(true)
    try {
      const [cl, st] = await Promise.all([classApi.listClasses(), schoolApi.getStudents()])
      setClasses(Array.isArray(cl.data) ? cl.data : [])
      setStudents(Array.isArray(st.data) ? st.data : [])
    } catch (err) {
      notify('error', apiError(err, 'Không tải được dữ liệu lớp.'))
    } finally {
      setLoading(false)
    }
  }, [notify])
  useEffect(() => { loadBase() }, [loadBase])

  const loadRoster = useCallback(async (cid) => {
    if (!cid) { setRoster([]); return }
    try { const { data } = await classApi.getRoster(cid); setRoster(Array.isArray(data) ? data : []) }
    catch (err) { notify('error', apiError(err, 'Không tải được sĩ số.')) }
  }, [notify])
  useEffect(() => { loadRoster(classId) }, [classId, loadRoster])

  const selected = classes.find((c) => String(c.class_id) === String(classId))
  const enrolledIds = new Set(roster.map((s) => s.student_id))
  const options = students.filter((s) => !enrolledIds.has(s.student_id))

  const add = async () => {
    if (!classId || !picked) { notify('warning', 'Chọn lớp và học sinh.'); return }
    setBusy(true)
    try {
      await classApi.enrollStudent(classId, picked.student_id)
      notify('success', 'Đã ghi danh.')
      setPicked(null)
      await loadRoster(classId); await loadBase()
    } catch (err) { notify('error', apiError(err, 'Ghi danh thất bại.')) }
    finally { setBusy(false) }
  }
  const remove = async (s) => {
    if (!window.confirm(`Huỷ ghi danh ${s.student_name}?`)) return
    try { await classApi.unenrollStudent(classId, s.student_id); notify('success', 'Đã huỷ ghi danh.'); await loadRoster(classId); await loadBase() }
    catch (err) { notify('error', apiError(err, 'Huỷ thất bại.')) }
  }

  return (
    <Box>
      <Typography variant="h6" mb={2}>Ghi danh học sinh vào lớp</Typography>
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}><CircularProgress /></Box>
      ) : (
        <>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} alignItems={{ md: 'center' }} mb={2} flexWrap="wrap" useFlexGap>
            <FormControl size="small" sx={{ minWidth: 320 }}>
              <InputLabel>Lớp</InputLabel>
              <Select label="Lớp" value={classId} onChange={(e) => setClassId(e.target.value)}>
                {classes.map((c) => (
                  <MenuItem key={c.class_id} value={c.class_id}>
                    {classLabel(c)} — {c.enrolled}/{c.capacity || '∞'}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            {selected && (
              <Chip
                color={selected.capacity > 0 && selected.enrolled >= selected.capacity ? 'error' : 'default'}
                label={`Sĩ số: ${selected.enrolled}/${selected.capacity || '∞'}`}
              />
            )}
          </Stack>

          {classId && (
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }} mb={2}>
              <Autocomplete
                size="small"
                sx={{ minWidth: 320 }}
                options={options}
                value={picked}
                onChange={(_, v) => setPicked(v)}
                getOptionLabel={(s) => (s ? `${s.mssv || s.student_id} — ${s.student_name}` : '')}
                isOptionEqualToValue={(o, v) => o.student_id === v.student_id}
                renderInput={(params) => <TextField {...params} label="Thêm học sinh" />}
              />
              <Button variant="contained" startIcon={<AddIcon />} disabled={busy || !picked} onClick={add}>Ghi danh</Button>
            </Stack>
          )}

          {!classId ? (
            <Alert severity="info">Chọn một lớp để xem và quản lý sĩ số.</Alert>
          ) : roster.length === 0 ? (
            <Alert severity="info">Lớp chưa có học sinh nào.</Alert>
          ) : (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead><TableRow><TableCell>MSSV</TableCell><TableCell>Họ và tên</TableCell><TableCell align="right">Thao tác</TableCell></TableRow></TableHead>
                <TableBody>
                  {roster.map((s) => (
                    <TableRow key={s.student_id} hover>
                      <TableCell>{s.mssv}</TableCell>
                      <TableCell>{s.student_name}</TableCell>
                      <TableCell align="right">
                        <Tooltip title="Huỷ ghi danh"><IconButton size="small" color="error" onClick={() => remove(s)}><LinkOffIcon fontSize="small" /></IconButton></Tooltip>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </>
      )}
    </Box>
  )
}

// Teacher ↔ classroom assignment manager.
function AssignmentPanel({ notify }) {
  const [rows, setRows] = useState([])
  const [classrooms, setClassrooms] = useState([])
  const [teachers, setTeachers] = useState([])
  const [loading, setLoading] = useState(true)
  const [classroomId, setClassroomId] = useState('')
  const [teacherId, setTeacherId] = useState('')
  const [saving, setSaving] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const [a, cr, te] = await Promise.all([
        classroomTeacherApi.list(),
        schoolApi.getClassrooms(),
        schoolApi.getTeachers(),
      ])
      setRows(Array.isArray(a.data) ? a.data : [])
      setClassrooms(Array.isArray(cr.data) ? cr.data : [])
      setTeachers(Array.isArray(te.data) ? te.data : [])
    } catch (err) {
      notify('error', apiError(err, 'Không tải được phân công.'))
    } finally {
      setLoading(false)
    }
  }, [notify])

  useEffect(() => { load() }, [load])

  const assign = async () => {
    if (!classroomId || !teacherId) {
      notify('warning', 'Chọn phòng học và giáo viên.')
      return
    }
    setSaving(true)
    try {
      await classroomTeacherApi.assign(Number(classroomId), Number(teacherId))
      notify('success', 'Đã phân công.')
      await load()
    } catch (err) {
      notify('error', apiError(err, 'Phân công thất bại.'))
    } finally {
      setSaving(false)
    }
  }

  const remove = async (r) => {
    if (!window.confirm(`Gỡ phân công ${r.teacher_name} khỏi ${r.classroom_name}?`)) return
    try {
      await classroomTeacherApi.remove(r.classroom_id, r.teacher_id)
      notify('success', 'Đã gỡ phân công.')
      await load()
    } catch (err) {
      notify('error', apiError(err, 'Gỡ phân công thất bại.'))
    }
  }

  return (
    <Box>
      <Typography variant="h6" mb={2}>Phân công giáo viên ↔ lớp học</Typography>
      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} alignItems={{ sm: 'center' }} mb={2}>
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel id="cr-l">Phòng học</InputLabel>
          <Select labelId="cr-l" label="Phòng học" value={classroomId} onChange={(e) => setClassroomId(e.target.value)}>
            {classrooms.map((c) => <MenuItem key={c.classroom_id} value={c.classroom_id}>{c.classroom_name}</MenuItem>)}
          </Select>
        </FormControl>
        <FormControl size="small" sx={{ minWidth: 220 }}>
          <InputLabel id="te-l">Giáo viên</InputLabel>
          <Select labelId="te-l" label="Giáo viên" value={teacherId} onChange={(e) => setTeacherId(e.target.value)}>
            {teachers.map((t) => <MenuItem key={t.teacher_id} value={t.teacher_id}>{t.teacher_name} ({t.subject})</MenuItem>)}
          </Select>
        </FormControl>
        <Button variant="contained" startIcon={<AddIcon />} onClick={assign} disabled={saving}>Phân công</Button>
      </Stack>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}><CircularProgress /></Box>
      ) : rows.length === 0 ? (
        <Alert severity="info">Chưa có phân công nào.</Alert>
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Phòng học</TableCell>
                <TableCell>Giáo viên</TableCell>
                <TableCell align="right">Thao tác</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rows.map((r) => (
                <TableRow key={`${r.classroom_id}-${r.teacher_id}`} hover>
                  <TableCell>{r.classroom_name}</TableCell>
                  <TableCell>{r.teacher_name}</TableCell>
                  <TableCell align="right">
                    <Tooltip title="Gỡ phân công">
                      <IconButton size="small" color="error" onClick={() => remove(r)}><LinkOffIcon fontSize="small" /></IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  )
}

// Build an empty form object from the editable columns of a tab.
function blankForm(cfg) {
  const f = {}
  cfg.columns
    .filter((c) => c.editable !== false)
    .forEach((c) => {
      f[c.key] = ''
    })
  return f
}

// ---- Generic CRUD table for one entity (one tab) ----
function CrudTable({ cfg, notify }) {
  const editableCols = cfg.columns.filter((c) => c.editable !== false)

  const [rows, setRows] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const [dialogOpen, setDialogOpen] = useState(false)
  const [editing, setEditing] = useState(null) // null = create mode
  const [form, setForm] = useState(() => blankForm(cfg))
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState(null)

  const [confirm, setConfirm] = useState(null) // row pending deletion
  const [deleting, setDeleting] = useState(false)

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(0)
  const [rowsPerPage, setRowsPerPage] = useState(10)

  // Client-side filter across all columns + pagination (handles ~700 rows).
  const filtered = search.trim()
    ? rows.filter((r) =>
        cfg.columns.some((c) => String(r[c.key] ?? '').toLowerCase().includes(search.trim().toLowerCase()))
      )
    : rows
  const paged = filtered.slice(page * rowsPerPage, page * rowsPerPage + rowsPerPage)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const { data } = await cfg.api.list()
      setRows(Array.isArray(data) ? data : [])
    } catch {
      setError('Không thể tải dữ liệu. Vui lòng thử lại.')
      setRows([])
    } finally {
      setLoading(false)
    }
  }, [cfg])

  useEffect(() => {
    load()
  }, [load])

  const openCreate = () => {
    setEditing(null)
    setForm(blankForm(cfg))
    setFormError(null)
    setDialogOpen(true)
  }

  const openEdit = (row) => {
    setEditing(row)
    const f = {}
    editableCols.forEach((c) => {
      f[c.key] = row[c.key] ?? ''
    })
    setForm(f)
    setFormError(null)
    setDialogOpen(true)
  }

  const closeDialog = () => {
    if (saving) return
    setDialogOpen(false)
  }

  const handleField = (key) => (e) => {
    setForm((prev) => ({ ...prev, [key]: e.target.value }))
  }

  const buildPayload = () => {
    const payload = {}
    editableCols.forEach((c) => {
      const raw = form[c.key]
      if (c.type === 'number') {
        payload[c.key] = raw === '' || raw === null ? null : Number(raw)
      } else {
        payload[c.key] = raw
      }
    })
    return payload
  }

  const validate = () => {
    for (const c of editableCols) {
      const v = form[c.key]
      if (c.required && (v === '' || v === null || v === undefined)) {
        return `Vui lòng nhập "${c.label}".`
      }
      if (c.type === 'number' && v !== '' && Number.isNaN(Number(v))) {
        return `"${c.label}" phải là số.`
      }
    }
    return null
  }

  const handleSave = async () => {
    const v = validate()
    if (v) {
      setFormError(v)
      return
    }
    setSaving(true)
    setFormError(null)
    try {
      const payload = buildPayload()
      if (editing) {
        await cfg.api.update(editing[cfg.idKey], payload)
        notify('success', 'Cập nhật thành công.')
      } else {
        await cfg.api.create(payload)
        notify('success', 'Thêm mới thành công.')
      }
      setDialogOpen(false)
      await load()
    } catch {
      setFormError('Thao tác thất bại. Vui lòng kiểm tra lại.')
      notify('error', editing ? 'Cập nhật thất bại.' : 'Thêm mới thất bại.')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!confirm) return
    setDeleting(true)
    try {
      await cfg.api.remove(confirm[cfg.idKey])
      notify('success', 'Đã xoá thành công.')
      setConfirm(null)
      await load()
    } catch {
      notify('error', 'Xoá thất bại.')
    } finally {
      setDeleting(false)
    }
  }

  return (
    <Box>
      <Stack direction="row" justifyContent="space-between" alignItems="center" mb={2} flexWrap="wrap" gap={1}>
        <Typography variant="h6">
          {cfg.label} <Typography component="span" variant="body2" color="text.secondary">({filtered.length})</Typography>
        </Typography>
        <Stack direction="row" spacing={1} alignItems="center">
          <TextField
            size="small"
            placeholder="Tìm kiếm..."
            value={search}
            onChange={(e) => {
              setSearch(e.target.value)
              setPage(0)
            }}
            InputProps={{ startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment> }}
          />
          <Button startIcon={<RefreshIcon />} onClick={load} disabled={loading}>
            Làm mới
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
            Thêm
          </Button>
        </Stack>
      </Stack>

      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
          <CircularProgress />
        </Box>
      ) : error ? (
        <Alert
          severity="error"
          action={
            <Button color="inherit" size="small" onClick={load}>
              Thử lại
            </Button>
          }
        >
          {error}
        </Alert>
      ) : rows.length === 0 ? (
        <Alert severity="info">Chưa có dữ liệu. Nhấn "Thêm" để tạo mới.</Alert>
      ) : (
        <>
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                {cfg.columns.map((c) => (
                  <TableCell key={c.key}>{c.label}</TableCell>
                ))}
                <TableCell align="right">Thao tác</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {paged.map((row) => (
                <TableRow key={row[cfg.idKey]} hover>
                  {cfg.columns.map((c) => (
                    <TableCell key={c.key}>{row[c.key] ?? '—'}</TableCell>
                  ))}
                  <TableCell align="right">
                    <Tooltip title="Sửa">
                      <IconButton size="small" onClick={() => openEdit(row)}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Xoá">
                      <IconButton size="small" color="error" onClick={() => setConfirm(row)}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={filtered.length}
          page={page}
          onPageChange={(_, p) => setPage(p)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10))
            setPage(0)
          }}
          rowsPerPageOptions={[10, 25, 50, 100]}
          labelRowsPerPage="Số dòng/trang"
        />
        </>
      )}

      {/* Create / edit dialog */}
      <Dialog open={dialogOpen} onClose={closeDialog} fullWidth maxWidth="sm">
        <DialogTitle>{editing ? `Sửa ${cfg.label.toLowerCase()}` : `Thêm ${cfg.label.toLowerCase()}`}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            {formError && <Alert severity="error">{formError}</Alert>}
            {editableCols.map((c) => (
              <TextField
                key={c.key}
                label={c.label}
                type={c.type === 'number' ? 'number' : 'text'}
                value={form[c.key] ?? ''}
                onChange={handleField(c.key)}
                required={!!c.required}
                fullWidth
                autoFocus={c === editableCols[0]}
              />
            ))}
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={saving}>
            Huỷ
          </Button>
          <Button variant="contained" onClick={handleSave} disabled={saving}>
            {saving ? <CircularProgress size={20} /> : editing ? 'Lưu' : 'Thêm'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Delete confirmation */}
      <Dialog open={!!confirm} onClose={() => !deleting && setConfirm(null)}>
        <DialogTitle>Xác nhận xoá</DialogTitle>
        <DialogContent>
          <Typography>
            Bạn có chắc chắn muốn xoá{' '}
            <strong>
              {confirm ? confirm[editableCols[0]?.key] ?? confirm[cfg.idKey] : ''}
            </strong>
            ? Hành động này không thể hoàn tác.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirm(null)} disabled={deleting}>
            Huỷ
          </Button>
          <Button color="error" variant="contained" onClick={handleDelete} disabled={deleting}>
            {deleting ? <CircularProgress size={20} /> : 'Xoá'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

export default function Admin() {
  const { role } = useAuth()
  const [tab, setTab] = useState(0)
  const [toast, setToast] = useState(null)

  const notify = useCallback((severity, msg) => {
    setToast({ severity, msg })
  }, [])

  if (role !== 'admin') {
    return (
      <Box>
        <Typography variant="h4" mb={2}>
          Quản trị hệ thống
        </Typography>
        <Alert severity="warning">Chỉ quản trị viên mới có quyền truy cập trang này.</Alert>
      </Box>
    )
  }

  return (
    <Box>
      <Typography variant="h4" mb={2}>
        Quản trị hệ thống
      </Typography>

      <Card>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v)}
          variant="scrollable"
          scrollButtons="auto"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          {TABS.map((t) => (
            <Tab key={t.label} label={t.label} />
          ))}
        </Tabs>
        <CardContent>
          {/* key forces remount per tab so each panel loads its own data */}
          {TABS[tab].custom === 'assignments' ? (
            <AssignmentPanel key="assignments" notify={notify} />
          ) : TABS[tab].custom === 'holidays' ? (
            <HolidaysPanel key="holidays" notify={notify} />
          ) : TABS[tab].custom === 'makeups' ? (
            <MakeupsPanel key="makeups" notify={notify} />
          ) : TABS[tab].custom === 'enroll' ? (
            <EnrollPanel key="enroll" notify={notify} />
          ) : (
            <CrudTable key={TABS[tab].label} cfg={TABS[tab]} notify={notify} />
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
          <Alert severity={toast.severity} onClose={() => setToast(null)} variant="filled">
            {toast.msg}
          </Alert>
        ) : null}
      </Snackbar>
    </Box>
  )
}
