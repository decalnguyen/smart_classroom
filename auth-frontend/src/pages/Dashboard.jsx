import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, Chip, Snackbar, Alert,
  ToggleButton, ToggleButtonGroup, Skeleton, Divider, LinearProgress,
  Select, MenuItem, FormControl, InputLabel,
} from '@mui/material'
import { useNavigate } from 'react-router-dom'
import MeetingRoomIcon from '@mui/icons-material/MeetingRoom'
import GroupsIcon from '@mui/icons-material/Groups'
import FactCheckIcon from '@mui/icons-material/FactCheck'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import SensorsIcon from '@mui/icons-material/Sensors'
import LightModeIcon from '@mui/icons-material/LightMode'
import ThermostatIcon from '@mui/icons-material/Thermostat'
import WaterDropIcon from '@mui/icons-material/WaterDrop'
import LocalFireDepartmentIcon from '@mui/icons-material/LocalFireDepartment'
import LightbulbIcon from '@mui/icons-material/Lightbulb'
import AcUnitIcon from '@mui/icons-material/AcUnit'
import FaceRetouchingNaturalIcon from '@mui/icons-material/FaceRetouchingNatural'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import GaugeCard from '../components/GaugeCard'
import useSensorStream from '../hooks/useSensorStream'
import useAttendanceStream from '../hooks/useAttendanceStream'
import { useAuth } from '../context/AuthContext'
import { useRealtime } from '../context/RealtimeContext'
import { sensorApi, statsApi } from '../api/client'
import { LIGHT_THRESHOLDS, TEMP_THRESHOLDS, HUMIDITY_THRESHOLDS, SMOKE_THRESHOLDS } from '../constants/sensorThresholds'

const SMOKE_LIMIT = 300
const TEMP_LIMIT = 50

