import { useState, useEffect, useCallback, useMemo, useRef } from 'react'
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
  MenuItem,
  ToggleButton,
  ToggleButtonGroup,
} from '@mui/material'
import LightModeIcon from '@mui/icons-material/LightMode'
import ThermostatIcon from '@mui/icons-material/Thermostat'
import WaterDropIcon from '@mui/icons-material/WaterDrop'
import LocalFireDepartmentIcon from '@mui/icons-material/LocalFireDepartment'
import LightbulbIcon from '@mui/icons-material/Lightbulb'
import AcUnitIcon from '@mui/icons-material/AcUnit'
import NotificationsActiveIcon from '@mui/icons-material/NotificationsActive'
import AddIcon from '@mui/icons-material/Add'
import EditIcon from '@mui/icons-material/Edit'
import DeleteIcon from '@mui/icons-material/Delete'
import RefreshIcon from '@mui/icons-material/Refresh'
import DevicesOtherIcon from '@mui/icons-material/DevicesOther'
import TimelineIcon from '@mui/icons-material/Timeline'
import dayjs from 'dayjs'
import { useTheme } from '@mui/material/styles'
import {
  ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid,
  Tooltip as RTooltip, ReferenceLine,
} from 'recharts'
import PageHeader from '../components/PageHeader'
import EmptyState from '../components/EmptyState'
import GaugeCard from '../components/GaugeCard'
import useSensorStream from '../hooks/useSensorStream'
import { useAuth } from '../context/AuthContext'
import { sensorApi, apiError } from '../api/client'
import { LIGHT_THRESHOLDS, TEMP_THRESHOLDS, HUMIDITY_THRESHOLDS, SMOKE_THRESHOLDS } from '../constants/sensorThresholds'

// History-chart config per metric: device-id suffix, label, unit, color, and the
// reference lines to overlay (danger boundary in red; comfort-band edges in amber).
const HIST_METRICS = {
  temp: { suffix: 'temp', label: 'Nhiệt độ', unit: '°C', color: '#ea580c', refs: [{ y: 50, color: '#dc2626', label: 'Nguy hiểm 50°C' }] },
  humi: { suffix: 'humi', label: 'Độ ẩm', unit: '%', color: '#0284c7', refs: [{ y: 30, color: '#f59e0b' }, { y: 70, color: '#f59e0b' }] },
  light: { suffix: 'light', label: 'Ánh sáng', unit: 'lux', color: '#f59e0b', refs: [{ y: 200, color: '#94a3b8' }, { y: 750, color: '#94a3b8' }] },
  smoke: { suffix: 'smoke', label: 'Khói', unit: 'ppm', color: '#6d4c41', refs: [{ y: 300, color: '#dc2626', label: 'Nguy hiểm 300ppm' }] },
}
const HIST_RANGES = { '1h': 3600e3, '6h': 6 * 3600e3, '24h': 24 * 3600e3 }
const HIST_BUCKET = { '1h': 60e3, '6h': 300e3, '24h': 900e3 }

const EMPTY_FORM = {
  device_id: '',
  device_name: '',
  device_type: '',
  location: '',
  status: 'Active',
}

// Actuators controllable from the UI. `levels` lists the allowed command values:
// the fan has speeds 0–3; on/off devices (đèn/còi) are just 0 (off) / 1 (on).
// NOTE: the lamp uses the 'led' channel — the 'light' topic carries the LDR lux
// READING, so a 'light' actuator would collide with the sensor gauge.
const ACTUATORS = [
  { type: 'led', label: 'Đèn', icon: LightbulbIcon, color: '#f59e0b', levels: [0, 1] },
  { type: 'fan', label: 'Quạt', icon: AcUnitIcon, color: '#0284c7', levels: [0, 1, 2, 3] },
  { type: 'buzzer', label: 'Còi', icon: NotificationsActiveIcon, color: '#dc2626', levels: [0, 1] },
]

// Label for a level button: 0 → "Tắt"; on/off device 1 → "Bật"; fan → the number.
function levelLabel(actuator, n) {
  if (n === 0) return 'Tắt'
  return actuator.levels.length === 2 ? 'Bật' : n
}

function fmtTs(ts) {
  if (!ts) return '--'
  const d = dayjs(ts)
  return d.isValid() ? d.format('DD/MM/YYYY HH:mm:ss') : String(ts)
}

