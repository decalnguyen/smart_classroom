import { useState, useEffect } from 'react'
import {
  Box, Card, CardContent, Typography, Stack, Chip, Skeleton, Divider, LinearProgress, Alert,
} from '@mui/material'
import { useNavigate } from 'react-router-dom'
import MeetingRoomIcon from '@mui/icons-material/MeetingRoom'
import GroupsIcon from '@mui/icons-material/Groups'
import FactCheckIcon from '@mui/icons-material/FactCheck'
import WarningAmberIcon from '@mui/icons-material/WarningAmber'
import SensorsIcon from '@mui/icons-material/Sensors'
import FaceRetouchingNaturalIcon from '@mui/icons-material/FaceRetouchingNatural'
import RadioButtonCheckedIcon from '@mui/icons-material/RadioButtonChecked'
import HistoryIcon from '@mui/icons-material/History'
import PageHeader from '../components/PageHeader'
import StatCard from '../components/StatCard'
import useAttendanceStream from '../hooks/useAttendanceStream'
import { useRealtime } from '../context/RealtimeContext'
import { statsApi } from '../api/client'

// One class-session of today's timeline: room · subject · tiết · time · GV + attendance.
function ClassCard({ c }) {
  const attended = (c.present || 0) + (c.late || 0)
  const pct = Math.round((c.rate || 0) * 100)
  const ongoing = c.status === 'ongoing'
  return (
    <Card sx={{ borderLeft: 5, borderLeftColor: ongoing ? 'success.main' : 'grey.400' }}>
      <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography fontWeight={800}>{c.classroom_name}</Typography>
          <Chip size="small" color={ongoing ? 'success' : 'default'} variant={ongoing ? 'filled' : 'outlined'}
            label={ongoing ? '● Đang diễn ra' : 'Đã kết thúc'} />
        </Stack>
        <Typography variant="caption" sx={{ display: 'block', mt: 0.5, color: 'primary.main', fontWeight: 600 }}>
          📚 {c.subject} · Tiết {c.period} · {c.time}{c.teacher_name ? ` · ${c.teacher_name}` : ''}
        </Typography>
        <Stack direction="row" spacing={0.5} flexWrap="wrap" useFlexGap sx={{ mt: 1 }}>
          <Chip size="small" color="success" variant="outlined" label={`Có mặt ${c.present || 0}`} />
          <Chip size="small" color="warning" variant="outlined" label={`Muộn ${c.late || 0}`} />
          <Chip size="small" color="info" variant="outlined" label={`Phép ${c.excused || 0}`} />
          <Chip size="small" color="error" variant="outlined" label={`Vắng ${c.absent || 0}`} />
        </Stack>
        <Stack direction="row" justifyContent="space-between" sx={{ mt: 1 }}>
          <Typography variant="caption" color="text.secondary">Sĩ số {c.enrolled}</Typography>
          <Typography variant="caption" fontWeight={700}>{attended}/{c.enrolled} · {pct}%</Typography>
        </Stack>
        <LinearProgress variant="determinate" value={pct} color={pct >= 75 ? 'success' : pct >= 50 ? 'warning' : 'error'} sx={{ height: 7, borderRadius: 4, mt: 0.5 }} />
      </CardContent>
    </Card>
  )
}

