import { useState, useEffect } from 'react'
import {
  Box,
  Card,
  CardContent,
  Typography,
  Stack,
  Chip,
  Snackbar,
  Alert,
  ToggleButton,
  ToggleButtonGroup,
  Skeleton,
  Divider,
} from '@mui/material'
import { useTheme } from '@mui/material/styles'
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
import { ResponsiveContainer, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip as RTooltip, Legend } from 'recharts'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import GaugeCard from '../components/GaugeCard'
import useSensorStream from '../hooks/useSensorStream'
import useAttendanceStream from '../hooks/useAttendanceStream'
import { useAuth } from '../context/AuthContext'
import { useRealtime } from '../context/RealtimeContext'
import { sensorApi, statsApi } from '../api/client'

const SMOKE_LIMIT = 300
const TEMP_LIMIT = 50

export default function Dashboard() {
  const theme = useTheme()
  const navigate = useNavigate()
  const { role } = useAuth()
  const canControl = role === 'admin' || role === 'teacher'
  const { latest, points, status } = useSensorStream()
  const { events } = useAttendanceStream()
  const { notifications } = useRealtime()
  const [toast, setToast] = useState(null)
  const [light, setLight] = useState(false)
  const [fan, setFan] = useState(false)
  const [stats, setStats] = useState(null)

  useEffect(() => {
    let active = true
    const load = () => statsApi.overview().then(({ data }) => active && setStats(data)).catch(() => {})
    load()
    const t = setInterval(load, 15000)
    return () => { active = false; clearInterval(t) }
  }, [])

  const num = (m) => (latest[m] ? latest[m].value : null)
  const smokeVal = latest.smoke?.value ?? 0
  const tempVal = latest.temperature?.value ?? 0
  const danger = smokeVal >= SMOKE_LIMIT || tempVal >= TEMP_LIMIT
  const rate = stats?.attendance?.rate ? Math.round(stats.attendance.rate * 100) : 0

  // Merge recent recognitions + alerts into one activity feed.
  const activity = [
    ...events.map((e) => ({ type: 'attendance', time: e.detection_time, text: `${e.student_name} (MSSV ${e.mssv}) điểm danh`, ts: e._ts || 0 })),
    ...notifications.filter((n) => n.title === 'alert').map((n) => ({ type: 'alert', time: '', text: n.message, ts: new Date(n.created_at).getTime() || 0 })),
  ].sort((a, b) => b.ts - a.ts).slice(0, 8)

  const control = async (type, on) => {
    try {
      await sensorApi.setDeviceMode(type, `A101-${type}`, on ? 1 : 0)
      setToast({ severity: 'success', msg: `Đã gửi lệnh ${type === 'light' ? 'đèn' : 'quạt'}: ${on ? 'BẬT' : 'TẮT'}` })
    } catch {
      setToast({ severity: 'error', msg: 'Gửi lệnh thất bại' })
    }
  }

  const grid = theme.palette.mode === 'dark' ? 'rgba(148,163,184,0.15)' : '#eef2f7'

  return (
    <Box>
      <PageHeader
        title="Tổng quan"
        subtitle="Giám sát môi trường, an toàn và điểm danh theo thời gian thực"
        action={<Chip label={status === 'open' ? '● Dữ liệu realtime' : 'Đang kết nối...'} color={status === 'open' ? 'success' : 'default'} variant="outlined" />}
      />

      {/* KPI row */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(3, 1fr)', lg: 'repeat(5, 1fr)' }, mb: 3 }}>
        {!stats ? (
          Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)
        ) : (
          <>
            <StatCard icon={<MeetingRoomIcon />} value={stats.classrooms} label="Phòng học" color="#2563eb" onClick={() => navigate('/sensors')} />
            <StatCard icon={<GroupsIcon />} value={stats.students} label="Học sinh" color="#0891b2" sub={`${stats.teachers} giáo viên`} />
            <StatCard icon={<FactCheckIcon />} value={stats.attendance.present_today} label="Điểm danh hôm nay" color="#16a34a" sub={`Tỉ lệ ${rate}%`} onClick={() => navigate('/attendance')} />
            <StatCard icon={<WarningAmberIcon />} value={stats.alerts_today} label="Cảnh báo hôm nay" color={stats.alerts_today > 0 ? '#dc2626' : '#64748b'} onClick={() => navigate('/notifications')} />
            <StatCard icon={<SensorsIcon />} value={`${stats.sensors_active}/${stats.sensors_total}`} label="Thiết bị hoạt động" color="#7c3aed" />
          </>
        )}
      </Box>

      {/* Gauges */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(4, 1fr)' }, mb: 3 }}>
        <GaugeCard label="Ánh sáng" value={num('light')} unit="lux" min={0} max={1000} color="#f59e0b" icon={<LightModeIcon fontSize="small" />} />
        <GaugeCard label="Nhiệt độ" value={num('temperature')} unit="°C" min={0} max={60} color="#ea580c" danger={tempVal >= TEMP_LIMIT} icon={<ThermostatIcon fontSize="small" />} />
        <GaugeCard label="Độ ẩm" value={num('humidity')} unit="%" min={0} max={100} color="#0284c7" icon={<WaterDropIcon fontSize="small" />} />
        <GaugeCard label="Khói / khí gas" value={num('smoke')} unit="ppm" min={0} max={600} color="#6d4c41" danger={smokeVal >= SMOKE_LIMIT} icon={<LocalFireDepartmentIcon fontSize="small" />} />
      </Box>

      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', lg: '2fr 1fr' } }}>
        {/* Realtime chart */}
        <Card>
          <CardContent>
            <Typography variant="h6" mb={1}>Biểu đồ cảm biến thời gian thực</Typography>
            <Box sx={{ height: 320 }}>
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={points} margin={{ top: 8, right: 16, bottom: 0, left: -12 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={grid} />
                  <XAxis dataKey="time" tick={{ fontSize: 11, fill: theme.palette.text.secondary }} minTickGap={28} />
                  <YAxis tick={{ fontSize: 11, fill: theme.palette.text.secondary }} />
                  <RTooltip contentStyle={{ background: theme.palette.background.paper, border: `1px solid ${theme.palette.divider}`, borderRadius: 8 }} />
                  <Legend />
                  <Line type="monotone" dataKey="temperature" name="Nhiệt độ (°C)" stroke="#ea580c" dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="humidity" name="Độ ẩm (%)" stroke="#0284c7" dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="smoke" name="Khói" stroke="#dc2626" dot={false} isAnimationActive={false} />
                  <Line type="monotone" dataKey="light" name="Ánh sáng (lux)" stroke="#f59e0b" dot={false} isAnimationActive={false} />
                </LineChart>
              </ResponsiveContainer>
            </Box>
          </CardContent>
        </Card>

        <Stack spacing={2}>
          {/* Safety status */}
          <Card sx={{ bgcolor: danger ? 'error.main' : 'success.main', color: '#fff', border: 'none' }}>
            <CardContent>
              <Typography variant="overline" sx={{ opacity: 0.9 }}>Trạng thái an toàn</Typography>
              <Typography variant="h5" fontWeight={800}>{danger ? '⚠️ NGUY HIỂM' : '✓ AN TOÀN'}</Typography>
              <Typography variant="body2" sx={{ opacity: 0.95 }}>
                {danger ? 'Thông số vượt ngưỡng — còi báo động đã kích hoạt!' : 'Các thông số trong ngưỡng cho phép.'}
              </Typography>
            </CardContent>
          </Card>

          {/* Device controls */}
          <Card>
            <CardContent>
              <Typography variant="h6" mb={2}>Điều khiển thiết bị</Typography>
              {!canControl && <Alert severity="info" sx={{ mb: 2 }}>Chỉ giáo viên / quản trị viên có quyền điều khiển.</Alert>}
              <Stack spacing={2}>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <LightbulbIcon sx={{ color: light ? '#f59e0b' : 'text.disabled' }} />
                    <Typography>Đèn LED</Typography>
                  </Stack>
                  <ToggleButtonGroup size="small" exclusive value={light} onChange={(_, v) => { if (v === null) return; setLight(v); control('light', v) }} disabled={!canControl}>
                    <ToggleButton value={false}>Tắt</ToggleButton>
                    <ToggleButton value={true}>Bật</ToggleButton>
                  </ToggleButtonGroup>
                </Box>
                <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <AcUnitIcon sx={{ color: fan ? '#0284c7' : 'text.disabled' }} />
                    <Typography>Quạt</Typography>
                  </Stack>
                  <ToggleButtonGroup size="small" exclusive value={fan} onChange={(_, v) => { if (v === null) return; setFan(v); control('fan', v) }} disabled={!canControl}>
                    <ToggleButton value={false}>Tắt</ToggleButton>
                    <ToggleButton value={true}>Bật</ToggleButton>
                  </ToggleButtonGroup>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Stack>
      </Box>

      {/* Recent activity */}
      <Card sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" mb={1}>Hoạt động gần đây</Typography>
          <Divider sx={{ mb: 1 }} />
          {activity.length === 0 ? (
            <Typography variant="body2" color="text.secondary" sx={{ py: 2 }}>Chưa có hoạt động.</Typography>
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