export default function Sensors() {
  const { role } = useAuth()
  const theme = useTheme()
  const isAdmin = role === 'admin'
  const canControl = role === 'admin' || role === 'teacher'

  // Empty until devices load — the pickedInitialRoom effect selects the most
  // recently active room, avoiding a race that subscribes to a dead room.
  const [room, setRoom] = useState('')
  const { latest, status } = useSensorStream(40, room)
  const online = status === 'open'

  // History trend chart state (one metric + time range at a time).
  const [histMetric, setHistMetric] = useState('temp')
  const [histRange, setHistRange] = useState('1h')
  const [histRows, setHistRows] = useState([])
  const [histLoading, setHistLoading] = useState(false)

  const [devices, setDevices] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  const [toast, setToast] = useState(null)

  // Dialog state for create/edit.
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingId, setEditingId] = useState(null) // null = create mode
  const [form, setForm] = useState(EMPTY_FORM)
  const [saving, setSaving] = useState(false)

  // Actuator command levels (0–3) per device type (optimistic, reset per room).
  const [levels, setLevels] = useState({ fan: 0, led: 0, buzzer: 0 })
  const [busyCtrl, setBusyCtrl] = useState(null)
  useEffect(() => { setLevels({ fan: 0, led: 0, buzzer: 0 }) }, [room])

  // Room list derived from the (role-scoped) registered devices. Admin sees all
  // rooms; a teacher/student only sees rooms in their teaching/study schedule.
  const rooms = useMemo(() => {
    const set = new Set()
    devices.forEach((d) => {
      const r = (d.device_id || '').split('-')[0]
      if (r) set.add(r)
    })
    return [...set].sort()
  }, [devices])

  // Default to the room with the most-recent activity (by registry heartbeat),
  // not a hardcoded A101 — so the first view shows a live room, never a dead one.
  const pickedInitialRoom = useRef(false)
  useEffect(() => {
    if (pickedInitialRoom.current || devices.length === 0) return
    let best = null
    let bestTs = -1
    devices.forEach((d) => {
      const r = (d.device_id || '').split('-')[0]
      const ts = d.timestamp ? new Date(d.timestamp).getTime() : 0
      if (r && ts > bestTs) {
        bestTs = ts
        best = r
      }
    })
    if (best) {
      setRoom(best)
      pickedInitialRoom.current = true
    }
  }, [devices])

  // Latest numeric value for a metric, or null when not yet received.
  const num = (m) => {
    const v = latest[m] ? latest[m].value : null
    return v === null || v === undefined || Number.isNaN(Number(v)) ? null : Number(v)
  }

  // Actuator level the DEVICE itself reports (echoed on /<room>/<type>/value),
  // scoped to the selected room and clamped to 0–3 (ignores e.g. the LDR lux on
  // the "light" topic). null when the device hasn't reported a valid level.
  const liveLevel = (type) => {
    const e = latest[type]
    if (e && e.device_id === `${room}-${type}`) {
      const v = Number(e.value)
      if (Number.isInteger(v) && v >= 0 && v <= 3) return v
    }
    return null
  }

  // Keep the selector in sync with what the device actually reports — so a
  // fire-alarm buzzer (or any external change) shows up AND the operator can
  // turn it off. User clicks set it optimistically; echoes reconcile it.
  useEffect(() => {
    setLevels((prev) => {
      let next = null
      for (const a of ACTUATORS) {
        const e = latest[a.type]
        if (e && e.device_id === `${room}-${a.type}`) {
          const v = Number(e.value)
          if (Number.isInteger(v) && v >= 0 && v <= 3 && v !== prev[a.type]) {
            if (!next) next = { ...prev }
            next[a.type] = v
          }
        }
      }
      return next || prev
    })
  }, [latest, room])

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

  // Fetch + time-bucket the history for the selected room/metric/range.
  useEffect(() => {
    if (!room) { setHistRows([]); return }
    let active = true
    const m = HIST_METRICS[histMetric]
    const rangeMs = HIST_RANGES[histRange]
    const bucketMs = HIST_BUCKET[histRange]
    setHistLoading(true)
    ;(async () => {
      try {
        const start = new Date(Date.now() - rangeMs).toISOString()
        const end = new Date().toISOString()
        const { data } = await sensorApi.history(`${room}-${m.suffix}`, start, end)
        const list = Array.isArray(data) ? data : []
        const agg = new Map()
        for (const p of list) {
          const t = Math.floor(new Date(p.timestamp).getTime() / bucketMs) * bucketMs
          const a = agg.get(t) || { sum: 0, n: 0 }
          a.sum += Number(p.value); a.n += 1
          agg.set(t, a)
        }
        const rows = [...agg.entries()].map(([t, a]) => ({ t, v: a.sum / a.n })).sort((x, y) => x.t - y.t)
        if (active) setHistRows(rows)
      } catch {
        if (active) setHistRows([])
      } finally {
        if (active) setHistLoading(false)
      }
    })()
    return () => { active = false }
  }, [room, histMetric, histRange])

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

  // Send an actuator command at the given level (0–3) to the selected room.
  const control = async (type, level) => {
    setBusyCtrl(type)
    try {
      await sensorApi.setDeviceMode(type, `${room}-${type}`, level)
      setLevels((prev) => ({ ...prev, [type]: level }))
      const label = ACTUATORS.find((a) => a.type === type)?.label || type
      setToast({
        severity: 'success',
        msg: level === 0 ? `Đã tắt ${label}` : `Đã đặt ${label} mức ${level}`,
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
        subtitle={isAdmin ? 'Theo dõi và điều khiển thiết bị trong lớp' : 'Chỉ hiển thị phòng & khung giờ theo lịch dạy/học của bạn'}
        action={
          <Stack direction="row" alignItems="center" spacing={1.5}>
            <TextField
              select size="small" label="Phòng" value={room}
              onChange={(e) => setRoom(e.target.value)} sx={{ minWidth: 120 }}
            >
              {rooms.map((r) => (
                <MenuItem key={r} value={r}>{r}</MenuItem>
              ))}
            </TextField>
            <Chip
              label={online ? 'Trực tuyến' : 'Ngoại tuyến'}
              color={online ? 'success' : 'default'}
              variant={online ? 'filled' : 'outlined'}
            />
          </Stack>
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
          thresholds={LIGHT_THRESHOLDS}
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
          thresholds={TEMP_THRESHOLDS}
        />
        <GaugeCard
          label="Độ ẩm"
          value={num('humidity')}
          unit="%"
          min={0}
          max={100}
          color="#0284c7"
          icon={<WaterDropIcon fontSize="small" />}
          thresholds={HUMIDITY_THRESHOLDS}
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
          thresholds={SMOKE_THRESHOLDS}
        />
      </Box>

      {/* History trend — how a reading evolved over time (gauges show only "now") */}
      {room && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Stack direction="row" alignItems="center" justifyContent="space-between" flexWrap="wrap" gap={1} mb={1}>
              <Stack direction="row" alignItems="center" spacing={1}>
                <TimelineIcon color="primary" />
                <Typography variant="h6">Diễn biến theo thời gian — {room}</Typography>
              </Stack>
              <Stack direction="row" spacing={1} flexWrap="wrap">
                <ToggleButtonGroup size="small" exclusive value={histMetric} onChange={(_, v) => v && setHistMetric(v)}>
                  {Object.entries(HIST_METRICS).map(([k, m]) => (
                    <ToggleButton key={k} value={k}>{m.label}</ToggleButton>
                  ))}
                </ToggleButtonGroup>
                <ToggleButtonGroup size="small" exclusive value={histRange} onChange={(_, v) => v && setHistRange(v)}>
                  <ToggleButton value="1h">1 giờ</ToggleButton>
                  <ToggleButton value="6h">6 giờ</ToggleButton>
                  <ToggleButton value="24h">24 giờ</ToggleButton>
                </ToggleButtonGroup>
              </Stack>
            </Stack>
            <Box sx={{ height: 300 }}>
              {histLoading ? (
                <Skeleton variant="rounded" height={280} />
              ) : histRows.length === 0 ? (
                <EmptyState dense icon={<TimelineIcon />} title="Chưa có dữ liệu lịch sử" description="Phòng này chưa có dữ liệu cảm biến trong khoảng thời gian đã chọn." />
              ) : (
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={histRows} margin={{ top: 8, right: 12, bottom: 0, left: -8 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'} />
                    <XAxis dataKey="t" type="number" domain={['dataMin', 'dataMax']} scale="time"
                      tickFormatter={(t) => dayjs(t).format(histRange === '24h' ? 'DD/MM HH:mm' : 'HH:mm')}
                      tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                    <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} unit={HIST_METRICS[histMetric].unit} width={56} />
                    <RTooltip
                      labelFormatter={(t) => dayjs(t).format('DD/MM/YYYY HH:mm')}
                      formatter={(v) => [`${Number(v).toFixed(1)} ${HIST_METRICS[histMetric].unit}`, HIST_METRICS[histMetric].label]}
                      contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                    {HIST_METRICS[histMetric].refs.map((r, i) => (
                      <ReferenceLine key={i} y={r.y} stroke={r.color} strokeDasharray="4 4" label={r.label ? { value: r.label, fontSize: 10, fill: r.color, position: 'insideTopRight' } : undefined} />
                    ))}
                    <Line dataKey="v" name={HIST_METRICS[histMetric].label} stroke={HIST_METRICS[histMetric].color} strokeWidth={2} dot={false} isAnimationActive={false} />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </Box>
          </CardContent>
        </Card>
      )}

      {/* Device control card (admin / teacher only) — command level 0–3 */}
      {canControl && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Box>
              <Typography variant="h6">Điều khiển thiết bị — {room}</Typography>
              <Typography variant="caption" color="text.secondary" display="block">
                Chọn mức 0–3 (0 = tắt; quạt: 1–3 là tốc độ) → gửi lệnh MQTT tới thiết bị
              </Typography>
              <Typography variant="caption" color="text.secondary" display="block">
                Tự động tắt đèn/quạt khi phòng không có tiết (theo thời khóa biểu).
              </Typography>
            </Box>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
                gap: 2,
                mt: 2,
              }}
            >
              {ACTUATORS.map((a) => {
                const Icon = a.icon
                const sel = levels[a.type]            // user selection (immediate feedback)
                const live = liveLevel(a.type)        // device-reported state (read-only)
                const liveText = a.levels.length === 2 ? (live > 0 ? 'Bật' : 'Tắt') : live
                return (
                  <Box
                    key={a.type}
                    sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 1 }}
                  >
                    <Stack direction="row" spacing={1} alignItems="center">
                      <Icon sx={{ color: sel > 0 ? a.color : 'text.disabled' }} />
                      <Box>
                        <Typography>{a.label}</Typography>
                        {live != null && (
                          <Typography variant="caption" color="success.main">● thiết bị: {liveText}</Typography>
                        )}
                      </Box>
                    </Stack>
                    <ToggleButtonGroup
                      size="small"
                      exclusive
                      value={sel}
                      disabled={busyCtrl === a.type}
                      onChange={(_, v) => v != null && control(a.type, v)}
                    >
                      {a.levels.map((n) => (
                        <ToggleButton key={n} value={n} sx={{ minWidth: 44 }}>
                          {levelLabel(a, n)}
                        </ToggleButton>
                      ))}
                    </ToggleButtonGroup>
                  </Box>
                )
              })}
            </Box>
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
          ) : (() => {
            // Group devices by room (location field, fallback to device_id prefix)
            const grouped = {}
            devices.forEach((d) => {
              const loc = d.location || (d.device_id || '').split('-')[0] || 'Khác'
              if (!grouped[loc]) grouped[loc] = []
              grouped[loc].push(d)
            })
            const sortedRooms = Object.keys(grouped).sort()
            return (
              <Stack spacing={2}>
                {sortedRooms.map((loc) => (
                  <Box key={loc}>
                    <Stack direction="row" alignItems="center" spacing={1} mb={0.5}>
                      <Typography variant="subtitle2" fontWeight={700} color="primary">{loc}</Typography>
                      <Chip size="small" label={`${grouped[loc].length} thiết bị`} variant="outlined" />
                    </Stack>
                    <TableContainer component={Paper} variant="outlined">
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>Tên thiết bị</TableCell>
                            <TableCell>Mã</TableCell>
                            <TableCell>Loại</TableCell>
                            <TableCell>Trạng thái</TableCell>
                            <TableCell>Cập nhật</TableCell>
                            {isAdmin && <TableCell align="right">Thao tác</TableCell>}
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {grouped[loc].map((d) => {
                            const active = String(d.status || '').toLowerCase() === 'active'
                            return (
                              <TableRow key={d.device_id} hover>
                                <TableCell>{d.device_name || '--'}</TableCell>
                                <TableCell>{d.device_id}</TableCell>
                                <TableCell>{d.device_type || '--'}</TableCell>
                                <TableCell>
                                  <Chip size="small" label={d.status || '--'}
                                    color={active ? 'success' : 'default'}
                                    variant={active ? 'filled' : 'outlined'} />
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
                  </Box>
                ))}
              </Stack>
            )
          })()}
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