export default function Dashboard() {
  const navigate = useNavigate()
  const { events, status } = useAttendanceStream()
  const { notifications } = useRealtime()
  const [stats, setStats] = useState(null)
  const [classes, setClasses] = useState(null) // { ongoing, ended, as_of }

  useEffect(() => {
    let active = true
    const load = () => {
      statsApi.overview().then(({ data }) => active && setStats(data)).catch(() => {})
      statsApi.classesToday().then(({ data }) => active && setClasses(data)).catch(() => {})
    }
    load()
    const t = setInterval(load, 8000)
    return () => { active = false; clearInterval(t) }
  }, [])

  const rate = stats?.attendance?.rate ? Math.round(stats.attendance.rate * 100) : 0
  const ongoing = classes?.ongoing || []
  const ended = classes?.ended || []

  const activity = [
    ...events.map((e) => ({ type: 'attendance', time: e.detection_time, text: `${e.student_name} (MSSV ${e.mssv}) điểm danh · ${e.subject || ''}`, ts: e._ts || 0 })),
    ...notifications.filter((n) => n.title === 'alert').map((n) => ({ type: 'alert', time: '', text: n.message, ts: new Date(n.created_at).getTime() || 0 })),
  ].sort((a, b) => b.ts - a.ts).slice(0, 8)

  const GRID = { display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)', lg: 'repeat(4, 1fr)' }, mb: 3 }

  return (
    <Box>
      <PageHeader
        title="Tổng quan"
        subtitle="Theo dõi lớp học & điểm danh theo thời gian thực"
        action={<Chip label={status === 'open' ? '● Dữ liệu realtime' : 'Đang kết nối...'} color={status === 'open' ? 'success' : 'default'} variant="outlined" />}
      />

      {/* Global KPIs */}
      <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr 1fr', md: 'repeat(3, 1fr)', lg: 'repeat(5, 1fr)' }, mb: 3 }}>
        {!stats ? (
          Array.from({ length: 5 }).map((_, i) => <Skeleton key={i} variant="rounded" height={92} />)
        ) : (
          <>
            <StatCard icon={<MeetingRoomIcon />} value={stats.classrooms} label="Phòng học" color="#2563eb" sub={classes ? `${ongoing.length} lớp đang diễn ra` : undefined} />
            <StatCard icon={<GroupsIcon />} value={stats.students} label="Học sinh" color="#0891b2" sub={`${stats.teachers} giáo viên`} />
            <StatCard icon={<FactCheckIcon />} value={stats.attendance.attended_today ?? (stats.attendance.present_today + (stats.attendance.late_today || 0))} label="Lượt có mặt hôm nay" color="#16a34a" sub={`${stats.attendance.late_today || 0} lượt muộn · ${rate}% tham gia`} onClick={() => navigate('/reports')} />
            <StatCard icon={<WarningAmberIcon />} value={stats.alerts_today} label="Cảnh báo hôm nay" color={stats.alerts_today > 0 ? '#dc2626' : '#64748b'} onClick={() => navigate('/notifications')} />
            <StatCard icon={<SensorsIcon />} value={`${stats.sensors_active}/${stats.sensors_total}`} label="Thiết bị hoạt động" color="#7c3aed" onClick={() => navigate('/sensors')} />
          </>
        )}
      </Box>

      {/* ONGOING classes (right now) */}
      <Stack direction="row" alignItems="center" spacing={1} mb={1.5}>
        <RadioButtonCheckedIcon color="success" fontSize="small" />
        <Typography variant="h6">Lớp đang diễn ra</Typography>
        {classes && <Chip size="small" label={ongoing.length} color={ongoing.length ? 'success' : 'default'} variant="outlined" />}
      </Stack>
      <Box sx={GRID}>
        {!classes ? (
          Array.from({ length: 3 }).map((_, i) => <Skeleton key={i} variant="rounded" height={160} />)
        ) : ongoing.length === 0 ? (
          <Alert severity="info" sx={{ gridColumn: '1 / -1' }}>Hiện không có lớp nào đang diễn ra.</Alert>
        ) : ongoing.map((c) => <ClassCard key={c.class_id} c={c} />)}
      </Box>

      {/* ENDED classes earlier today */}
      <Stack direction="row" alignItems="center" spacing={1} mb={1.5}>
        <HistoryIcon color="action" fontSize="small" />
        <Typography variant="h6">Lớp đã diễn ra hôm nay</Typography>
        {classes && <Chip size="small" label={ended.length} variant="outlined" />}
        {classes?.as_of && <Typography variant="caption" color="text.secondary">· tính tới {classes.as_of}</Typography>}
      </Stack>
      <Box sx={GRID}>
        {!classes ? (
          Array.from({ length: 4 }).map((_, i) => <Skeleton key={i} variant="rounded" height={160} />)
        ) : ended.length === 0 ? (
          <Alert severity="info" sx={{ gridColumn: '1 / -1' }}>Chưa có lớp nào kết thúc hôm nay.</Alert>
        ) : ended.map((c) => <ClassCard key={c.class_id} c={c} />)}
      </Box>

      {/* Recent realtime activity (attendance + safety alerts) */}
      <Card>
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
    </Box>
  )
}
