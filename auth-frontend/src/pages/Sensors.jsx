import { useState, useEffect, useCallback } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Stack,
  Chip,
  Snackbar,
  Alert,
  Skeleton,
  Table,
  TableContainer,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  Paper,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
} from '@mui/material'
import LightModeIcon from '@mui/icons-material/LightMode'
import ThermostatIcon from '@mui/icons-material/Thermostat'
import WaterDropIcon from '@mui/icons-material/WaterDrop'
import LocalFireDepartmentIcon from '@mui/icons-material/LocalFireDepartment'
import LightbulbIcon from '@mui/icons-material/Lightbulb'
import AcUnitIcon from '@mui/icons-material/AcUnit'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import RefreshIcon from '@mui/icons-material/Refresh'
import DevicesOtherIcon from '@mui/icons-material/DevicesOther'
import dayjs from 'dayjs'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import GaugeCard from '../components/GaugeCard'
import useSensorStream from '../hooks/useSensorStream'
import { useAuth } from '../context/AuthContext'
import { sensorApi, apiError } from '../api/client'

const EMPTY_FORM = {
  device_id: '',
  device_name: '',
  device_type: '',
  location: '',
  status: 'Active',
}

function fmtTs(ts) {
  if (!ts) return '--'
  const d = dayjs(ts)
  return d.isValid() ? d.format('DD/MM/YYYY HH:mm:ss') : String(ts)
}