export default function Dashboard() {
  const navigate = useNavigate()
  const { role } = useAuth()
  const canControl = role === 'admin' || role === 'teacher'
  const { events } = useAttendanceStream()
  const { notifications } = useRealtime()
  const [toast, setToast] = useState(null)
  const [light, setLight] = useState(false)
  const [fan, setFan] = useState(false)
  const [stats, setStats] = useState(null)
  const [overview, setOverview] = useState(null)
  const [room, setRoom] = useState('')

  useEffect(() => {
    let active = true
    const load = () => {
      statsApi.overview().then(({ data }) => active && setStats(data)).catch(() => {})
      statsApi.classroomsOverview().then(({ data }) => {
        if (!active) return
        const list = Array.isArray(data) ? data : []
        setOverview(list)
        setRoom((r) => r || (list[0] && list[0].classroom_name) || '')
      }).catch(() => {})
    }
    load()
    const t = setInterval(load, 8000)
    return () => { active = false; clearInterval(t) }
  }, [])

  // Reset optimistic actuator toggles when switching rooms (stale cross-room state).
  useEffect(() => { setLight(false); setFan(false) }, [room])

  // Realtime detail for the selected room.
  const { latest, status } = useSensorStream(40, room)
  const num = (m) => (latest[m] ? latest[m].value : null)
  const smokeVal = latest.smoke?.value ?? 0
  const tempVal = latest.temperature?.value ?? 0
  // Room safety verdict mirrors the overview card badge (backend calibrated
  // thresholds) so the same room can't show "An toàn" here and "Nguy hiểm" there.
  // Fall back to live limits only until the overview row is available.
  const selectedRoom = (overview || []).find((r) => r.classroom_name === room)
  const danger = selectedRoom ? !!selectedRoom.danger : (smokeVal >= SMOKE_LIMIT || tempVal >= TEMP_LIMIT)
  const rate = stats?.attendance?.rate ? Math.round(stats.attendance.rate * 100) : 0

  // Rooms that actually have a class in session right now (for honest labels).
  const activeRooms = (overview || []).filter((r) => r.current_class).length

  const activity = [
    ...events.map((e) => ({ type: 'attendance', time: e.detection_time, text: `${e.student_name} (MSSV ${e.mssv}) điểm danh tại ${e.subject || ''}`, ts: e._ts || 0 })),
    ...notifications.filter((n) => n.title === 'alert').map((n) => ({ type: 'alert', time: '', text: n.message, ts: new Date(n.created_at).getTime() || 0 })),
  ].sort((a, b) => b.ts - a.ts).slice(0, 8)

  const control = async (type, on) => {
    try {
      await sensorApi.setDeviceMode(type, `${room}-${type}`, on ? 1 : 0)
      setToast({ severity: 'success', msg: `Đã gửi lệnh ${type === 'light' ? 'đèn' : 'quạt'} (${room}): ${on ? 'BẬT' : 'TẮT'}` })
    } catch {
      setToast({ severity: 'error', msg: 'Gửi lệnh thất bại' })
    }
  }


  return (
    <Box>
      <PageHeader
        title="Tổng quan"
        subtitle="Giám sát môi trường, an toàn và điểm danh toàn bộ lớp học theo thời gian thực"
        action={<Chip label={status === 'open' ? '● Dữ liệu realtime' : 'Đang kết nối...'} color={status === 'open' ? 'success' : 'default'} variant="outlined" />}
      />

      {/* Global KPIs */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(3, 1fr)', lg: 'repeat(5, 1fr)' }, mb: 3 }}>
        {!stats ? (
          Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)
        ) : (
          <>
            <StatCard icon={<MeetingRoomIcon />} value={stats.classrooms} label="Phòng học" color="#2563eb" sub={overview !== null ? `${activeRooms} đang có tiết` : undefined} />
            <StatCard icon={<GroupsIcon />} value={stats.students} label="Học sinh" color="#0891b2" sub={`${stats.teachers} giáo viên`} />
            <StatCard icon={<FactCheckIcon />} value={stats.attendance.attended_today ?? (stats.attendance.present_today + (stats.attendance.late_today || 0))} label="Lượt có mặt hôm nay" color="#16a34a" sub={`${stats.attendance.late_today || 0} lượt muộn · ${rate}% tham gia`} onClick={() => navigate('/reports')} />
            <StatCard icon={<WarningAmberIcon />} value={stats.alerts_today} label="Cảnh báo hôm nay" color={stats.alerts_today > 0 ? '#dc2626' : '#64748b'} onClick={() => navigate('/notifications')} />
            <StatCard icon={<SensorsIcon />} value={`${stats.sensors_active}/${stats.sensors_total}`} label="Thiết bị hoạt động" color="#7c3aed" />
          </>
        )}
      </Box>

      {/* ALL-CLASSROOMS OVERVIEW GRID */}
      <Stack direction="row" alignItems="center" spacing={1} mb={1.5}>
        <Typography variant="h6">Trạng thái các lớp học</Typography>
        {overview && (
          <Chip
            size="small"
            label={role === 'admin' ? `${overview.length} phòng` : `${overview.length} phòng đang có tiết`}
            variant="outlined"
          />
        )}
      </Stack>
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)', lg: 'repeat(4, 1fr)' }, mb: 3 }}>
        {!overview ? (
          Array.from({ length: 8 }).map((_, i) => <Skeleton key={i} variant="rounded" height={150} />)
        ) : overview.length === 0 ? (
          <Alert severity="info" sx={{ gridColumn: '1 / -1' }}>
            Hiện không có tiết của bạn đang diễn ra. Bạn chỉ giám sát phòng trong khung giờ dạy/học của mình.
          </Alert>
        ) : (
          overview.map((r) => {
            const a = r.attendance || {}
            const s = r.sensors || {}
            const pct = Math.round((a.rate || 0) * 100)
            const selected = r.classroom_name === room
            return (
              <Card
                key={r.classroom_id}
                onClick={() => setRoom(r.classroom_name)}
                sx={{
                  cursor: 'pointer', borderLeft: 5, borderLeftColor: r.danger ? 'error.main' : 'success.main',
                  outline: selected ? '2px solid' : 'none', outlineColor: 'primary.main',
                  transition: 'box-shadow .15s', '&:hover': { boxShadow: 3 },
                }}
              >
                <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                  <Stack direction="row" justifyContent="space-between" alignItems="center">
                    <Typography fontWeight={800}>{r.classroom_name}</Typography>
                    <Chip size="small" label={r.danger ? '⚠ Nguy hiểm' : '✓ An toàn'} color={r.danger ? 'error' : 'success'} variant={r.danger ? 'filled' : 'outlined'} />
                  </Stack>
                  <Typography variant="caption" color="text.secondary">{r.building}</Typography>
                  {r.current_class ? (
                    <Typography variant="caption" sx={{ display: 'block', mt: 0.5, color: 'primary.main', fontWeight: 600 }}>
                      📚 {r.current_class.subject} · Tiết {r.current_class.period} · {r.current_class.time}
                      {r.current_class.teacher ? ` · ${r.current_class.teacher}` : ''}
                    </Typography>
                  ) : (
                    <Typography variant="caption" sx={{ display: 'block', mt: 0.5, color: 'text.disabled' }}>
                      Phòng trống (không có tiết)
                    </Typography>
                  )}
                  <Stack direction="row" spacing={1.5} flexWrap="wrap" sx={{ mt: 1, color: 'text.secondary' }}>
                    <Typography variant="caption">🌡 {s.temperature || '--'}°C</Typography>
                    <Typography variant="caption">💧 {s.humidity || '--'}%</Typography>
                    <Typography variant="caption">☀ {s.light || '--'}</Typography>
                    <Typography variant="caption" sx={{ color: s.smoke >= SMOKE_LIMIT ? 'error.main' : 'inherit' }}>🔥 {s.smoke || '--'}</Typography>
                  </Stack>
                  {(a.enrolled || 0) > 0 && (
                    <Box sx={{ mt: 1.5 }}>
                      <Stack direction="row" justifyContent="space-between">
                        <Typography variant="caption" color="text.secondary">
                          {r.current_class ? 'Lượt điểm danh (gộp tiết)' : 'Lượt điểm danh hôm nay'}
                        </Typography>
                        <Typography variant="caption" fontWeight={700}>{(a.present || 0) + (a.late || 0)}/{a.enrolled} · {pct}%</Typography>
                      </Stack>
                      <LinearProgress variant="determinate" value={pct} color={pct >= 75 ? 'success' : pct >= 50 ? 'warning' : 'error'} sx={{ height: 7, borderRadius: 4, mt: 0.5 }} />
                    </Box>
                  )}
                </CardContent>
              </Card>
            )
          })
        )}
      </Box>

      {/* SELECTED-ROOM DETAIL */}
      <Stack direction="row" alignItems="center" justifyContent="space-between" mb={1.5} flexWrap="wrap" gap={1}>
        <Typography variant="h6">Chi tiết phòng</Typography>
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel id="room-l">Phòng</InputLabel>
          <Select labelId="room-l" label="Phòng" value={overview ? room : ''} onChange={(e) => setRoom(e.target.value)} disabled={!overview}>
            {(overview || []).map((r) => <MenuItem key={r.classroom_id} value={r.classroom_name}>{r.classroom_name}</MenuItem>)}
          </Select>
        </FormControl>
      </Stack>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(4, 1fr)' }, mb: 3 }}>
        <GaugeCard label="Ánh sáng" value={num('light')} unit="lux" min={0} max={1000} color="#f59e0b" icon={<LightModeIcon fontSize="small" />} thresholds={LIGHT_THRESHOLDS} />
        <GaugeCard label="Nhiệt độ" value={num('temperature')} unit="°C" min={0} max={60} color="#ea580c" danger={tempVal >= TEMP_LIMIT} icon={<ThermostatIcon fontSize="small" />} thresholds={TEMP_THRESHOLDS} />
        <GaugeCard label="Độ ẩm" value={num('humidity')} unit="%" min={0} max={100} color="#0284c7" icon={<WaterDropIcon fontSize="small" />} thresholds={HUMIDITY_THRESHOLDS} />
        <GaugeCard label="Khói / khí gas" value={num('smoke')} unit="ppm" min={0} max={600} color="#6d4c41" danger={smokeVal >= SMOKE_LIMIT} icon={<LocalFireDepartmentIcon fontSize="small" />} thresholds={SMOKE_THRESHOLDS} />
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' } }}>
        <Stack spacing={2}>
          <Card sx={{ bgcolor: danger ? 'error.main' : 'success.main', color: '#fff', border: 'none' }}>
            <CardContent>
              <Typography variant="overline" sx={{ opacity: 0.9 }}>An toàn — {room}</Typography>
              <Typography variant="h5" fontWeight={800}>{danger ? '⚠️ NGUY HIỂM' : '✓ AN TOÀN'}</Typography>
              <Typography variant="body2" sx={{ opacity: 0.95 }}>{danger ? 'Vượt ngưỡng — còi báo động kích hoạt!' : 'Các thông số trong ngưỡng cho phép.'}</Typography>
            </CardContent>
          </Card>
          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>Điều khiển — {room}</Typography>
              {!canControl && <Alert severity="info" sx={{ mb: 2 }}>Chỉ giáo viên / quản trị viên có quyền điều khiển.</Alert>}
              <Stack spacing={2}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Stack direction="row" spacing={1} alignItems="center"><LightbulbIcon sx={{ color: light ? '#f59e0b' : 'text.disabled' }} /><Typography>Đèn LED</Typography></Stack>
                  <ToggleButtonGroup size="small" exclusive value={light} onChange={(_, v) => { if (v === null) return; setLight(v); control('light', v) }} disabled={!canControl}>
                    <ToggleButton value={false}>Tắt</ToggleButton><ToggleButton value={true}>Bật</ToggleButton>
                  </ToggleButtonGroup>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Stack direction="row" spacing={1} alignItems="center"><AcUnitIcon sx={{ color: fan ? '#0284c7' : 'text.disabled' }} /><Typography>Quạt</Typography></Stack>
                  <ToggleButtonGroup size="small" exclusive value={fan} onChange={(_, v) => { if (v === null) return; setFan(v); control('fan', v) }} disabled={!canControl}>
                    <ToggleButton value={false}>Tắt</ToggleButton><ToggleButton value={true}>Bật</ToggleButton>
                  </ToggleButtonGroup>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Stack>
      </Box>

      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" mb={1}>Hoạt động gần đây</Typography>
          <Divider sx={{ mb: 1 }} />
          {activity.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>Đang chờ hoạt động realtime...</Typography>
          ) : (
            <Stack divider={<Divider flexItem />} spacing={1}>
              {activity.map((a, i) => (
                <Stack key={i} direction="row" alignItems="center" spacing={1.5} sx={{ py: 0.5 }}>
                  {a.type === 'alert' ? <WarningAmberIcon color="error" fontSize="small" /> : <FaceRetouchingNaturalIcon color="success" fontSize="small" />}
                  <Typography variant="body2" sx={{ flex: 1 }}>{a.text}</Typography>
                  {a.time && <Typography variant="caption" color="text.secondary">{a.time}</Typography>}
                </Stack>
              ))}
            </Stack>
          )}
        </CardContent>
      </Card>

      <Snackbar open={!!toast} autoHideDuration={3000} onClose={() => setToast(null)} anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}>
        {toast ? <Alert severity={toast.severity} onClose={() => setToast(null)}>{toast.msg}</Alert> : null}
      </Snackbar>
    </Box>
  )
}
