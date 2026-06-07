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
} from '@mui/material'
import SearchIcon from '@mui/icons-material/Search'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import RefreshIcon from '@mui/icons-material/Refresh'
import LinkOffIcon from '@mui/icons-material/LinkOff'
import { useAuth } from '../context/AuthContext'
import { schoolApi, classroomTeacherApi, apiError } from '../api/client'

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
]

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