export default function Sensors() {
  const { role } = useAuth()
  const isAdmin = role === 'admin'
  const canControl = role === 'admin' || role === 'teacher'

  const { latest, status } = useSensorStream()
  const online = status === 'open'

  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const [toast, setToast] = useState(null)

  // Dialog state for create/edit.
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState(null) // null = create mode
  const [form, setForm] = useState(EMPTY_FORM)
  const [saving, setSaving] = useState(false)

  // Device control toggle states.
  const [light, setLight] = useState(false)
  const [fan, setFan] = useState(false)
  const [busyCtrl, setBusyCtrl] = useState(null)

  // Latest numeric value for a metric, or null when not yet received.
  const num = (m) => {
    const v = latest[m] ? latest[m].value : null
    return v === null || v === undefined || Number.isNaN(Number(v)) ? null : Number(v)
  }

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const { data } = await sensorApi.listSensors()
      setDevices(Array.isArray(data) ? data : [])
    } catch (err) {
      setError(apiError(err, 'Không tải được danh sách thiết bị. Vui lòng thử lại.'))
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const openCreate = () => {
    setEditingId(null)
    setForm(EMPTY_FORM)
    setDialogOpen(true)
  }

  const openEdit = (d) => {
    setEditingId(d.device_id)
    setForm({
      device_id: d.device_id || '',
      device_name: d.device_name || '',
      device_type: d.device_type || '',
      location: d.location || '',
      status: d.status || 'Active',
    })
    setDialogOpen(true)
  }

  const closeDialog = () => {
    if (saving) return
    setDialogOpen(false)
  }

  const setField = (key) => (e) => setForm((f) => ({ ...f, [key]: e.target.value }))

  const submitForm = async () => {
    if (!form.device_id.trim() || !form.device_name.trim()) {
      setToast({ severity: 'warning', msg: 'Vui lòng nhập Mã và Tên thiết bị.' })
      return
    }
    setSaving(true)
    try {
      const payload = {
        device_id: form.device_id.trim(),
        device_name: form.device_name.trim(),
        device_type: form.device_type.trim(),
        location: form.location.trim(),
        status: form.status.trim() || 'Active',
      }
      if (editingId == null) {
        await sensorApi.createSensor(payload)
        setToast({ severity: 'success', msg: 'Đã thêm thiết bị.' })
      } else {
        await sensorApi.updateSensor(editingId, payload)
        setToast({ severity: 'success', msg: 'Đã cập nhật thiết bị.' })
      }
      setDialogOpen(false)
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Lưu thiết bị thất bại.') })
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async (d) => {
    // eslint-disable-next-line no-alert
    if (!window.confirm(`Xoá thiết bị "${d.device_name || d.device_id}"?`)) return
    try {
      await sensorApi.deleteSensor(d.device_id)
      setToast({ severity: 'success', msg: 'Đã xoá thiết bị.' })
      await load()
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Xoá thiết bị thất bại.') })
    }
  }

  const control = async (type, on) => {
    setBusyCtrl(type)
    try {
      const { data } = await sensorApi.setDeviceMode(type, `A101-${type}`, on ? 1 : 0)
      const label = type === 'light' ? 'đèn' : 'quạt'
      const queued = typeof data === 'string' && data.toLowerCase().includes('queued')
      if (type === 'light') setLight(on)
      else setFan(on)
      setToast({
        severity: queued ? 'info' : 'success',
        msg: queued
          ? `Lệnh ${label} đã xếp hàng (thiết bị ngoại tuyến): ${on ? 'BẬT' : 'TẮT'}`
          : `Đã gửi lệnh ${label}: ${on ? 'BẬT' : 'TẮT'}`,
      })
    } catch (err) {
      setToast({ severity: 'error', msg: apiError(err, 'Gửi lệnh thất bại.') })
    } finally {
      setBusyCtrl(null)
    }
  }

  const smokeVal = num('smoke')
  const tempVal = num('temperature')

  return (
    <Box>
      <PageHeader
        title="Cảm biến & Thiết bị"
        subtitle="Theo dõi và điều khiển thiết bị trong lớp"
        action={
          <Chip
            label={online ? 'Trực tuyến' : 'Ngoại tuyến'}
            color={online ? 'success' : 'default'}
            variant={online ? 'filled' : 'outlined'}
          />
        }
      />

      {/* Live readings as gauges */}
      <Box
        sx={{
          display: 'grid',
          gap: 2,
          gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(4, 1fr)' },
          mb: 3,
        }}
      >
        <GaugeCard
          label="Ánh sáng"
          value={num('light')}
          unit="lux"
          min={0}
          max={1000}
          color="#f59e0b"
          icon={<LightModeIcon fontSize="small" />}
        />
        <GaugeCard
          label="Nhiệt độ"
          value={tempVal}
          unit="°C"
          min={0}
          max={60}
          color="#ea580c"
          danger={tempVal != null && tempVal >= 50}
          icon={<ThermostatIcon fontSize="small" />}
        />
        <GaugeCard
          label="Độ ẩm"
          value={num('humidity')}
          unit="%"
          min={0}
          max={100}
          color="#0284c7"
          icon={<WaterDropIcon fontSize="small" />}
        />
        <GaugeCard
          label="Khói / khí gas"
          value={smokeVal}
          unit="ppm"
          min={0}
          max={600}
          color="#6d4c41"
          danger={smokeVal != null && smokeVal >= 300}
          icon={<LocalFireDepartmentIcon fontSize="small" />}
        />
      </Box>

      {/* Device control card (admin / teacher only) */}
      {canControl && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="h6" mb={2}>
              Điều khiển thiết bị (A101)
            </Typography>
            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3}>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <Stack direction="row" spacing={1} alignItems="center">
                  <LightbulbIcon sx={{ color: light ? '#f59e0b' : 'text.disabled' }} />
                  <Typography>Đèn LED</Typography>
                </Stack>
                <Stack direction="row" spacing={1}>
                  <Button
                    size="small"
                    variant={light ? 'contained' : 'outlined'}
                    color="warning"
                    disabled={busyCtrl === 'light'}
                    onClick={() => control('light', true)}
                  >
                    Bật
                  </Button>
                  <Button
                    size="small"
                    variant={!light ? 'contained' : 'outlined'}
                    color="inherit"
                    disabled={busyCtrl === 'light'}
                    onClick={() => control('light', false)}
                  >
                    Tắt
                  </Button>
                </Stack>
              </Box>
              <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flex: 1 }}>
                <Stack direction="row" spacing={1} alignItems="center">
                  <AcUnitIcon sx={{ color: fan ? '#0284c7' : 'text.disabled' }} />
                  <Typography>Quạt</Typography>
                </Stack>
                <Stack direction="row" spacing={1}>
                  <Button
                    size="small"
                    variant={fan ? 'contained' : 'outlined'}
                    color="info"
                    disabled={busyCtrl === 'fan'}
                    onClick={() => control('fan', true)}
                  >
                    Bật
                  </Button>
                  <Button
                    size="small"
                    variant={!fan ? 'contained' : 'outlined'}
                    color="inherit"
                    disabled={busyCtrl === 'fan'}
                    onClick={() => control('fan', false)}
                  >
                    Tắt
                  </Button>
                </Stack>
              </Box>
            </Stack>
          </CardContent>
        </Card>
      )}

      {/* Registered devices table */}
      <Card>
        <CardContent>
          <Stack
            direction="row"
            alignItems="center"
            justifyContent="space-between"
            mb={2}
            flexWrap="wrap"
            gap={1}
          >
            <Typography variant="h6">Danh sách thiết bị đã đăng ký</Typography>
            <Stack direction="row" spacing={1}>
              <Tooltip title="Tải lại">
                <span>
                  <IconButton onClick={load} disabled={loading}>
                    <RefreshIcon />
                  </IconButton>
                </span>
              </Tooltip>
              {isAdmin && (
                <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
                  Thêm thiết bị
                </Button>
              )}
            </Stack>
          </Stack>

          {loading ? (
            <Stack spacing={1}>
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} variant="rounded" height={44} />
              ))}
            </Stack>
          ) : error ? (
            <Alert severity="error" action={<Button color="inherit" size="small" onClick={load}>Thử lại</Button>}>
              {error}
            </Alert>
          ) : devices.length === 0 ? (
            <EmptyState
              icon={<DevicesOtherIcon />}
              title="Chưa có thiết bị nào"
              description="Chưa có thiết bị nào được đăng ký trong hệ thống."
              action={
                isAdmin ? (
                  <Button variant="contained" startIcon={<AddIcon />} onClick={openCreate}>
                    Thêm thiết bị
                  </Button>
                ) : undefined
              }
              dense
            />
          ) : (
            <TableContainer component={Paper} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow>
                    <TableCell>Tên thiết bị</TableCell>
                    <TableCell>Mã</TableCell>
                    <TableCell>Loại</TableCell>
                    <TableCell>Vị trí</TableCell>
                    <TableCell>Trạng thái</TableCell>
                    <TableCell>Cập nhật</TableCell>
                    {isAdmin && <TableCell align="right">Thao tác</TableCell>}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {devices.map((d) => {
                    const active = String(d.status || '').toLowerCase() === 'active'
                    return (
                      <TableRow key={d.device_id} hover>
                        <TableCell>{d.device_name || '--'}</TableCell>
                        <TableCell>{d.device_id}</TableCell>
                        <TableCell>{d.device_type || '--'}</TableCell>
                        <TableCell>{d.location || '--'}</TableCell>
                        <TableCell>
                          <Chip
                            size="small"
                            label={d.status || '--'}
                            color={active ? 'success' : 'default'}
                            variant={active ? 'filled' : 'outlined'}
                          />
                        </TableCell>
                        <TableCell>{fmtTs(d.timestamp)}</TableCell>
                        {isAdmin && (
                          <TableCell align="right">
                            <Tooltip title="Sửa">
                              <IconButton size="small" onClick={() => openEdit(d)}>
                                <EditIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
                            <Tooltip title="Xoá">
                              <IconButton size="small" color="error" onClick={() => handleDelete(d)}>
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Tooltip>
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

      {/* Create / Edit dialog (admin only) */}
      <Dialog open={dialogOpen} onClose={closeDialog} fullWidth maxWidth="sm">
        <DialogTitle>{editingId == null ? 'Thêm thiết bị' : 'Sửa thiết bị'}</DialogTitle>
        <DialogContent>
          <Stack spacing={2} sx={{ mt: 1 }}>
            <TextField
              label="Mã thiết bị (device_id)"
              value={form.device_id}
              onChange={setField('device_id')}
              fullWidth
              required
              disabled={editingId != null}
            />
            <TextField
              label="Tên thiết bị"
              value={form.device_name}
              onChange={setField('device_name')}
              fullWidth
              required
            />
            <TextField
              label="Loại thiết bị"
              value={form.device_type}
              onChange={setField('device_type')}
              fullWidth
              placeholder="vd: temperature, light, smoke"
            />
            <TextField
              label="Vị trí"
              value={form.location}
              onChange={setField('location')}
              fullWidth
            />
            <TextField
              label="Trạng thái"
              value={form.status}
              onChange={setField('status')}
              fullWidth
              placeholder="Active / Inactive"
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={closeDialog} disabled={saving}>
            Huỷ
          </Button>
          <Button variant="contained" onClick={submitForm} disabled={saving}>
            {saving ? 'Đang lưu...' : 'Lưu'}
          </Button>
        </DialogActions>
      </Dialog>

      <Snackbar
        open={!!toast}
        autoHideDuration={3500}
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
